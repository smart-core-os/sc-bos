// install-proto-tools installs the three external protobuf toolchain binaries
// that genproto (cmd/tools/genproto) requires to generate Go and JavaScript
// code from .proto files:
//
//   - protoc            – the Protocol Buffers compiler
//   - protoc-gen-js     – generates JavaScript stubs
//   - protoc-gen-grpc-web – generates gRPC-Web JavaScript stubs
//
// Versions are read directly from
// cmd/tools/genproto/internal/toolchain/versions.go so this tool and the
// generator always agree on what to install.  If the version constants in
// that file change, re-running this tool will update your local install.
//
// # Installation strategy
//
// With Homebrew (the common macOS path):
//
//   - protoc is managed through a local Homebrew tap (sc-bos/proto-tools).
//     The tap is created under $(brew --repository)/Library/Taps/ if it does
//     not already exist.  The correct .rb formula is fetched from the
//     homebrew-core GitHub repository by walking commit history until the
//     formula at that commit matches the required version, then written into
//     the tap.
//
//   - protoc-gen-grpc-web and protoc-gen-js are downloaded as pre-built
//     binaries from their respective GitHub release pages and placed directly
//     into $(brew --prefix)/bin.  Homebrew bottles for these plugins are
//     dynamically linked against whichever libprotobuf happened to be current
//     when the bottle was built, making them unreliable whenever the pinned
//     protobuf version differs from Homebrew's latest.  The GitHub release
//     binaries do not carry this dependency.
//
// Without Homebrew, all three binaries are downloaded from GitHub releases
// and installed to /usr/local/bin (Intel) or /opt/homebrew/bin (Apple Silicon).
//
// # GitHub API rate limits
//
// The formula search paginates through homebrew-core commit history, which
// consumes GitHub API quota.  Without authentication the limit is 60 requests
// per hour; setting GITHUB_TOKEN (or the -token flag) raises it to 5 000.
//
// # Usage
//
//	go run ./cmd/tools/install-proto-tools            # install or upgrade
//	go run ./cmd/tools/install-proto-tools -check     # verify without installing
//	GITHUB_TOKEN=ghp_xxx go run ./cmd/tools/install-proto-tools
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ── Constants ─────────────────────────────────────────────────────────────

const (
	// localTapOwner and localTapRepo together form the Homebrew tap reference
	// "sc-bos/proto-tools".  Homebrew requires the on-disk directory to be
	// named homebrew-<repo>, so it lives at:
	//   $(brew --repository)/Library/Taps/sc-bos/homebrew-proto-tools/
	localTapOwner = "sc-bos"
	localTapRepo  = "proto-tools"

	// versionsRelPath is the path to the genproto versions file relative to
	// the repository root.  This is the single source of truth for which tool
	// versions are required.
	versionsRelPath = "cmd/tools/genproto/internal/toolchain/versions.go"

	// hbCoreRepo is the GitHub repository that hosts homebrew-core formulas.
	hbCoreRepo = "Homebrew/homebrew-core"

	// protocFormulaP is the path to the protobuf formula inside homebrew-core.
	protocFormulaP = "Formula/p/protobuf.rb"

	// GitHub repositories used when downloading binaries directly.
	repoProtoc      = "protocolbuffers/protobuf"
	repoGRPCWeb     = "grpc/grpc-web"
	repoProtocGenJS = "protocolbuffers/protobuf-javascript"
)

// ── Data types ────────────────────────────────────────────────────────────

// Versions mirrors the three version constants declared in versions.go.
type Versions struct {
	Protoc           string // version reported by `protoc --version`, e.g. "34.0"
	ProtocGenJS      string // version reported by `protoc-gen-js --version`
	ProtocGenGRPCWeb string // version reported by `protoc-gen-grpc-web --version`
}

// The gh* types are minimal shapes for the GitHub REST API responses we care
// about.  Only the fields we actually use are declared.

type ghCommit struct {
	SHA string `json:"sha"`
}

type ghContent struct {
	Content  string `json:"content"`  // base64-encoded file content
	Encoding string `json:"encoding"` // always "base64" for blobs
}

