package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sfreiberg/simplessh"
	"os"
)

var flags = struct {
	root struct {
		port *int
		host *string
		key  *string
	}
	setup struct {
		tcpPorts *[]int32
	}
}{}

func step(counter *int, beginDesc string, action func()) {
	color.Green("%v. %v...\n", *counter, beginDesc)
	action()
	*counter++
	color.Green("   complete\n")
}
func assertNoErr(err error, message string) {
	if err != nil {
		color.Red("%v", message)
		color.Yellow(": %v\n", err)
		os.Exit(1)
	}
}

func assertNoErrF(err error, message string, args string) {
	if err != nil {
		color.Red(message, args)
		color.Yellow(": %v\n", err)
		os.Exit(1)
	}
}

func sshCommand(client *simplessh.Client, command string) {
	session, err := client.SSHClient.NewSession()
	assertNoErr(err, "could not open session for ssh command")

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// this is total hack, but I don't know why the last line gets cut off
	err = session.Run(command)
	assertNoErr(err, "could not run session for ssh command")

	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func installPackage(counter *int, client *simplessh.Client, packageName string) {
	step(counter, fmt.Sprintf("installing %v", packageName), func() {
		sshCommand(client, fmt.Sprintf("apt-get install %v -y", packageName))
	})
}
