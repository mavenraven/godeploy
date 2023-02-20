package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sfreiberg/simplessh"
	"os"
)

func main() {
	port := flag.Int("ssh-port", 22, "port number that sshd is listening on")
	host := flag.String("host", "", "name of host / ip address that sshd is listening on")

	home := os.Getenv("HOME")

	privateKeyPath := flag.String("private-key-path", "", "location of private key used to login, $HOME/.ssh/id_rsa will be used if not set")
	tcpOpenPortList := flag.String("server-listen", "", "tcp ports to open on the remote machine, comma seperated")
	user := flag.String("user", "root", "name of user to use on remote machine")

	flag.Parse()

	if *host == "" {
		fmt.Println("-host is required.")
		os.Exit(1)
	}

	if *privateKeyPath == "" && home == "" {
		fmt.Println("$HOME is not set and no key-location was passed in.")
		os.Exit(1)
	}

	if *privateKeyPath == "" {
		defaultKeyLocation := filepath.Join(home, ".ssh", "id_rsa")
		privateKeyPath = &defaultKeyLocation
	}

	tcpPortsToOpen := getPorts(*tcpOpenPortList)

	socket := fmt.Sprintf("%v:%v", *host, *port)

	var client *simplessh.Client
	var err error

	fmt.Println("attempting to connect as root...")
	client, err = simplessh.ConnectWithKeyFile(socket, *user, *privateKeyPath)
	assertNoErr(err, "unable to establish a connection")
	defer client.Close()
	fmt.Println("connection to remote host established!")

	fmt.Println("clearing existing firewall rules...")
	//ran into this: https://askubuntu.com/a/1293947
	sshCommand(client, "iptables -P INPUT ACCEPT")
	sshCommand(client, "iptables -P OUTPUT ACCEPT")
	sshCommand(client, "iptables -F")
	fmt.Println("all existing firewall rules removed")

	fmt.Println("adding rule to allow ssh...")
	sshCommand(client, "iptables -A INPUT -p tcp --dport ssh -j ACCEPT")
	fmt.Println("rule to allow ssh added")

	for _, port := range tcpPortsToOpen {
		fmt.Printf("adding rule to open tcp port %v\n", port)

		cmd := fmt.Sprintf("iptables -A INPUT -p tcp --dport %v -j ACCEPT", port)
		sshCommand(client, cmd)

		fmt.Printf("rule to open tcp port %v added\n", port)
	}

	fmt.Println("adding rule to deny all other incoming tcp traffic...")

	//a bit of extra safety to ensure the ssh rule is there
	cmd := "iptables -L | grep 'ACCEPT' | grep 'ssh' > /dev/null && sudo iptables -P INPUT DROP"
	sshCommand(client, cmd)

	fmt.Println("rule to deny all other traffic added")

	fmt.Println("updating apt...")
	sshCommand(client, "apt-get update")
	fmt.Println("apt updated")

	installPackage(client, "podman")
	installPackage(client, "curl")

	fmt.Println("downloading pack-cli tarball...")
	sshCommand(client, "curl -m 5 -O -L https://github.com/buildpacks/pack/releases/download/v0.28.0/pack-v0.28.0-linux-arm64.tgz")

	out, err := client.Exec(fmt.Sprintf("sha256sum %v", "pack-v0.28.0-linux-arm64.tgz"))
	assertNoErr(err, "could not get hash of pack-cli tarball")

	if string(out) != "f4940962d1d65b3abcb1996e98cae6497f525999991e9d9dbc7d78a4029d5bb6" {
		fmt.Println("pack-cli tarball corrupt, or someone is doing something sneaky...")
		os.Exit(1)
	}

	fmt.Println("pack-cli tarball downloaded")

	fmt.Println("unpacking pack-cli tarball...")

	fmt.Println("creating tarball...")
	tarballName := createTarball()
	if tarballName == "" {
		fmt.Printf("no files specificed for uploading, nothing left to do")
		os.Exit(0)
	}
	fmt.Println("tarball created")

	remoteTempFileNameBytes, err := client.Exec("mktemp")
	remoteTempFileName := strings.TrimSpace(string(remoteTempFileNameBytes))

	assertNoErr(err, "could not create remote temp file name")
	fmt.Printf("uploading tarball to %v at %v...\n", remoteTempFileName, time.Now().Format("15:04:05"))

	//client.Upload is unusably slow, just shell out instead
	args := make([]string, 0)
	if *privateKeyPath != "" {
		args = append(args, "-i", *privateKeyPath)
	}
	args = append(args, tarballName, fmt.Sprintf("%v@%v:%v", *user, *host, remoteTempFileName))

	scpCmd := exec.Command("scp", args...)
	output, err := scpCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %v\n", string(output), err)
		os.Exit(1)
	}
	fmt.Printf("tarball uploaded at %v\n", time.Now().Format("15:04:05"))

	return
}

