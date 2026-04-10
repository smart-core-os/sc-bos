package protofile

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/toolchain"
)

// Parse parses a proto file and returns its descriptor.
// includeDirs are additional -I include paths passed to protoc.
func Parse(protoDir, fileName string, includeDirs []string) (*descriptorpb.FileDescriptorProto, error) {
	// Create a temporary file for the descriptor set output
	tmpFile, err := os.CreateTemp("", "proto-descriptor-*.pb")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	protocArgs := []string{"-I", protoDir}
	for _, dir := range includeDirs {
		protocArgs = append(protocArgs, "-I", dir)
	}
	protocArgs = append(protocArgs,
		"--descriptor_set_out="+tmpPath,
		"--include_imports",
		fileName,
	)
	err = toolchain.RunProtoc("", protocArgs...)
	if err != nil {
		return nil, fmt.Errorf("running protoc: %w", err)
	}

	// Read the descriptor set from the temp file
	output, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("reading descriptor file: %w", err)
	}

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(output, &fds); err != nil {
		return nil, fmt.Errorf("unmarshaling descriptor set: %w", err)
	}

	for _, fd := range fds.GetFile() {
		if fd.GetName() == fileName {
			return fd, nil
		}
	}

	return nil, fmt.Errorf("file descriptor not found for %s", fileName)
}
