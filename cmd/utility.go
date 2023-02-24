package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sfreiberg/simplessh"
	"golang.org/x/crypto/ssh"
	"os"
	"strconv"
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

const CARRIAGE_RETURN = 13
const LINE_PADDING = "    "

// My general philosophy of whether to print output of stuff inside a step is that if it's over the network,
// like APT or curl, just print everything. Otherwise, skip it.
func step(counter *int, beginDesc string, action func()) {
	numSize := len(strconv.FormatInt(int64(*counter), 10))

	var padding string
	if numSize == 1 {
		padding = "  "
	} else {
		padding = " "
	}

	color.HiGreen("%v.%v%v...\n", *counter, padding, beginDesc)
	action()

	if numSize == 1 {
		padding = "  "
	} else {
		padding = "  "
	}
	color.HiGreen("%v  complete.\n", padding)
	*counter++
}

func printDiffHeader() {
	printSubStepInformation(fmt.Sprintf("%vDiff of changes. Left of the '|' is before the file was changed, right of the '|' is after.", LINE_PADDING))
}

func printSubStepInformation(message string) {
	color.Cyan(message)
}
func assertNoErr(err error, message string) {
	if err != nil {
		color.Red("%v%v\n", LINE_PADDING, err)
		printMessageAndQuit(message)
	}
}

func printMessageAndQuit(message string) {
	color.HiRed("%v%v", LINE_PADDING, message)
	os.Exit(1)
}

type LineHolder struct {
	lineCallback func([]byte)
	buffer       []byte
}

func (f *LineHolder) Write(p []byte) (n int, err error) {
	for _, c := range p {
		if c == '\n' {
			f.lineCallback(f.buffer)
			f.buffer = make([]byte, 0, 100)
			continue
		}
		f.buffer = append(f.buffer, c)
	}

	return len(p), nil
}

func sshCommand(client *simplessh.Client, command string) {
	session, err := client.SSHClient.NewSession()
	assertNoErr(err, "Could not open session for running an ssh command.")

	session.Stdout = &LineHolder{buffer: make([]byte, 0, 100), lineCallback: func(bytes []byte) {
		fmt.Printf("%v%v\n", LINE_PADDING, string(bytes))
	}}

	session.Stderr = &LineHolder{buffer: make([]byte, 0, 100), lineCallback: func(bytes []byte) {
		color.Red("%v%v\n", LINE_PADDING, string(bytes))
	}}

	err = session.Run(command)

	if err != nil {
		color.HiRed("    %v\n", err)
		os.Exit(1)
	}
}

type CurlWriter struct {
	buffer []byte
}

func isAnAsciiNumber(c byte) bool {
	return c >= 48 && c <= 57
}

func (f *CurlWriter) Write(p []byte) (n int, err error) {
	for _, c := range p {
		if isAnAsciiNumber(c) {
			f.buffer = append(f.buffer, c)
			continue
		}
		if c == '%' {
			f.buffer = append(f.buffer[:len(f.buffer)-1], '.', f.buffer[len(f.buffer)-1])
			f.buffer = append(f.buffer, c)
			fmt.Printf("%c%vDownload progress: %v", CARRIAGE_RETURN, LINE_PADDING, string(f.buffer))
			f.buffer = make([]byte, 0, 5)
			continue
		}
		if c == '\n' {
			fmt.Println()
		}
	}
	return len(p), nil
}

// Mainly needed because curl output is very non-standard.
func curlCommand(client *simplessh.Client, command string) {
	session, err := client.SSHClient.NewSession()
	assertNoErr(err, "Could not open session for running a curl command over ssh.")

	session.Stderr = &CurlWriter{buffer: make([]byte, 0, 5)}

	err = session.Run(fmt.Sprintf("curl %v", command))

	if err != nil {
		color.HiRed("    %v\n", err)
		os.Exit(1)
	}
}

func installPackage(counter *int, client *simplessh.Client, packageName string) {
	step(counter, fmt.Sprintf("Installing %v", packageName), func() {
		// We don't want to run install on subsequent runs as that could cause the package to update and cause a broken system.
		// See https://serverfault.com/a/670688
		_, err := client.Exec(fmt.Sprintf("DEBIAN_FRONTEND=noninteractive dpkg -l %v", packageName))
		assertAnyErrWasDueToNonZeroExitCode(err, fmt.Sprintf("%v'dpkg' listing was interrupted.", LINE_PADDING))

		if err == nil {
			printSubStepInformation(fmt.Sprintf("%v'%v' package was previously installed.\n", LINE_PADDING, packageName))
			return
		}

		sshCommand(client, fmt.Sprintf("apt-get install %v -y", packageName))
	})
}

func assertAnyErrWasDueToNonZeroExitCode(err error, message string) {
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
	assertAnyErrWasDueToNonZeroExitCode(err, "Interrupted while checking if target was already copied over successfully.")

	if err == nil {
		return
	}

	_, err = client.Exec(fmt.Sprintf("[ -f \"%v\" ] && ! [ -f \"%v.finished\" ]", targetFilePath, targetFilePath))
	assertAnyErrWasDueToNonZeroExitCode(err, "Interrupted while checking for corrupted target file.")

	if err != nil {
		// The copy was interrupted before it finished the last time it was run. Remove everything and start over.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v\" ]", targetFilePath))
		assertNoErr(err, "Unable to remove corrupt target file.")

		// Can't forget to remove this one!
		//
		// If we left the .finished behind, then ran copy again, then were interrupted, we would have a corrupt
		// target file with a .finished file.
		_, err = client.Exec(fmt.Sprintf("rm -f  \"%v.finished\" ]", targetFilePath))
		assertNoErr(err, "Unable to remove corrupt .finished file.")
	}

	_, err = client.Exec(fmt.Sprintf("cp \"%v\" \"%v\"", sourceFilePath, targetFilePath))
	assertNoErr(err, "Unable to copy file to target.")

	_, err = client.Exec(fmt.Sprintf("touch \"%v.finished\"", targetFilePath))
	assertNoErr(err, "Unable to create .finished file.")
}