func createTarball() string {
	if len(flag.Args()) == 0 {
		return ""
	}

	tarballFile, err := os.CreateTemp("", "")
	assertNoErr(err, "unable to create tarball file")

	fmt.Printf("creating tarball of files to upload: %v...\n", tarballFile.Name())
	defer tarballFile.Close()

	gzipWriter := gzip.NewWriter(tarballFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, filePath := range flag.Args() {
		func() {
			fileToAdd, err := os.Open(filePath)
			assertNoErrF(err, "could not open file to add to tarball %v", filePath)
			defer fileToAdd.Close()

			stat, err := fileToAdd.Stat()
			assertNoErrF(err, "could not get stat of file to add to tarball %v", filePath)

			header := &tar.Header{
				Name:    filePath,
				Size:    stat.Size(),
				Mode:    int64(stat.Mode()),
				ModTime: stat.ModTime(),
			}

			err = tarWriter.WriteHeader(header)
			assertNoErrF(err, "could not write header for tarball: %v", filePath)

			_, err = io.Copy(tarWriter, fileToAdd)
			assertNoErrF(err, "could not copy file to tarball: %v", filePath)
		}()
	}

	return tarballFile.Name()
}

func sshCommand(client *simplessh.Client, command string) {
	sudoPassword := os.Getenv("GODEPLOY_SUDO")

	var output []byte
	var err error

	if sudoPassword == "" {
		output, err = client.Exec(command)
	} else {
		output, err = client.ExecSudo(sudoPassword, command)
	}

	if err != nil {
		fmt.Printf("output of failed command: %v", string(output))
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

}
func assertNoErr(err error, message string) {
	if err != nil {
		fmt.Printf("%v", message)
		fmt.Printf(": %v\n", err)
		os.Exit(1)
	}
}

func assertNoErrF(err error, message string, args string) {
	if err != nil {
		fmt.Printf(message, args)
		fmt.Printf(": %v\n", err)
		os.Exit(1)
	}
}

func installPackage(client *simplessh.Client, packageName string) {
	fmt.Printf("installing %v...\n", packageName)
	sshCommand(client, fmt.Sprintf("apt-get install %v -y", packageName))
	fmt.Printf("%v installed\n", packageName)

}

func getPorts(portStr string) []int {
	if portStr == "" {
		return []int{}
	}

	openPortsStrs := strings.Split(portStr, ",")
	tcpPortsToOpen := make([]int, len(openPortsStrs))
	for i, pStr := range openPortsStrs {
		pInt, err := strconv.Atoi(pStr)
		if err != nil {
			fmt.Printf("could not convert port to integer: %v\n", pStr)
			os.Exit(1)
		}

		if pInt < 0 || pInt > 65535 {
			fmt.Printf("port must be between 0 and 65535, inclusive: %v\n", pInt)
			os.Exit(1)
		}

		tcpPortsToOpen[i] = pInt
	}
	return tcpPortsToOpen
}
