package toolchain

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Expected tool versions for proto code generation.
//
// To update these versions:
//  1. Update the tool on your system (e.g. brew upgrade protobuf, go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)
//  2. Verify the new version by running the tool with --version (e.g., protoc --version)
//  3. Update the corresponding constant below with the new version number
//  4. Run the code generator to update generated files with new version headers
const (
	ExpectedProtoc           = "32.1"
	ExpectedProtocGenGo      = "1.36.10"
	ExpectedProtocGenGoGRPC  = "1.5.1"
	ExpectedProtocGenJS      = "4.0.1"
	ExpectedProtocGenGRPCWeb = "2.0.2"
)

// Tool represents a toolchain command with its expected version.
type Tool struct {
	Name            string
	VersionArg      string // Command-line arg to get version (usually --version)
	VersionPattern  *regexp.Regexp
	ExpectedVersion string
}

var tools = []Tool{
	{
		Name:            "protoc",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`libprotoc (\d+\.\d+(?:\.\d+)?)`),
		ExpectedVersion: ExpectedProtoc,
	},
	{
		Name:            "protoc-gen-go",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`protoc-gen-go v(\d+\.\d+\.\d+)`),
		ExpectedVersion: ExpectedProtocGenGo,
	},
	{
		Name:            "protoc-gen-go-grpc",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`protoc-gen-go-grpc (\d+\.\d+\.\d+)`),
		ExpectedVersion: ExpectedProtocGenGoGRPC,
	},
	{
		Name:            "protoc-gen-js",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`protoc-gen-js version (\d+\.\d+\.\d+)`),
		ExpectedVersion: ExpectedProtocGenJS,
	},
	{
		Name:            "protoc-gen-grpc-web",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`protoc-gen-grpc-web (\d+\.\d+\.\d+)`),
		ExpectedVersion: ExpectedProtocGenGRPCWeb,
	},
}

// VerifyVersions checks that all required tools are installed with the expected versions.
func VerifyVersions() error {
	var errs []string
	for _, tool := range tools {
		if err := verifyTool(tool); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("version verification failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func verifyTool(tool Tool) error {
	cmd := exec.Command(tool.Name, tool.VersionArg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: not found or failed to run", tool.Name)
	}

	matches := tool.VersionPattern.FindStringSubmatch(string(output))
	if matches == nil {
		return fmt.Errorf("%s: could not parse version from output: %q", tool.Name, strings.TrimSpace(string(output)))
	}

	version := matches[1]
	if version != tool.ExpectedVersion {
		return fmt.Errorf("%s: got version %q != %q", tool.Name, version, tool.ExpectedVersion)
	}

	return nil
}
