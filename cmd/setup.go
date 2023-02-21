package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
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
	flags.setup.tcpPorts = setupCmd.Flags().Int32SliceP("tcpPorts", "", []int32{}, "A comma seperated list of extra tcp ports to open in your server's firewall")
}

func setup(cmd *cobra.Command, args []string) {
	if *flags.root.key == "" && os.Getenv("HOME") == "" {
		fmt.Println("$HOME is not set and --key was not set.")
		os.Exit(1)
	}

	privateKeyPath := *flags.root.key
	if privateKeyPath == "" {
		privateKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}

	socket := fmt.Sprintf("%v:%v", *flags.root.host, *flags.root.port)
	fmt.Println(socket)

}
