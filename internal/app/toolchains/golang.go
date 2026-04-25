package toolchains

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type GoInstaller struct{}

var (
	goDownloadBaseURL = "https://go.dev/dl/"
	goChecksumBaseURL = "https://dl.google.com/go/"
	goReleaseIndexURL = "https://go.dev/dl/?mode=json&include=all"
	goReadURL         = helpers.ReadURL
)

type goRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func (g GoInstaller) Name() string {
	return "go"
}

func (g GoInstaller) archiveName(version, goos, goarch string) string {
	return fmt.Sprintf("go%s.%s-%s.tar.gz", version, goos, goarch)
}

func (g GoInstaller) downloadURL(version, goos, goarch string) string {
	return goDownloadBaseURL + g.archiveName(version, goos, goarch)
}

func (g GoInstaller) checksumURL(version, goos, goarch string) string {
	return goChecksumBaseURL + g.archiveName(version, goos, goarch) + ".sha256"
}

func (g GoInstaller) installDir(root, version string) string {
	return filepath.Join(root, "go", version)
}

func (g GoInstaller) binaryPath(root, version string) string {
	return filepath.Join(root, "go", version, "go", "bin", "go")
}

func (g GoInstaller) ResolveVersion(ic *itoolchain.InstallContext, version string) (string, error) {
	if version == "latest" {
		resolvedVersion, err := g.resolveLatestPublishedVersion()
		if err == nil {
			return resolvedVersion, nil
		}

		localVersion, localErr := g.latestInstalledVersion(ic.ToolchainDir, "")
		if localErr == nil {
			return localVersion, nil
		}

		return "", fmt.Errorf("resolve latest Go version: %w", err)
	}

	if !isGoMinorSeries(version) {
		return version, nil
	}

	resolvedVersion, err := g.resolveLatestPublishedVersionForSeries(version)
	if err == nil {
		return resolvedVersion, nil
	}

	localVersion, localErr := g.latestInstalledVersion(ic.ToolchainDir, version)
	if localErr == nil {
		return localVersion, nil
	}

	return "", fmt.Errorf("resolve Go version %s: %w", version, err)
}

func (g GoInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	archiveName := g.archiveName(version, ic.GOOS, ic.GOARCH)
	return installArchiveIfNeeded(
		ic,
		g.binaryPath(ic.ToolchainDir, version),
		g.downloadURL(version, ic.GOOS, ic.GOARCH),
		archiveName,
		g.installDir(ic.ToolchainDir, version),
		func(archivePath string) error {
			return helpers.VerifyDownloadedArchive(
				archivePath,
				archiveName,
				g.checksumURL(version, ic.GOOS, ic.GOARCH),
			)
		},
	)
}

func (g GoInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return filepath.Dir(g.binaryPath(ic.ToolchainDir, version)), nil
}

func (g GoInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}

func (g GoInstaller) resolveLatestPublishedVersion() (string, error) {
	releases, err := g.releaseIndex()
	if err != nil {
		return "", err
	}

	return latestStableGoVersion(releases)
}

func (g GoInstaller) resolveLatestPublishedVersionForSeries(series string) (string, error) {
	releases, err := g.releaseIndex()
	if err != nil {
		return "", err
	}

	return latestGoPatchVersion(releases, series)
}

func (g GoInstaller) releaseIndex() ([]goRelease, error) {
	body, err := goReadURL(goReleaseIndexURL)
	if err != nil {
		return nil, err
	}

	var releases []goRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (g GoInstaller) latestInstalledVersion(root, series string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "go"))
	if err != nil {
		return "", err
	}

	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		if !isExactGoVersion(version) {
			continue
		}
		if series != "" && !strings.HasPrefix(version, series+".") {
			continue
		}

		candidates = append(candidates, version)
	}

	if len(candidates) == 0 {
		if series == "" {
			return "", fmt.Errorf("no installed Go release found")
		}
		return "", fmt.Errorf("no installed Go release found for %s", series)
	}

	slices.SortFunc(candidates, compareGoVersions)
	return candidates[len(candidates)-1], nil
}

func latestStableGoVersion(releases []goRelease) (string, error) {
	candidates := make([]string, 0, len(releases))
	seen := make(map[string]struct{}, len(releases))

	for _, release := range releases {
		if !release.Stable {
			continue
		}

		version, ok := normalizeGoReleaseVersion(release.Version)
		if !ok {
			continue
		}
		if _, exists := seen[version]; exists {
			continue
		}

		seen[version] = struct{}{}
		candidates = append(candidates, version)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no stable Go release found")
	}

	slices.SortFunc(candidates, compareGoVersions)
	return candidates[len(candidates)-1], nil
}

func latestGoPatchVersion(releases []goRelease, series string) (string, error) {
	candidates := make([]string, 0, len(releases))
	seen := make(map[string]struct{}, len(releases))

	for _, release := range releases {
		if !release.Stable {
			continue
		}

		version, ok := normalizeGoReleaseVersion(release.Version)
		if !ok {
			continue
		}
		if !strings.HasPrefix(version, series+".") {
			continue
		}
		if _, exists := seen[version]; exists {
			continue
		}

		seen[version] = struct{}{}
		candidates = append(candidates, version)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no Go patch release found for %s", series)
	}

	slices.SortFunc(candidates, compareGoVersions)
	return candidates[len(candidates)-1], nil
}

func normalizeGoReleaseVersion(version string) (string, bool) {
	if !strings.HasPrefix(version, "go") {
		return "", false
	}

	version = strings.TrimPrefix(version, "go")
	if !isExactGoVersion(version) {
		return "", false
	}

	return version, true
}

func isGoMinorSeries(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

func isExactGoVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

func compareGoVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	return 0
}
