package cmd

import (
	"fmt"
	"github.com/sfreiberg/simplessh"
	"github.com/spf13/cobra"
	"time"
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
	socket := fmt.Sprintf("%v:%v", *flags.root.host, *flags.root.port)

	counter := 1
	var client *simplessh.Client
	var err error

	step(&counter, "connecting as root...", func() {
		client, err = simplessh.ConnectWithKeyFileTimeout(socket, "root", *flags.root.key, 5*time.Second)
		assertNoErr(err, "unable to establish a connection")
		defer client.Close()
		fmt.Printf("hi\n")
	})

}
