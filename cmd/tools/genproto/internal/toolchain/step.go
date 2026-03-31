package toolchain

import (
	"errors"
	"slices"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
)

var Step = generator.Step{
	ID:   "verify",
	Desc: "Verify toolchain versions",
	Run:  run,
}

func run(ctx *generator.Context) error {
	ctx.Info("Verifying unmanaged toolchain versions...")

	var errs []error
	for _, tool := range tools {
		if err := verifyTool(tool); err != nil {
			errs = append(errs, err)
		} else {
			ctx.Verbose("  %s: got version %q", tool.Name, tool.ExpectedVersion)
		}
	}

	if len(errs) > 0 {
		// Print errors at Info level so they're visible even without -v
		for _, err := range errs {
			ctx.Info("  %s", err)
		}
		errs = slices.Insert(errs, 0, errors.New("toolchain verification failed"))
		return errors.Join(errs...)
	}

	return nil
}