type ghRelease struct {
	Assets []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// ── Flags ─────────────────────────────────────────────────────────────────

var (
	checkOnly = flag.Bool("check", false,
		"verify installed versions and exit without installing anything")
	githubToken = flag.String("token", os.Getenv("GITHUB_TOKEN"),
		"GitHub personal access token; overrides GITHUB_TOKEN env var")
)

// ── Entry point ───────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	if runtime.GOOS != "darwin" {
		die("this tool only supports macOS")
	}

	repoRoot := gitOutput("rev-parse", "--show-toplevel")
	versions := parseVersions(filepath.Join(repoRoot, versionsRelPath))

	logf("Required versions:")
	logf("  protoc              %s", versions.Protoc)
	logf("  protoc-gen-js       %s", versions.ProtocGenJS)
	logf("  protoc-gen-grpc-web %s", versions.ProtocGenGRPCWeb)

	if *checkOnly {
		if !verifyAll(versions, false) {
			os.Exit(1)
		}
		return
	}

	// Fast path: nothing to do if all tools are already correct.
	logf("")
	if verifyAll(versions, true) {
		logf("All tools are already at the required versions.")
		return
	}

	brewPath, _ := exec.LookPath("brew")
	if brewPath != "" {
		logf("Homebrew found at %s – using local tap approach.", brewPath)
		installViaHomebrew(versions)
	} else {
		logf("Homebrew not found – downloading binaries directly.")
		installDirect(versions)
	}

	logf("")
	logf("Verifying installations...")
	verifyAll(versions, false)
	logf("")
	logf("Done!")
}

// ── Version parsing ───────────────────────────────────────────────────────

// Regexps that extract the version string from each constant declaration in
// versions.go.  We parse the source file directly rather than importing the
// internal package so this tool can be run stand-alone without building the
// whole module.
var (
	reProtoc     = regexp.MustCompile(`ExpectedProtoc\b[^=]*=\s*"([^"]+)"`)
	reGenJS      = regexp.MustCompile(`ExpectedProtocGenJS\b[^=]*=\s*"([^"]+)"`)
	reGenGRPCWeb = regexp.MustCompile(`ExpectedProtocGenGRPCWeb\b[^=]*=\s*"([^"]+)"`)
)

// parseVersions reads the three version constants from versions.go.
func parseVersions(path string) *Versions {
	src, err := os.ReadFile(path)
	if err != nil {
		die("reading %s: %v", path, err)
	}
	s := string(src)
	get := func(re *regexp.Regexp, label string) string {
		m := re.FindStringSubmatch(s)
		if m == nil {
			die("could not find %s in %s", label, path)
		}
		return m[1]
	}
	return &Versions{
		Protoc:           get(reProtoc, "ExpectedProtoc"),
		ProtocGenJS:      get(reGenJS, "ExpectedProtocGenJS"),
		ProtocGenGRPCWeb: get(reGenGRPCWeb, "ExpectedProtocGenGRPCWeb"),
	}
}

// ── Homebrew installation ─────────────────────────────────────────────────

