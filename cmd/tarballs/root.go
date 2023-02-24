package tarballs

import (
	"fmt"
	"github.com/mavenraven/snakeplant/cmd"
	"os"

	"github.com/spf13/cobra"
)

var rootTarballsCmd = &cobra.Command{
	Use:   "tarballs",
	Short: "Allows you to interact with the source releases (\"tarballs\") on your server.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tarballs called")
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
}

func init() {
	cmd.RootCmd.AddCommand(rootTarballsCmd)
}
