package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func versionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(_ *cobra.Command, _ []string) {
			printVersion(out)
		},
	}

	return cmd
}

func printVersion(out io.Writer) {
	const format = "%-10s %s\n"

	fmt.Fprintf(out, format, "Version:", version)
	fmt.Fprintf(out, format, "Commit:", commit)
	fmt.Fprintf(out, format, "Date:", date)
}
