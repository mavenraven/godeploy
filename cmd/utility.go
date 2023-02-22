package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sfreiberg/simplessh"
	"golang.org/x/crypto/ssh"
	"os"
)

var flags = struct {
	root struct {
		port *int
		host *string
		key  *string
	}
	setup struct {
		tcpPorts   *[]int32
		rebootTime *string
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

func assertErrWasDueToNonZeroExitCode(err error, message string) {
	if exitErr, ok := err.(*ssh.ExitError); ok {
		// I spent an hour looking into this because I wasn't sure if it was right. The following is correct (probably ¯\_(ツ)_/¯),
		// but the API for `ExitError` is EXTREMELY esoteric.
		//
		// POSIX exit codes can only be 0-255. The code in session.go that returns an ExitError uses everything >= 128
		// for returning a signal. But that code path will also have some value filled in for Signal().
		//
		// So if Signal() is blank, then we have a "real" non-zero exit code, even if it's >= 128.

		if exitErr.Signal() == "" {
			return
		}
	}
	assertNoErr(err, message)
}

func safeBackupFile(client *simplessh.Client, filepath string) {
	_, err := client.Exec(fmt.Sprintf("[ -f \"%v.bak\" ] && [ -f \"%v.bak.finished\" ]", filepath, filepath))
	if err == nil {
		// We don't want clobber the backup once it has been successfully taken, so do nothing.
		return
	} else {
		fmt.Printf("nothing to do")
		assertErrWasDueToNonZeroExitCode(err, "interrupted while checking if backup already completed successfully")
	}

	_, err = client.Exec(fmt.Sprintf("[ -f \"%v.bak\" ] && ! [ -f \"%v.bak.finished\" ]", filepath, filepath))
	if err != nil {
		assertErrWasDueToNonZeroExitCode(err, "interrupted while checking for corrupted .bak file")

		// The backup was interrupted before it finished the last time it was run. Remove everything and start over.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v.bak\" ]", filepath))
		assertNoErr(err, "unable to remove corrupt bak file")

		// Can't forget to remove this one! If we didn't and the copy got interrupted, we would still have a .finished
		// file, and we could corrupt our data.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v.bak.finished\" ]", filepath))
		assertNoErr(err, "unable to remove corrupt bak.finished file")
	}

	_, err = client.Exec(fmt.Sprintf("cp \"%v\" \"%v.bak\"", filepath, filepath))
	assertNoErr(err, "unable to copy file to bak")

	_, err = client.Exec(fmt.Sprintf("touch \"%v.bak.finished\"", filepath))
	assertNoErr(err, "unable to copy file to bak")
}
