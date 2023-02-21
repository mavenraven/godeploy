package cmd

import (
	"github.com/spf13/cobra"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up all the programs needed to deploy on your server.",
	Long:  `'setup' is designed to be idempotent. This means that it's always safe to run it again, even if it errors out.'`,
	Run:   setup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().Int32SliceP("tcp-ports", "", []int32{}, "A comma seperated list of extra tcp ports to open in your server's firewall")
}

func setup(cmd *cobra.Command, args []string) {

}