// installViaHomebrew manages the three tools using the local Homebrew tap for
// protobuf and GitHub release binaries for the two plugins.
func installViaHomebrew(versions *Versions) {
	tapDir := ensureLocalTap()
	formulaDir := filepath.Join(tapDir, "Formula")
	tapFull := localTapOwner + "/" + localTapRepo
	binDir := homebrewBinDir()

	// protobuf ──────────────────────────────────────────────────────────────
	if toolVersionOK("protoc", "--version", regexp.MustCompile(`libprotoc (\S+)`), versions.Protoc) {
		logf("protobuf %s already installed – skipping.", versions.Protoc)
	} else {
		logf("")
		logf("Fetching protobuf formula for version %s from homebrew-core...", versions.Protoc)
		rb := fetchFormulaAtVersion(protocFormulaP, versions.Protoc)
		// The formula file name must match the Ruby class name that Homebrew
		// derives from it (protobuf.rb → class Protobuf).
		writeTextFile(filepath.Join(formulaDir, "protobuf.rb"), rb)
		commitTap(tapDir)
		// Homebrew refuses to install a formula that is already installed from
		// a different tap.  Uninstall first so we can own it from our tap.
		if brewFormulaInstalled("protobuf") {
			logf("Uninstalling existing protobuf to replace with version %s...", versions.Protoc)
			brewRun("uninstall", "--force", "protobuf")
		}
		brewRun("install", tapFull+"/protobuf")
	}

	// protoc-gen-grpc-web ───────────────────────────────────────────────────
	// We intentionally bypass Homebrew for this plugin.  The bottled formula
	// is dynamically linked against whichever libprotobuf was current when the
	// bottle was built, so it breaks whenever our pinned protobuf version
	// differs from Homebrew's latest.  The GitHub release binary carries no
	// such dependency.
	if toolVersionOK("protoc-gen-grpc-web", "--version",
		regexp.MustCompile(`protoc-gen-grpc-web (\S+)`), versions.ProtocGenGRPCWeb) {
		logf("protoc-gen-grpc-web %s already installed – skipping.", versions.ProtocGenGRPCWeb)
	} else {
		logf("")
		logf("Installing protoc-gen-grpc-web %s from GitHub releases...", versions.ProtocGenGRPCWeb)
		installProtocGenGRPCWebBinary(versions.ProtocGenGRPCWeb, binDir)
	}

	// protoc-gen-js ─────────────────────────────────────────────────────────
	// Not published to homebrew-core; always installed from GitHub releases.
	// Same dynamic-linking concern as protoc-gen-grpc-web applies here too.
	if toolVersionOK("protoc-gen-js", "--version",
		regexp.MustCompile(`protoc-gen-js version (\S+)`), versions.ProtocGenJS) {
		logf("protoc-gen-js %s already installed – skipping.", versions.ProtocGenJS)
	} else {
		logf("")
		logf("Installing protoc-gen-js %s from GitHub releases...", versions.ProtocGenJS)
		installProtocGenJS(versions.ProtocGenJS, binDir)
	}
}

// toolVersionOK returns true when binary name is on PATH and its version
// output matches want exactly.
func toolVersionOK(name, versionArg string, pattern *regexp.Regexp, want string) bool {
	out, err := exec.Command(name, versionArg).CombinedOutput()
	if err != nil {
		return false
	}
	m := pattern.FindStringSubmatch(string(out))
	return m != nil && m[1] == want
}

// ensureLocalTap returns the local tap directory path, initialising it as a
// git repository if it does not yet exist.  Homebrew requires taps to be git
// repos so it can track formula provenance.
func ensureLocalTap() string {
	out, err := exec.Command("brew", "--repository").Output()
	if err != nil {
		die("brew --repository: %v", err)
	}
	brewRepo := strings.TrimSpace(string(out))
	// Homebrew tap directories follow the naming convention:
	//   <brew-repo>/Library/Taps/<owner>/homebrew-<repo>
	tapDir := filepath.Join(brewRepo, "Library", "Taps", localTapOwner, "homebrew-"+localTapRepo)

	if _, statErr := os.Stat(tapDir); os.IsNotExist(statErr) {
		logf("Creating local tap at %s", tapDir)
		must(os.MkdirAll(filepath.Join(tapDir, "Formula"), 0o755))
		gitIn(tapDir, "init")
		gitIn(tapDir, "config", "user.email", "install-proto-tools@local")
		gitIn(tapDir, "config", "user.name", "install-proto-tools")
		gitIn(tapDir, "commit", "--allow-empty", "-m", "init tap")
	} else {
		logf("Using existing local tap at %s", tapDir)
	}
	return tapDir
}

// commitTap stages all changes in the tap directory and creates a new commit
// so that Homebrew picks up any updated formula files.
func commitTap(tapDir string) {
	gitIn(tapDir, "add", "-A")
	gitIn(tapDir, "commit", "--allow-empty", "-m", "update proto tool formulas")
}

