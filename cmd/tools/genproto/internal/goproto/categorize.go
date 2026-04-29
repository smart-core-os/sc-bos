package goproto

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/protofile"
)

// ProtoFileInfo holds the generator flags and output directory for a proto file.
type ProtoFileInfo struct {
	Gen       Generator
	OutputDir string // relative to repo root, e.g. "pkg/proto/accesspb"; empty if unresolvable
}

// analyzeProtoFiles returns the required generators and output dirs for each proto file in protoDir.
// The keys of the returned map are relative paths from protoDir for each proto file.
// analyzeProtoFiles recursively walks protoDir to find all .proto files.
// includeDirs are additional -I include paths passed to protoc when parsing files.
// modulePrefix is the Go module prefix used to determine which proto files belong to the repo.
func analyzeProtoFiles(protoDir string, includeDirs []string, modulePrefix string) (map[string]ProtoFileInfo, error) {
	fileInfos := make(map[string]ProtoFileInfo)

	err := filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".proto") {
			return nil
		}

		relPath, err := filepath.Rel(protoDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		fileInfo, err := determineProtoFileInfo(protoDir, relPath, includeDirs, modulePrefix)
		if err != nil {
			return fmt.Errorf("analyzing %s: %w", relPath, err)
		}

		fileInfos[relPath] = fileInfo
		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileInfos, nil
}

// determineProtoFileInfo analyzes a proto file to determine its generator flags and output directory.
func determineProtoFileInfo(protoDir, relPath string, includeDirs []string, modulePrefix string) (ProtoFileInfo, error) {
	fileDesc, err := protofile.Parse(protoDir, relPath, includeDirs)
	if err != nil {
		return ProtoFileInfo{}, fmt.Errorf("parsing proto file: %w", err)
	}

	return ProtoFileInfo{
		Gen:       determineGeneratorsFromDescriptor(fileDesc),
		OutputDir: outputDirFromDescriptor(fileDesc, modulePrefix),
	}, nil
}

// determineGenerators analyzes a proto file to determine which generators it needs.
func determineGenerators(protoDir, relPath string, includeDirs []string) (Generator, error) {
	fileDesc, err := protofile.Parse(protoDir, relPath, includeDirs)
	if err != nil {
		return 0, fmt.Errorf("parsing proto file: %w", err)
	}

	return determineGeneratorsFromDescriptor(fileDesc), nil
}

// determineGeneratorsFromDescriptor analyzes a file descriptor to determine which generators it needs.
// This is separated from determineGenerators to allow testing without file I/O.
func determineGeneratorsFromDescriptor(fileDesc *descriptorpb.FileDescriptorProto) Generator {
	var gen Generator

	if len(fileDesc.GetService()) == 0 {
		// No services, no special generators needed
		return gen
	}

	// Files with services get wrappers
	gen |= GenWrapper
	if isRoutedAPI(fileDesc) {
		gen |= GenRouter
	}
	return gen
}

// outputDirFromDescriptor extracts the output directory relative to the repo root
// from the go_package option of a proto file descriptor.
// Returns empty string if the package path doesn't start with modulePrefix.
func outputDirFromDescriptor(fileDesc *descriptorpb.FileDescriptorProto, modulePrefix string) string {
	goPkg := fileDesc.GetOptions().GetGoPackage()
	if i := strings.Index(goPkg, ";"); i >= 0 {
		goPkg = goPkg[:i]
	}
	prefix := strings.TrimSuffix(modulePrefix, "/") + "/"
	if !strings.HasPrefix(goPkg, prefix) {
		return ""
	}
	return strings.TrimPrefix(goPkg, prefix)
}

// isRoutedAPI determines if a proto file defines a routed API.
// A routed API has services where ALL request messages have a 'name' string field.
func isRoutedAPI(fileDesc *descriptorpb.FileDescriptorProto) bool {
	services := fileDesc.GetService()
	if len(services) == 0 {
		return false
	}

	// Build a map of message types in this file
	messages := make(map[string]*descriptorpb.DescriptorProto)
	pkg := fileDesc.GetPackage()
	for _, msg := range fileDesc.GetMessageType() {
		fullName := msg.GetName()
		if pkg != "" {
			fullName = pkg + "." + fullName
		}
		messages[fullName] = msg
		// Also register without package prefix for local references
		messages[msg.GetName()] = msg
	}

	// Check all methods in all services
	hasAnyRequestMessages := false
	for _, service := range services {
		for _, method := range service.GetMethod() {
			inputType := method.GetInputType()
			inputType = strings.TrimPrefix(inputType, ".")
			// Try with and without package prefix
			simpleName := filepath.Base(strings.ReplaceAll(inputType, ".", "/"))

			msg, ok := messages[inputType]
			if !ok {
				msg, ok = messages[simpleName]
			}

			if !ok {
				// Request message is defined in another file, skip this check
				continue
			}

			hasAnyRequestMessages = true

			if !hasNameField(msg) {
				return false
			}
		}
	}

	// If we didn't find any request messages defined in this file, it's not routed
	if !hasAnyRequestMessages {
		return false
	}

	return true
}

// hasNameField checks if a message has a 'name' string field.
func hasNameField(msg *descriptorpb.DescriptorProto) bool {
	for _, field := range msg.GetField() {
		if field.GetName() == "name" && field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_STRING {
			return true
		}
	}
	return false
}

// groupByGeneratorSet groups proto files by their generator flags.
func groupByGeneratorSet(fileInfos map[string]ProtoFileInfo) map[Generator][]string {
	buckets := make(map[Generator][]string)
	for file, info := range fileInfos {
		buckets[info.Gen] = append(buckets[info.Gen], file)
	}
	// Sort each bucket for deterministic output
	for _, files := range buckets {
		slices.Sort(files)
	}
	return buckets
}
