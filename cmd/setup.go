package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sfreiberg/simplessh"
	"github.com/spf13/cobra"
	"os"
	"strings"
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
	flags.setup.rebootTime = setupCmd.Flags().StringP("rebootTime", "", "", "Time to reboot your server for security updates that require a reboot. An example value is 02:00 for 2 AM. Remember that your server might be in a different timezone than you!")
	setupCmd.MarkFlagRequired("rebootTime")
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

	step(&counter, "checking OS version of server", func() {
		output, err := client.Exec("uname -a")
		assertNoErr(err, "could not get os version")

		if !strings.Contains(string(output), "Linux Ubuntu-2204-jammy-amd64-base") {
			color.Red("snakeplant is only supported for https://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/")
			os.Exit(1)
		}
	})

	step(&counter, "checking architecture of server", func() {
		output, err := client.Exec("uname -p")
		assertNoErr(err, "could not get architecture")

		if !strings.Contains(string(output), "x86_64") {
			color.Red("snakeplant is only supported on x86_64")
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

	installPackage(&counter, client, "unattended-upgrades")

	step(&counter, "setting up automatic reboots", func() {
		out, err := client.Exec("mktemp")
		if err != nil {
			color.Red("could not create temp file")
			os.Exit(1)

		}

		tempFile := strings.TrimSpace(string(out))

		fmt.Println(tempFile)

		sshCommand(client, fmt.Sprintf("cp /etc/apt/apt.conf.d/50unattended-upgrades %v", tempFile))
		sshCommand(client, fmt.Sprintf("[ -f /etc/apt/apt.conf.d/50unattended-upgrades.bak ] || cp /etc/apt/apt.conf.d/50unattended-upgrades /etc/apt/apt.conf.d/50unattended-upgrades.bak"))

		sshCommand(client, fmt.Sprintf("sed -i 's|.*Unattended-Upgrade::Automatic-Reboot \"false\".*|Unattended-Upgrade::Automatic-Reboot \"true\";|'  %v", tempFile))
		sshCommand(client, fmt.Sprintf("sed -i 's|.*Unattended-Upgrade::Automatic-Reboot-WithUsers \"true\".*|Unattended-Upgrade::Automatic-Reboot-WithUsers \"true\";|'  %v", tempFile))
		sshCommand(client, fmt.Sprintf("sed -i 's|.*Unattended-Upgrade::Automatic-Reboot-Time \"02:00\".*|Unattended-Upgrade::Automatic-Reboot-Time \"%v\";|'  %v", *flags.setup.rebootTime, tempFile))

		sshCommand(client, fmt.Sprintf("sed -i 's|.*Unattended-Upgrade::SyslogEnable \"false\".*|Unattended-Upgrade::SyslogEnable \"true\";|' %v", tempFile))
		sshCommand(client, fmt.Sprintf("sed -i 's|.*Unattended-Upgrade::Verbose \"false\".*|Unattended-Upgrade::Verbose \"true\";|' %v", tempFile))

		sshCommand(client, fmt.Sprintf("mv %v /etc/apt/apt.conf.d/50unattended-upgrades", tempFile))
		sshCommand(client, fmt.Sprintf("echo 'automatic upgrade changes: '; diff /etc/apt/apt.conf.d/50unattended-upgrades.bak /etc/apt/apt.conf.d/50unattended-upgrades || true"))
	})

	step(&counter, "installing pack", func() {
		fileName := "pack-v0.28.0-linux.tgz"
		sshCommand(client, "curl -m 5 -O -L https://github.com/buildpacks/pack/releases/download/v0.28.0/pack-v0.28.0-linux.tgz")

		out, err := client.Exec(fmt.Sprintf("sha256sum %v | awk '{print $1}'", fileName))
		assertNoErr(err, "could not get hash of pack-cli tarball")
		fmt.Printf(string(out))

		if strings.TrimSpace(string(out)) != "4f51b82dea355cffc62b7588a2dfa461e26621dda3821034830702e5cae6f587" {
			fmt.Println("pack-cli tarball corrupt, or someone is doing something sneaky...")
			os.Exit(1)
		}

		sshCommand(client, fmt.Sprintf("tar xvf %v", fileName))
		sshCommand(client, "mv pack /usr/local/bin/pack")
		sshCommand(client, "chmod +x /usr/local/bin/pack")
	})

	color.Green("Setup is complete. You're now ready to deploy!")
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
