package tarballs

import (
	"fmt"
	"github.com/mavenraven/snakeplant/cmd"

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
}

func init() {
	cmd.RootCmd.AddCommand(rootTarballsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rootTarballsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rootTarballsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
