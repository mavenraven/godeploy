package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sfreiberg/simplessh"
	"os"
)

func main() {
	port := flag.Int("port", 22, "port number that sshd is listening on")
	host := flag.String("host", "", "name of host / ip address that sshd is listening on")

	home := os.Getenv("HOME")

	privateKeyPath := flag.String("private-key-path", "", "location of private key used to login, $HOME/.ssh/id_rsa will be used if not set")
	tcpOpenPortList := flag.String("tcp-open-ports", "", "tcp ports to open on the remote machine, comma seperated")

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

	openPortsStrs := strings.Split(*tcpOpenPortList, ",")
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

	socket := fmt.Sprintf("%v:%v", *host, *port)

	var client *simplessh.Client
	var err error

	fmt.Println("attempting to connect as root...")

	client, err = simplessh.ConnectWithKeyFile(socket, "root", *privateKeyPath)
	if err != nil {
		fmt.Println("unable to establish a connection")
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("connection to remote host established!")
	fmt.Println("clearing existing firewall rules...")

	_, err = client.Exec("iptables -F")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	fmt.Println("all existing firewall rules removed")
	fmt.Println("adding rule to allow ssh...")

	_, err = client.Exec("iptables -A INPUT -p tcp --dport ssh -j ACCEPT")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	fmt.Println("rule to allow ssh added")

	for _, port := range tcpPortsToOpen {
		fmt.Printf("adding rule to open tcp port %v\n", port)

		cmd := fmt.Sprintf("iptables -A INPUT -p tcp --dport %v -j ACCEPT", port)

		if _, err := client.Exec(cmd); err != nil {
			log.Println(err)
			os.Exit(1)
		}

		fmt.Printf("rule to open tcp port %v added\n", port)
	}

	return
}
