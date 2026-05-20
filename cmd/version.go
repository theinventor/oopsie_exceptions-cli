package cmd

import "github.com/spf13/cobra"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version metadata as JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printJSON(cmd.OutOrStdout(), map[string]string{
				"version":    Version,
				"commit":     Commit,
				"build_date": BuildDate,
			})
		},
	}
}