// homebrewBinDir returns the bin directory under $(brew --prefix), falling
// back to the default platform bin directory if brew is not queryable.
func homebrewBinDir() string {
	out, err := exec.Command("brew", "--prefix").Output()
	if err != nil {
		return defaultBinDir()
	}
	return filepath.Join(strings.TrimSpace(string(out)), "bin")
}

// brewFormulaInstalled reports whether formula is currently installed via brew.
func brewFormulaInstalled(formula string) bool {
	return exec.Command("brew", "list", "--formula", formula).Run() == nil
}

// brewRun runs brew with the given arguments, streaming output to stdout/stderr.
// It terminates the process on failure.
func brewRun(args ...string) {
	logf("  brew %s", strings.Join(args, " "))
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("brew %s: %v", strings.Join(args, " "), err)
	}
}

// ── Formula fetching from homebrew-core GitHub ────────────────────────────

// fetchFormulaAtVersion searches the homebrew-core commit history for the
// formula file at formulaPath and returns the raw Ruby content from the most
// recent commit where the formula represents targetVersion.
//
// It works by paging through commits that touched the formula file (each page
// covering 100 commits), fetching the file content at every SHA via the
// GitHub Contents API, and delegating version detection to formulaIsVersion.
//
// The search covers up to 10 pages (1 000 commits).  For very old versions,
// or when GitHub rate-limits unauthenticated requests (60 req/hr), set
// GITHUB_TOKEN or -token to raise the limit to 5 000 req/hr.
func fetchFormulaAtVersion(formulaPath, targetVersion string) string {
	for page := 1; page <= 10; page++ {
		url := fmt.Sprintf(
			"https://api.github.com/repos/%s/commits?path=%s&per_page=100&page=%d",
			hbCoreRepo, formulaPath, page,
		)
		var commits []ghCommit
		ghGet(url, &commits)
		if len(commits) == 0 {
			break
		}
		logf("  Page %d: inspecting %d commits...", page, len(commits))
		for _, c := range commits {
			content := fetchFileAtSHA(formulaPath, c.SHA)
			if formulaIsVersion(content, targetVersion) {
				logf("  Found %s at commit %s", targetVersion, c.SHA[:8])
				return content
			}
		}
	}
	die("formula %s at version %s not found in homebrew-core history (10 pages × 100 commits).\n"+
		"  Tip: set GITHUB_TOKEN or -token to avoid API rate limiting.",
		formulaPath, targetVersion)
	return ""
}

// fetchFileAtSHA returns the decoded text content of a file in homebrew-core
// at the given commit SHA, using the GitHub Contents API.
func fetchFileAtSHA(path, sha string) string {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/contents/%s?ref=%s",
		hbCoreRepo, path, sha,
	)
	var c ghContent
	ghGet(url, &c)
	if c.Encoding != "base64" {
		die("unexpected content encoding %q at %s@%s", c.Encoding, path, sha)
	}
	// GitHub wraps the base64 at 60 characters with newlines; strip them first.
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.Content, "\n", ""))
	if err != nil {
		die("base64 decode %s@%s: %v", path, sha, err)
	}
	return string(raw)
}

// formulaIsVersion reports whether the Homebrew Ruby formula content
// represents the given version.  Two indicators are checked, in order:
//
//  1. An explicit `version "x"` field at class level.
//  2. The top-level `url` line (2-space indent), from which Homebrew
//     auto-detects the version when no explicit field is present.
//
// Lines inside `resource … end` blocks are skipped because those blocks
// describe bundled dependencies, not the formula's own version.
func formulaIsVersion(content, version string) bool {
	versionQ := regexp.QuoteMeta(version)
	explicitVersionRe := regexp.MustCompile(`^[ ]{0,4}version\s+"` + versionQ + `"`)
	// Matches version appearing as a path or filename component in a URL,
	// e.g. /v34.0/ or protobuf-34.0.tar.gz
	urlVersionRe := regexp.MustCompile(`[/\-_]v?` + versionQ + `[/\-_.]`)

	sc := bufio.NewScanner(strings.NewReader(content))
	inResource := false
	depth := 0

	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		// Track do … end / resource … end nesting depth so we can ignore
		// dependency URLs that might happen to contain our version string.
		if strings.HasPrefix(trimmed, "resource ") || strings.HasSuffix(trimmed, " do") {
			inResource = true
			depth++
		}
		if inResource {
			if trimmed == "end" {
				depth--
				if depth <= 0 {
					inResource = false
					depth = 0
				}
			}
			continue
		}

		if explicitVersionRe.MatchString(line) {
			return true
		}
		// Only match the class-level url (2-space indent); 4-space belongs to
		// a nested block.
		if strings.HasPrefix(line, "  url ") && urlVersionRe.MatchString(line) {
			return true
		}
	}
	return false
}

