package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/managed"
)

func newLsCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"ps"},
		Short:   "List containers started by fumi",
		Long: `List the bastions (fumi run) and forwards (fumi forward) that are still
around, e.g. because they run in another terminal.`,
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

			ctrs, err := managed.List(cmd.Context(), cli)
			if err != nil {
				return fmt.Errorf("list containers: %w", err)
			}

			out := cmd.OutOrStdout()
			if format == "json" {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if ctrs == nil {
					ctrs = []managed.Container{}
				}
				return enc.Encode(ctrs)
			}

			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tKIND\tNETWORK\tDETAIL\tSTATUS")
			for _, c := range ctrs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", c.Name, orDash(c.Kind), orDash(c.Network), orDash(c.Detail), c.Status)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp))

	return cmd
}
