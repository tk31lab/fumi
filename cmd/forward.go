package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/forward"
)

func newForwardCmd() *cobra.Command {
	opts := forward.Options{Stderr: os.Stderr}

	cmd := &cobra.Command{
		Use:   "forward --network NETWORK [LOCAL_PORT:]HOST:PORT",
		Short: "Forward a local port to a service on a network",
		Long: `Forward a TCP port on 127.0.0.1 to HOST:PORT on a Docker network, by running
socat in a throwaway container. The service itself does not need to publish
any port. Without LOCAL_PORT, the same port number is used locally.

The forward is only reachable from this machine (127.0.0.1). It runs until
Ctrl+C, then the container is removed.`,
		Example: `  fumi forward --network myapp_default 8090:web:80
  fumi forward -n myapp_default db:5432`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := forward.ParseSpec(args[0])
			if err != nil {
				return err
			}
			opts.Spec = spec

			cli, err := dockerutil.Connect(cmd.Context())
			if err != nil {
				return err
			}
			defer cli.Close()
			return forward.Run(cmd.Context(), cli, opts)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.Network, "network", "n", "", "network to join (required)")
	f.StringVar(&opts.Name, "name", "", `container name (default: generated, e.g. "fumi-shizuka-tsuki")`)
	_ = cmd.MarkFlagRequired("network")

	_ = cmd.RegisterFlagCompletionFunc("network", completeNetworks)
	_ = cmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions)
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}