// ── Plugin binary installation from GitHub releases ───────────────────────

// installProtocGenJS downloads the macOS protoc-gen-js binary for the given
// version from the protobuf-javascript GitHub release page and writes it to
// binDir.  Release tags use a "v" prefix (e.g. v4.0.1).
func installProtocGenJS(version, binDir string) {
	rel := fetchRelease(repoProtocGenJS, "v"+version)
	assetURL := pickMacOSAsset(rel.Assets)
	if assetURL == "" {
		die("no macOS asset for protoc-gen-js %s.\n  Available: %s",
			version, listAssets(rel.Assets))
	}
	data := download(assetURL)
	dest := filepath.Join(binDir, "protoc-gen-js")
	if isZip(assetURL) {
		extractZipEntry(data, "bin/protoc-gen-js", dest)
	} else {
		writeExec(dest, data)
	}
}

// installProtocGenGRPCWebBinary downloads the macOS protoc-gen-grpc-web binary
// for the given version from the grpc-web GitHub release page and writes it to
// binDir.  grpc-web release tags do not use a "v" prefix (e.g. 2.0.2).
func installProtocGenGRPCWebBinary(version, binDir string) {
	rel := fetchRelease(repoGRPCWeb, version)
	assetURL := pickMacOSAsset(rel.Assets)
	if assetURL == "" {
		die("no macOS asset for protoc-gen-grpc-web %s.\n  Available: %s",
			version, listAssets(rel.Assets))
	}
	data := download(assetURL)
	dest := filepath.Join(binDir, "protoc-gen-grpc-web")
	if isZip(assetURL) {
		extractZipEntry(data, "bin/protoc-gen-grpc-web", dest)
	} else {
		writeExec(dest, data)
	}
}

// ── Direct binary installation (no Homebrew) ──────────────────────────────

// installDirect downloads all three tool binaries from GitHub releases and
// writes them to the platform default bin directory.
func installDirect(versions *Versions) {
	binDir := defaultBinDir()
	must(os.MkdirAll(binDir, 0o755))
	logf("Installing binaries to %s", binDir)

	logf("")
	logf("Downloading protoc %s...", versions.Protoc)
	rel := fetchRelease(repoProtoc, "v"+versions.Protoc)
	url := pickMacOSAsset(rel.Assets)
	if url == "" {
		die("no macOS asset for protoc %s.\n  Available: %s", versions.Protoc, listAssets(rel.Assets))
	}
	data := download(url)
	if isZip(url) {
		extractZipEntry(data, "bin/protoc", filepath.Join(binDir, "protoc"))
		// protoc releases bundle well-known .proto files under include/; extract
		// them alongside the binary so protoc can find google/protobuf/*.proto.
		extractZipDir(data, "include", filepath.Join(filepath.Dir(binDir), "include"))
	} else {
		writeExec(filepath.Join(binDir, "protoc"), data)
	}

	logf("")
	logf("Downloading protoc-gen-grpc-web %s...", versions.ProtocGenGRPCWeb)
	installProtocGenGRPCWebBinary(versions.ProtocGenGRPCWeb, binDir)

	logf("")
	logf("Downloading protoc-gen-js %s...", versions.ProtocGenJS)
	installProtocGenJS(versions.ProtocGenJS, binDir)
}

// ── Verification ──────────────────────────────────────────────────────────

