package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestValidateProviderID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"openai", false},
		{"aleph_alpha", false},
		{"ai21", false},
		{"amazon-nova", false},
		{"../../../../etc/passwd", true},
		{"/etc/shadow", true},
		{"etc/shadow", true},
		{"..", true},
		{"", true},
		{"Openai", true},
		{"_leading", true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if err := validateProviderID(tt.id); (err != nil) != tt.wantErr {
				t.Fatalf("validateProviderID(%q) err = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

// The signing identity must be pinned to main exactly; both signing workflows
// allow workflow_dispatch, so an unanchored pattern would trust main-* branches.
func TestCertIdentityRegexpAnchoredToMain(t *testing.T) {
	re := regexp.MustCompile(certIdentityRegexp)

	const base = "https://github.com/ferro-labs/model-catalog/.github/workflows/"
	accept := []string{base + "pages.yml@refs/heads/main", base + "release.yml@refs/heads/main"}
	reject := []string{
		base + "pages.yml@refs/heads/main-attacker",
		base + "release.yml@refs/heads/maintenance",
		base + "scrape-weekly.yml@refs/heads/main",
		"https://github.com/evil/model-catalog/.github/workflows/pages.yml@refs/heads/main",
	}

	for _, id := range accept {
		if !re.MatchString(id) {
			t.Errorf("expected match for %q", id)
		}
	}
	for _, id := range reject {
		if re.MatchString(id) {
			t.Errorf("expected NO match for %q", id)
		}
	}
}

func TestLoadLocalSource(t *testing.T) {
	dir := t.TempDir()
	manifest := []byte(`{"catalog_sha256":"abc"}`)
	writeFile(t, filepath.Join(dir, "manifest.json"), manifest)

	// No bundle on disk: bundlePath stays empty so the signature step fails closed.
	src, err := loadLocalSource(dir)
	if err != nil {
		t.Fatalf("loadLocalSource() error: %v", err)
	}
	defer src.cleanup()

	if string(src.manifestBytes) != string(manifest) {
		t.Errorf("manifestBytes = %q, want %q", src.manifestBytes, manifest)
	}
	if src.bundlePath != "" {
		t.Errorf("bundlePath = %q, want empty when no bundle exists", src.bundlePath)
	}

	// With a bundle present it should be picked up.
	writeFile(t, filepath.Join(dir, sigstoreBundleName), []byte(`{}`))
	src2, err := loadLocalSource(dir)
	if err != nil {
		t.Fatalf("loadLocalSource() error: %v", err)
	}
	defer src2.cleanup()
	if src2.bundlePath == "" {
		t.Error("bundlePath is empty, want the discovered bundle path")
	}

	if _, err := loadLocalSource(filepath.Join(dir, "missing")); err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestVerifySourceFetchLocal(t *testing.T) {
	dir := t.TempDir()
	want := []byte(`{"openai/gpt-5":{}}`)
	if err := os.MkdirAll(filepath.Join(dir, "providers"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "providers", "openai.json"), want)

	src := &verifySource{dir: dir}

	got, err := src.fetch("", filepath.Join("providers", "openai.json"))
	if err != nil {
		t.Fatalf("fetch() error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("fetch() = %q, want %q", got, want)
	}

	if _, err := src.fetch("", filepath.Join("providers", "absent.json")); err == nil {
		t.Error("expected error for missing file")
	}

	// A traversal-style relative path must be refused rather than read.
	_, err = src.fetch("", filepath.Join("providers", "../../../../etc/passwd.json"))
	if err == nil {
		t.Fatal("expected error for path escaping the dist dir")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error = %v, want an escape rejection", err)
	}
}

func TestVerifySourceFetchRemote(t *testing.T) {
	want := []byte(`{"openai/gpt-5":{}}`)
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(want)
	}))
	defer srv.Close()

	src := &verifySource{base: srv.URL}

	got, err := src.fetch("/v1/providers/openai/abc.json", "providers/openai.json")
	if err != nil {
		t.Fatalf("fetch() error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("fetch() = %q, want %q", got, want)
	}
	if gotPath != "/v1/providers/openai/abc.json" {
		t.Errorf("requested %q, want the manifest-provided url", gotPath)
	}

	// A manifest entry without a url must fail rather than silently fall back.
	if _, err := src.fetch("", "providers/openai.json"); err == nil {
		t.Error("expected error when the manifest has no url")
	}
}

func TestLoadRemoteSource(t *testing.T) {
	manifest := []byte(`{"catalog_sha256":"abc"}`)
	bundle := []byte(`{"bundle":true}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/manifest.json":
			_, _ = w.Write(manifest)
		case "/v1/" + sigstoreBundleName:
			_, _ = w.Write(bundle)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Trailing slash must be trimmed so URLs don't end up with a double slash.
	src, err := loadRemoteSource(srv.URL + "/")
	if err != nil {
		t.Fatalf("loadRemoteSource() error: %v", err)
	}
	defer src.cleanup()

	if string(src.manifestBytes) != string(manifest) {
		t.Errorf("manifestBytes = %q, want %q", src.manifestBytes, manifest)
	}
	if src.base != srv.URL {
		t.Errorf("base = %q, want %q", src.base, srv.URL)
	}
	// cosign reads from disk, so both artifacts must be materialized.
	assertFileContains(t, src.manifestPath, manifest)
	assertFileContains(t, src.bundlePath, bundle)

	// cleanup must remove the temp dir.
	tmp := src.tmpDir
	src.cleanup()
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("tmpDir %s still exists after cleanup", tmp)
	}
}

func TestLoadRemoteSourceMissingBundle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/manifest.json" {
			_, _ = w.Write([]byte(`{"catalog_sha256":"abc"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	src, err := loadRemoteSource(srv.URL)
	if err != nil {
		t.Fatalf("loadRemoteSource() error: %v", err)
	}
	defer src.cleanup()

	// An absent bundle is not fatal here; verifySignature fails closed on it.
	if src.bundlePath != "" {
		t.Errorf("bundlePath = %q, want empty when the bundle is absent", src.bundlePath)
	}
}

func TestLoadRemoteSourceUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	srv.Close() // closed: connection refused

	if _, err := loadRemoteSource(srv.URL); err == nil {
		t.Error("expected error when the manifest cannot be fetched")
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileContains(t *testing.T, path string, want []byte) {
	t.Helper()
	if path == "" {
		t.Fatal("expected a non-empty path")
	}
	got, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
