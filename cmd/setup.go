package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sfreiberg/simplessh"
	"github.com/spf13/cobra"
	"os"
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

	step(&counter, "connecting as root", func() {
		client, err = simplessh.ConnectWithKeyFileTimeout(socket, "root", *flags.root.key, 5*time.Second)
		assertNoErr(err, "unable to establish a connection")
	})
	defer client.Close()

	step(&counter, "checking version of server", func() {
		_, err := client.Exec("uname -a | grep 'Linux Ubuntu-2204-jammy-amd64-base'")
		if err != nil {
			color.Red("snakeplant is only supported for https://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/")
			os.Exit(1)
		}
	})

	step(&counter, "updating apt", func() {
		sshCommand(client, "apt-get update")
	})

	step(&counter, "loading firewall rules", func() {
		sshCommand(client, firewallRulesCommand)
	})

	installPackage(&counter, client, "docker")
	installPackage(&counter, client, "curl")

	step(&counter, "persisting firewall rules", func() {
		sshCommand(client, "echo iptables-persistent iptables-persistent/autosave_v4 boolean true | debconf-set-selections")
		sshCommand(client, "echo iptables-persistent iptables-persistent/autosave_v6 boolean true | debconf-set-selections")
		sshCommand(client, "apt-get install iptables-persistent -y")
	})

}

var firewallRulesCommand = `iptables-restore <<-'EOF'
*filter
:INPUT ACCEPT [0:0]
:FORWARD ACCEPT [0:0]
:OUTPUT ACCEPT [0:0]
-A INPUT -m state --state RELATED,ESTABLISHED -j ACCEPT
-A INPUT -p tcp -m state --state NEW -m tcp -m multiport --dports 80,443 -j ACCEPT
-A INPUT -p tcp -m tcp --dport 22 -j ACCEPT
-A INPUT -i lo -j ACCEPT
-A INPUT -j REJECT --reject-with icmp-port-unreachable
COMMIT
EOF`