// verifyAll checks that each tool is on PATH and reports the expected version.
// When silent is true, output is suppressed and the function is used only for
// its boolean return value (e.g. the "already installed?" fast-path check).
func verifyAll(versions *Versions, silent bool) bool {
	type check struct {
		name    string
		arg     string
		pattern *regexp.Regexp
		want    string
	}
	checks := []check{
		{"protoc", "--version",
			regexp.MustCompile(`libprotoc (\S+)`), versions.Protoc},
		{"protoc-gen-js", "--version",
			regexp.MustCompile(`protoc-gen-js version (\S+)`), versions.ProtocGenJS},
		{"protoc-gen-grpc-web", "--version",
			regexp.MustCompile(`protoc-gen-grpc-web (\S+)`), versions.ProtocGenGRPCWeb},
	}
	all := true
	for _, c := range checks {
		out, err := exec.Command(c.name, c.arg).CombinedOutput()
		if err != nil {
			if !silent {
				logf("  FAIL  %-28s not found or failed to run", c.name)
			}
			all = false
			continue
		}
		m := c.pattern.FindStringSubmatch(string(out))
		if m == nil {
			if !silent {
				logf("  WARN  %-28s could not parse version from output", c.name)
			}
			all = false
			continue
		}
		if m[1] == c.want {
			if !silent {
				logf("  OK    %-28s %s", c.name, m[1])
			}
		} else {
			if !silent {
				logf("  MISMATCH  %-24s got %s, want %s", c.name, m[1], c.want)
			}
			all = false
		}
	}
	if !all && !silent {
		logf("")
		logf("Ensure %s is on your PATH.", defaultBinDir())
	}
	return all
}

// ── GitHub API helpers ────────────────────────────────────────────────────

// ghGet performs an authenticated (when -token / GITHUB_TOKEN is set) GET
// request to the GitHub REST API and JSON-decodes the response into out.
// It terminates the process on HTTP errors, with a specific message for rate
// limit responses.
func ghGet(url string, out interface{}) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		die("building request for %s: %v", url, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if *githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+*githubToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		die("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		die("reading response body from %s: %v", url, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		// success
	case http.StatusForbidden, http.StatusTooManyRequests:
		die("GitHub API rate limit reached.\n  Set GITHUB_TOKEN or use -token for higher limits.\n  Response: %s", body)
	case http.StatusNotFound:
		die("GitHub API 404 for %s", url)
	default:
		die("GET %s → HTTP %d: %s", url, resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		die("decoding response from %s: %v\n  body: %s", url, err, body)
	}
}

// fetchRelease fetches the GitHub release for repo at the given tag.  If the
// release has no assets, it retries with the tag's v-prefix toggled (some
// repos tag as "2.0.2" and others as "v2.0.2").
func fetchRelease(repo, tag string) *ghRelease {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
	var rel ghRelease
	ghGet(url, &rel)
	if len(rel.Assets) == 0 {
		alt := tag
		if strings.HasPrefix(tag, "v") {
			alt = tag[1:]
		} else {
			alt = "v" + tag
		}
		url2 := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, alt)
		ghGet(url2, &rel)
	}
	return &rel
}

