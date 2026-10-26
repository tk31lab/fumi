package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/network"
)

func newNetworksCmd() *cobra.Command {
	var (
		all    bool
		format string
	)

	cmd := &cobra.Command{
		Use:     "networks",
		Aliases: []string{"net"},
		Short:   "List networks a bastion can join",
		Long: `List Docker networks together with their Compose project and the running
containers attached to them (Compose service names when available).

Docker's built-in networks (bridge, host, none) are hidden unless --all is given.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "table" && format != "json" {
				return fmt.Errorf("unknown format %q (want table or json)", format)
			}
			cli, err := dockerutil.Connect(cmd.Context())
			if err != nil {
				return err
			}
			defer cli.Close()

			infos, err := network.List(cmd.Context(), cli, all)
			if err != nil {
				return fmt.Errorf("list networks: %w", err)
			}

			out := cmd.OutOrStdout()
			if format == "json" {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if infos == nil {
					infos = []network.Info{}
				}
				return enc.Encode(infos)
			}

			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tDRIVER\tPROJECT\tCONTAINERS")
			for _, n := range infos {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.Name, n.Driver, orDash(n.Project), orDash(strings.Join(n.Containers, ", ")))
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "include Docker's built-in networks (bridge, host, none)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp))

	return cmd
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
