package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"go.uber.org/multierr"
)

// RunUntilInterrupt will run the function with a context which is cancelled when the process receives a SIGINT.
// If the function returns an error, it is printed to os.Stderr and the process will exit with a nonzero status code.
// There is special handling for errors from the multierr package; each error will be printed on its own line.
func RunUntilInterrupt(run func(ctx context.Context) error) (status int) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	errs := multierr.Errors(run(ctx))

	switch len(errs) {
	case 0:
	case 1:
		err := errs[0]
		status = 1
		if ec, ok := err.(ExitCoder); ok {
			status = ec.ExitCode()
		}
		if status == 0 {
			_, _ = fmt.Fprintf(os.Stderr, "stopping: %s\n", err.Error())
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "fatal error: %s\n", err.Error())
		}
	default:
		_, _ = fmt.Fprintln(os.Stderr, "fatal errors:")
		for _, err := range errs {
			_, _ = fmt.Fprintf(os.Stderr, "\t%s\n", err.Error())
		}
		status = 1
	}
	return
}

type ExitCoder interface {
	ExitCode() int
}
