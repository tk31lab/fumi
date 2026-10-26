// Package cmd defines fumi's command-line interface.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// exitCodeError carries a non-zero exit code, such as the bastion shell's,
// without printing an error message.
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fumi",
		Short: "Start throwaway bastion containers on Docker networks",
		Long: `fumi (踏み) starts a throwaway "bastion" container on a Docker network and
opens a shell in it, so you can reach services that publish no ports,
such as a database started by Docker Compose.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newRunCmd(), newForwardCmd(), newNetworksCmd())
	return root
}

// Execute runs the root command and exits with the appropriate status.
func Execute() {
	err := newRootCmd().ExecuteContext(context.Background())
	if err == nil {
		return
	}
	var exitErr exitCodeError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.code)
	}
	fmt.Fprintln(os.Stderr, "fumi:", err)
	os.Exit(1)
}
