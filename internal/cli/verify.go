package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/spf13/cobra"
)

const (
	// certIdentityRegexp and certOIDCIssuer mirror the keyless signing identity
	// in .github/workflows/{pages,release}.yml. Keep in sync with those workflows.
	// The trailing $ is load-bearing: both workflows allow workflow_dispatch, so
	// an unanchored pattern would also accept a cert minted from refs/heads/main-*.
	certIdentityRegexp = `^https://github\.com/ferro-labs/model-catalog/\.github/workflows/(pages|release)\.yml@refs/heads/main$`
	certOIDCIssuer     = "https://token.actions.githubusercontent.com"
	sigstoreBundleName = catalog.ManifestFilename + ".sigstore.json"
)

var (
	verifyDir     string
	verifyURL     string
	verifySkipSig bool
)

func init() {
	verifyCmd.Flags().StringVar(&verifyDir, "dir", "dist", "local dist directory to verify")
	verifyCmd.Flags().StringVar(&verifyURL, "url", "", "verify a published catalog at this base URL (e.g. https://catalog.ferrolabs.ai) instead of --dir")
	verifyCmd.Flags().BoolVar(&verifySkipSig, "skip-signature", false, "skip cosign signature verification (hash-only)")
	rootCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify catalog artifacts against the manifest signature and hashes",
	Long: "Verifies the sigstore signature on manifest.json and checks that catalog.json\n" +
		"and every provider slice match the SHA-256 hashes recorded in the manifest.\n" +
		"A consumer must pass verify before trusting remote catalog data.",
	// A verification failure is an expected outcome, not a usage error — don't
	// dump the help text after it.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVerify()
	},
}

func runVerify() error {
	src, err := loadVerifySource()
	if err != nil {
		return err
	}
	defer src.cleanup()

	var m catalog.Manifest
	if err := json.Unmarshal(src.manifestBytes, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// 1. Signature (authenticity) is the trust root: it proves the manifest —
	// and therefore the hashes below — came from our signing workflow.
	if verifySkipSig {
		fmt.Println("! signature verification skipped (--skip-signature); hash-only")
	} else {
		if err := verifySignature(src.manifestPath, src.bundlePath); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
		fmt.Println("✓ manifest signature verified")
	}

	// 2. Catalog integrity.
	catalogBytes, err := src.fetch(m.CatalogURL, "catalog.json")
	if err != nil {
		return err
	}
	if err := m.VerifyCatalog(catalogBytes); err != nil {
		return err
	}
	fmt.Printf("✓ catalog.json (%s…)\n", shortHash(m.CatalogSHA256))

	// 3. Provider slice integrity.
	var failures []string
	for _, p := range m.Providers {
		if err := validateProviderID(p.ID); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		data, err := src.fetch(p.URL, filepath.Join("providers", p.ID+".json"))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", p.ID, err))
			continue
		}
		if err := m.VerifyProviderSlice(p.ID, data); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "✗ %s\n", f)
		}
		return fmt.Errorf("%d provider slice(s) failed verification", len(failures))
	}
	fmt.Printf("✓ %d provider slices verified\n", len(m.Providers))
	fmt.Println("VERIFY PASSED")
	return nil
}

func verifySignature(manifestPath, bundlePath string) error {
	if _, err := exec.LookPath("cosign"); err != nil {
		return fmt.Errorf("cosign not found in PATH (install cosign or pass --skip-signature): %w", err)
	}
	if bundlePath == "" {
		return fmt.Errorf("no sigstore bundle available (pass --skip-signature for hash-only, or --url to verify a published manifest)")
	}
	// #nosec G204 -- fixed subcommand + program-controlled file paths; no shell involved.
	cmd := exec.Command("cosign", "verify-blob",
		"--bundle", bundlePath,
		"--certificate-identity-regexp", certIdentityRegexp,
		"--certificate-oidc-issuer", certOIDCIssuer,
		manifestPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign: %v\n%s", err, out)
	}
	return nil
}