// pickMacOSAsset returns the download URL of the best matching macOS asset
// from a release asset list.  Selection priority:
//  1. macOS asset matching the current CPU architecture (arm64 or x86_64)
//  2. Any macOS asset (universal binary, or when no arch-specific one exists)
//
// Returns "" if no macOS asset is found.
func pickMacOSAsset(assets []ghAsset) string {
	isArm := runtime.GOARCH == "arm64"

	archMatch := func(name string) bool {
		n := strings.ToLower(name)
		if isArm {
			return strings.Contains(n, "aarch64") ||
				strings.Contains(n, "arm64") ||
				strings.Contains(n, "aarch_64")
		}
		return strings.Contains(n, "x86_64") || strings.Contains(n, "amd64")
	}
	macMatch := func(name string) bool {
		n := strings.ToLower(name)
		return strings.Contains(n, "osx") ||
			strings.Contains(n, "darwin") ||
			strings.Contains(n, "macos") ||
			strings.Contains(n, "-mac-")
	}

	for _, a := range assets {
		if macMatch(a.Name) && archMatch(a.Name) {
			return a.BrowserDownloadURL
		}
	}
	for _, a := range assets {
		if macMatch(a.Name) {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// listAssets returns a comma-separated list of asset names, used in error messages.
func listAssets(assets []ghAsset) string {
	names := make([]string, len(assets))
	for i, a := range assets {
		names[i] = a.Name
	}
	return strings.Join(names, ", ")
}

// ── Download & ZIP extraction ─────────────────────────────────────────────

// download fetches url and returns the response body.  It terminates the
// process on network errors or non-200 status codes.
func download(url string) []byte {
	logf("  Downloading %s", url)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		die("downloading %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		die("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		die("reading download body: %v", err)
	}
	return data
}

// extractZipEntry locates srcPath inside the zip archive (by exact path or
// trailing basename match) and writes it to destPath with executable
// permissions.
func extractZipEntry(data []byte, srcPath, destPath string) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		die("opening zip: %v", err)
	}
	base := filepath.Base(srcPath)
	for _, f := range r.File {
		if f.Name == srcPath || strings.HasSuffix(f.Name, "/"+base) {
			rc, err := f.Open()
			if err != nil {
				die("opening zip entry %s: %v", f.Name, err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				die("reading zip entry %s: %v", f.Name, err)
			}
			writeExec(destPath, content)
			return
		}
	}
	var entries []string
	for _, f := range r.File {
		entries = append(entries, f.Name)
	}
	die("entry %s not found in zip.\n  Entries: %s", srcPath, strings.Join(entries, ", "))
}

// extractZipDir extracts all regular files under srcDir/ in the zip archive
// to destDir, preserving relative paths.  Errors are silently ignored because
// this is used for bundled include files that are optional.
func extractZipDir(data []byte, srcDir, destDir string) {
	r, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	prefix := srcDir + "/"
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) || f.FileInfo().IsDir() {
			continue
		}
		dest := filepath.Join(destDir, strings.TrimPrefix(f.Name, prefix))
		_ = os.MkdirAll(filepath.Dir(dest), 0o755)
		rc, _ := f.Open()
		content, _ := io.ReadAll(rc)
		rc.Close()
		_ = os.WriteFile(dest, content, 0o644)
	}
}

// isZip reports whether the URL refers to a zip archive.
func isZip(url string) bool {
	return strings.HasSuffix(strings.ToLower(url), ".zip")
}

// writeExec writes data to path with executable permissions (0755), creating
// any missing parent directories.
func writeExec(path string, data []byte) {
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	must(os.WriteFile(path, data, 0o755))
	logf("  Installed %s", path)
}

// writeTextFile writes content to path with read-only permissions (0644).
func writeTextFile(path, content string) {
	must(os.WriteFile(path, []byte(content), 0o644))
	logf("  Wrote %s", path)
}

// ── Shell / git helpers ───────────────────────────────────────────────────

// gitIn runs a git command in dir, streaming output to stdout/stderr.
func gitIn(dir string, args ...string) {
	runIn(dir, append([]string{"git"}, args...)...)
}

// gitOutput runs a git command and returns its trimmed stdout output.
func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		die("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

// runIn runs the given command in dir, streaming stdout/stderr to the
// terminal.  It terminates the process if the command fails.
func runIn(dir string, args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("running %q in %s: %v", strings.Join(args, " "), dir, err)
	}
}

// must terminates the process if err is non-nil.
func must(err error) {
	if err != nil {
		die("%v", err)
	}
}

// defaultBinDir returns the conventional binary directory for non-Homebrew
// installs: /opt/homebrew/bin on Apple Silicon, /usr/local/bin on Intel.
func defaultBinDir() string {
	if runtime.GOARCH == "arm64" {
		return "/opt/homebrew/bin"
	}
	return "/usr/local/bin"
}

func logf(format string, args ...any) {
	fmt.Printf("[install-proto-tools] "+format+"\n", args...)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
