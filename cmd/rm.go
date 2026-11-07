package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/managed"
)

func newRmCmd() *cobra.Command {
	var all, force bool

	cmd := &cobra.Command{
		Use:   "rm NAME... | --all",
		Short: "Remove containers started by fumi",
		Long: `Remove containers started by fumi, stopping them if they run. Anyone still
in a removed bastion's shell, or using a removed forward, is disconnected.

Only containers started by fumi can be removed. --all asks for confirmation
unless --force is given.`,
		Example: `  fumi rm fumi-shizuka-tsuki
  fumi rm --all
  fumi rm --all --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all == (len(args) > 0) {
				return errors.New("give container names or --all, not both or neither")
			}
			cli, err := dockerutil.Connect(cmd.Context())
			if err != nil {
				return err
			}
			defer cli.Close()

			var ctrs []managed.Container
			if all {
				ctrs, err = managed.List(cmd.Context(), cli)
			} else {
				ctrs, err = managed.Resolve(cmd.Context(), cli, args)
			}
			if err != nil {
				return err
			}
			if len(ctrs) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No fumi containers to remove")
				return nil
			}

			if all && !force {
				ok, err := confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), ctrs)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aborted")
					return nil
				}
			}

			var errs []error
			for _, c := range ctrs {
				if err := managed.Remove(cmd.Context(), cli, c); err != nil {
					errs = append(errs, err)
					continue
				}
				fmt.Fprintln(cmd.OutOrStdout(), c.Name)
			}
			return errors.Join(errs...)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "remove every container started by fumi")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "do not ask for confirmation with --all")
	cmd.ValidArgsFunction = completeManagedNames

	return cmd
}

// confirm asks before removing ctrs. Without a terminal there is nobody to
// answer, so --force is required instead.
func confirm(in io.Reader, out io.Writer, ctrs []managed.Container) (bool, error) {
	if f, ok := in.(*os.File); !ok || !term.IsTerminal(int(f.Fd())) {
		return false, errors.New("refusing to remove every fumi container without confirmation; pass --force")
	}
	fmt.Fprintf(out, "Remove %d fumi container(s)? Anyone using them is disconnected.\n", len(ctrs))
	for _, c := range ctrs {
		fmt.Fprintf(out, "  %s (%s)\n", c.Name, orDash(c.Kind))
	}
	fmt.Fprint(out, "[y/N] ")
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// completeManagedNames offers fumi's containers not already on the command
// line. Like completeNetworks it stays silent on errors.
func completeManagedNames(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	cli, err := dockerutil.Connect(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer cli.Close()
	names, err := managed.Names(cmd.Context(), cli)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names = slices.DeleteFunc(names, func(n string) bool { return slices.Contains(args, n) })
	return names, cobra.ShellCompDirectiveNoFileComp
}
