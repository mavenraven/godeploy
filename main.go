package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sfreiberg/simplessh"
	"os"
)

func main() {
	port := flag.Int("ssh-port", 22, "port number that sshd is listening on")
	host := flag.String("host", "", "name of host / ip address that sshd is listening on")

	home := os.Getenv("HOME")

	privateKeyPath := flag.String("private-key-path", "", "location of private key used to login, $HOME/.ssh/id_rsa will be used if not set")
	tcpOpenPortList := flag.String("tcp-server-open-ports", "", "tcp ports to open on the remote machine, comma seperated")
	_ = flag.String("server-listen", "", "tcp ports to open on the remote machine, comma seperated")
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

	fmt.Println("installing podman...")
	sshCommand(client, "apt install podman -y")
	fmt.Println("podman installed")

	fmt.Println("creating tarball...")
	tarballName := createTarball()
	if tarballName == "" {
		fmt.Printf("no files specificed for uploading, nothing left to do")
		os.Exit(0)
	}
	fmt.Println("tarball created")

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
			assertNoErrF(err, "could not open file to add to tarball: %v\n", filePath)
			defer fileToAdd.Close()

			stat, err := fileToAdd.Stat()
			assertNoErrF(err, "could not get stat of file to add to tarball: %v\n", filePath)

			header := &tar.Header{
				Name:    filePath,
				Size:    stat.Size(),
				Mode:    int64(stat.Mode()),
				ModTime: stat.ModTime(),
			}

			err = tarWriter.WriteHeader(header)
			assertNoErrF(err, "could not write header for tarball: %v\n", filePath)

			_, err = io.Copy(tarWriter, fileToAdd)
			assertNoErrF(err, "could not copy file to tarball: %v\n", filePath)
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
		fmt.Println(message[0])
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func assertNoErrF(err error, message string, args string) {
	if err != nil {
		fmt.Printf(message, args)
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
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
