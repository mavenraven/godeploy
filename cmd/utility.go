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

func safeIdempotentCopyFile(client *simplessh.Client, sourceFilePath, targetFilePath string) {
	_, err := client.Exec(fmt.Sprintf("[ -f \"%v\" ] && [ -f \"%v.finished\" ]", targetFilePath, targetFilePath))
	if err == nil {
		// We don't want clobber the backup once it has been successfully taken, so do nothing.
		fmt.Printf("%v was previously copied\n", sourceFilePath)
		return
	} else {
		assertErrWasDueToNonZeroExitCode(err, "interrupted while checking if target was already copied over successfully")
	}

	_, err = client.Exec(fmt.Sprintf("[ -f \"%v\" ] && ! [ -f \"%v.finished\" ]", targetFilePath, targetFilePath))
	if err != nil {
		assertErrWasDueToNonZeroExitCode(err, "interrupted while checking for corrupted target file")

		// The copy was interrupted before it finished the last time it was run. Remove everything and start over.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v\" ]", targetFilePath))
		assertNoErr(err, "unable to remove corrupt target file")

		// Can't forget to remove this one!
		//
		// If we left the .finished behind, then ran copy again, then were interrupted, we would have a corrupt
		// target file with a .finished file.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v.finished\" ]", targetFilePath))
		assertNoErr(err, "unable to remove corrupt .finished file")
	}

	_, err = client.Exec(fmt.Sprintf("cp \"%v\" \"%v\"", sourceFilePath, targetFilePath))
	assertNoErr(err, "unable to copy file to target")

	_, err = client.Exec(fmt.Sprintf("touch \"%v.finished\"", targetFilePath))
	assertNoErr(err, "unable to create .finished file")
}
