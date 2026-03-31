// Package protov1 migrates proto files from unversioned to v1 versioned packages.
//
// The fixer makes the following changes:
//   - Add a .v1 suffix to proto package names that aren't versioned
//   - Update references to types defined in other protos to use the new versioned package name
//   - For any proto file that is moved, imports of that file are updated to the new path
//
// For protos in the root proto directory, these additional changes are made:
//   - The unversioned package is changed from smartcore.bos to smartcore.bos.<trait> before versioning
//   - The proto file is moved the directory matching the new package name (under proto/)
package protov1

// Implementation note: we rely quite heavily on regexp for finding and replacing parts of proto files.
// While protobufs do have a descriptor format we could use to find the location and types of the various
// elements, rewriting them back to text format is non-trivial. Given the relatively simple and consistent
// structure of proto files, especially out proto files, regexp-based transformations should be OK.
// If we find edge cases that break the regex approach, we can revisit using a proper parser.

import (
	"fmt"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "protov1",
	Desc: "Migrate proto files from unversioned to v1 versioned packages",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	files, err := collectProtoFiles(ctx)
	if err != nil {
		return 0, err
	}

	if len(files) == 0 {
		ctx.Verbose("No proto files to migrate")
		return 0, nil
	}

	for i := range files {
		if err := processProtoFile(&files[i], files); err != nil {
			return 0, fmt.Errorf("processing %s: %w", files[i].oldPath, err)
		}
	}

	// Move files and write updated content
	totalChanges := 0
	for i := range files {
		changed, err := writeProtoFile(ctx, &files[i])
		if err != nil {
			return totalChanges, fmt.Errorf("moving %s: %w", files[i].oldPath, err)
		}
		if changed {
			totalChanges++
		}
	}

	return totalChanges, nil
}
