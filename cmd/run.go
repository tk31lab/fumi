package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tk31lab/fumi/internal/bastion"
	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/network"
)

func newRunCmd() *cobra.Command {
	opts := bastion.Options{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}

	cmd := &cobra.Command{
		Use:   "run --network NETWORK",
		Short: "Start a bastion container on a network and open a shell",
		Example: `  fumi run --network myapp_default
  fumi run -n myapp_default --image debian:stable-slim --shell bash
  fumi run -n myapp_default --name my-bastion`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := dockerutil.Connect(cmd.Context())
			if err != nil {
				return err
			}
			defer cli.Close()

			code, err := bastion.Run(cmd.Context(), cli, opts)
			if err != nil {
				return err
			}
			if code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.Network, "network", "n", "", "network to join (required)")
	f.StringVar(&opts.Image, "image", bastion.DefaultImage, "image to run")
	f.StringVarP(&opts.Shell, "shell", "s", bastion.DefaultShell, "shell to start")
	f.StringVar(&opts.Name, "name", "", `container name (default: generated, e.g. "fumi-shizuka-tsuki")`)
	_ = cmd.MarkFlagRequired("network")

	_ = cmd.RegisterFlagCompletionFunc("network", completeNetworks)
	_ = cmd.RegisterFlagCompletionFunc("shell", cobra.FixedCompletions(
		[]string{"sh", "ash", "bash", "zsh"}, cobra.ShellCompDirectiveNoFileComp))
	_ = cmd.RegisterFlagCompletionFunc("image", cobra.NoFileCompletions)
	_ = cmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions)

	return cmd
}

// completeNetworks offers the names of existing networks. It runs on every
// Tab press, so it must be quick, have no side effects and stay silent on
// errors (e.g. when the daemon is not running).
func completeNetworks(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cli, err := dockerutil.Connect(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer cli.Close()
	names, err := network.Names(cmd.Context(), cli, false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
