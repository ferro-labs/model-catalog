package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var releasePlanDistDir string

func init() {
	releasePlanCmd.Flags().StringVar(&releasePlanDistDir, "dist", "dist", "dist directory containing manifest.json (repo-root-relative; also used as a git tree path)")
	rootCmd.AddCommand(releasePlanCmd)
}

var releasePlanCmd = &cobra.Command{
	Use:   "release-plan",
	Short: "Resolve the final, unique release version for the built catalog",
	Long: `Reads dist/manifest.json, compares its catalog hash against already-released
versions in the same CalVer date family, and decides whether to skip (identical
catalog already published) or release under a unique version (bumping .N on
collision). On a bump it rewrites the manifest version so the released asset
matches the tag. Emits GitHub Actions outputs (skip, version) on stdout.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReleasePlan(releasePlanDistDir, gitFamilyReleases, cmd.OutOrStdout())
	},
}

// existingReleasesFunc lists already-published releases in the same date family
// as base, paired with the catalog hash each was built from. It is a seam so the
// command can be tested without a real git repository.
type existingReleasesFunc func(distDir, base string) ([]catalog.ReleasedVersion, error)

// runReleasePlan is the testable core of the release-plan command.
func runReleasePlan(distDir string, listExisting existingReleasesFunc, out io.Writer) error {
	manifest, err := catalog.ReadManifest(distDir)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(manifest.Version)
	if base == "" {
		return fmt.Errorf("manifest %s has no version", distDir)
	}
	if strings.TrimSpace(manifest.CatalogSHA256) == "" {
		return fmt.Errorf("manifest %s has no catalog_sha256; build is malformed", distDir)
	}

	existing, err := listExisting(distDir, base)
	if err != nil {
		return fmt.Errorf("list existing releases: %w", err)
	}

	decision := catalog.ResolveReleaseVersion(base, manifest.CatalogSHA256, existing)
	if decision.Action == catalog.ActionSkip {
		fmt.Fprintf(os.Stderr, "release-plan: catalog %s already published, skipping\n", manifest.CatalogSHA256)
		return emit(out, "skip=true")
	}

	if decision.Version != base {
		manifest.Version = decision.Version
		if err := catalog.WriteManifest(distDir, manifest); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "release-plan: %s already tagged, bumped manifest version to %s\n", base, decision.Version)
	}

	return emit(out, "skip=false", "version="+decision.Version)
}

// emit writes GitHub Actions output lines, propagating any write error. These
// lines are the release contract; a silently dropped write would make downstream
// steps skip with no error surfaced, so the failure must reach the caller.
func emit(out io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return fmt.Errorf("write release-plan output: %w", err)
		}
	}
	return nil
}

// gitOutput runs git and returns stdout, trimmed stderr, and any error. Capturing
// stderr lets callers surface git's actual message ("not a git repository", auth
// failure, "path does not exist in …") instead of a bare exit status.
func gitOutput(args ...string) (stdout []byte, stderr string, err error) {
	cmd := exec.Command("git", args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	return out, strings.TrimSpace(errBuf.String()), err
}

// gitFamilyReleases is the production seam: it lists git tags in the CalVer date
// family (base and base.N) and reads each tag's catalog_sha256 from the manifest
// committed at that tag.
//
// A failed `git tag -l` is fatal — it means the repository itself is unusable, so
// releasing would be unsafe. A failed `git show` for an individual tag is treated
// as "version taken, hash unknown" (so we never reuse the version) but is logged
// loudly to stderr: a silent empty hash here would make skip-on-identical stop
// working and republish the same catalog every run.
func gitFamilyReleases(distDir, base string) ([]catalog.ReleasedVersion, error) {
	tagsOut, tagsErr, err := gitOutput("tag", "-l", base, base+".*")
	if err != nil {
		return nil, fmt.Errorf("git tag -l: %w: %s", err, tagsErr)
	}

	manifestPath := distDir + "/" + catalog.ManifestFilename
	var releases []catalog.ReleasedVersion
	for tag := range strings.FieldsSeq(string(tagsOut)) {
		show, showErrMsg, showErr := gitOutput("show", tag+":"+manifestPath)
		if showErr != nil {
			fmt.Fprintf(os.Stderr, "release-plan: warning: cannot read manifest at tag %s (%v: %s); treating version as taken with unknown hash\n", tag, showErr, showErrMsg)
			releases = append(releases, catalog.ReleasedVersion{Version: tag})
			continue
		}
		var m catalog.Manifest
		if err := json.Unmarshal(show, &m); err != nil {
			fmt.Fprintf(os.Stderr, "release-plan: warning: manifest at tag %s is not valid JSON (%v); treating version as taken with unknown hash\n", tag, err)
			releases = append(releases, catalog.ReleasedVersion{Version: tag})
			continue
		}
		releases = append(releases, catalog.ReleasedVersion{Version: tag, Hash: m.CatalogSHA256})
	}
	return releases, nil
}