// providerIDPattern matches the shape every real provider id uses (verified
// against all 83 slices in dist/manifest.json). Manifest data is untrusted until
// the signature check passes — and --skip-signature lets a caller opt out of it
// entirely — so an id must never be able to steer a read out of the providers
// directory, whether by traversal ("../../etc/passwd") or nesting ("etc/shadow").
var providerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validateProviderID(id string) error {
	if !providerIDPattern.MatchString(id) {
		return fmt.Errorf("provider %q: invalid id", id)
	}
	return nil
}

// verifySource abstracts local-dir vs remote-URL artifact loading so runVerify
// has a single code path. base != "" means remote; dir != "" means local.
type verifySource struct {
	manifestBytes []byte
	manifestPath  string // on-disk path for cosign to read
	bundlePath    string // on-disk bundle path; "" when absent
	base          string // remote base URL, "" for local
	dir           string // local dist dir, "" for remote
	tmpDir        string // temp dir to clean up (remote only)
}

func (s *verifySource) cleanup() {
	if s.tmpDir != "" {
		_ = os.RemoveAll(s.tmpDir)
	}
}

// fetch returns an artifact's bytes: over HTTP (remote, using the manifest's
// relative URL) or from disk (local, using the flat slice path).
func (s *verifySource) fetch(remoteRel, localRel string) ([]byte, error) {
	if s.base != "" {
		if remoteRel == "" {
			return nil, fmt.Errorf("%s: manifest has no url", localRel)
		}
		return scrape.FetchJSON(nil, s.base+remoteRel)
	}
	path := filepath.Clean(filepath.Join(s.dir, localRel))
	// Belt to validateProviderID's braces: keep every local read under s.dir, so
	// no future caller can reintroduce an escape through a different field.
	root := filepath.Clean(s.dir)
	if path != root && !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return nil, fmt.Errorf("read %s: path escapes %s", path, root)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

func loadVerifySource() (*verifySource, error) {
	if verifyURL != "" {
		return loadRemoteSource(verifyURL)
	}
	return loadLocalSource(verifyDir)
}

func loadLocalSource(dir string) (*verifySource, error) {
	manifestPath := filepath.Join(dir, catalog.ManifestFilename)
	data, err := os.ReadFile(filepath.Clean(manifestPath))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	// The sigstore bundle is produced in CI, so a freshly built local dist has
	// none; leave bundlePath empty and let the signature step fail closed unless
	// --skip-signature is set.
	bundlePath := filepath.Join(dir, sigstoreBundleName)
	if _, statErr := os.Stat(bundlePath); statErr != nil {
		bundlePath = ""
	}
	return &verifySource{manifestBytes: data, manifestPath: manifestPath, bundlePath: bundlePath, dir: dir}, nil
}

func loadRemoteSource(rawURL string) (*verifySource, error) {
	base := strings.TrimRight(rawURL, "/")
	manifestBytes, err := scrape.FetchJSON(nil, base+"/v1/"+catalog.ManifestFilename)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ferrocat-verify-")
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(tmpDir, catalog.ManifestFilename)
	if writeErr := os.WriteFile(manifestPath, manifestBytes, 0o600); writeErr != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, writeErr
	}

	bundlePath := ""
	if bundleBytes, bundleErr := scrape.FetchJSON(nil, base+"/v1/"+sigstoreBundleName); bundleErr == nil {
		bundlePath = filepath.Join(tmpDir, sigstoreBundleName)
		if writeErr := os.WriteFile(bundlePath, bundleBytes, 0o600); writeErr != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, writeErr
		}
	}

	return &verifySource{
		manifestBytes: manifestBytes,
		manifestPath:  manifestPath,
		bundlePath:    bundlePath,
		base:          base,
		tmpDir:        tmpDir,
	}, nil
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
