package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/sfreiberg/simplessh"
	"os"
)

func main() {
	user := flag.String("user", "root", "user name to use for ssh (probably root)")
	port := flag.Int("port", 22, "port number that sshd is listening on")
	host := flag.String("host", "", "name of host / ip address that sshd is listening on")

	home := os.Getenv("HOME")

	privateKeyPath := flag.String("private-key-path", "", "location of private key used to login, $HOME/.ssh/id_rsa will be used if not set")

	flag.Parse()

	if *host == "" {
		fmt.Printf("-host is required.")
		os.Exit(1)
	}

	if *privateKeyPath == "" && home == "" {
		fmt.Printf("$HOME is not set and no key-location was passed in.")
		os.Exit(1)
	}

	if *privateKeyPath == "" {
		defaultKeyLocation := filepath.Join(home, ".ssh", "id_rsa")
		privateKeyPath = &defaultKeyLocation
	}

	socket := fmt.Sprintf("%v:%v", *host, *port)

	var client *simplessh.Client
	var err error

	if client, err = simplessh.ConnectWithKeyFile(socket, *user, *privateKeyPath); err != nil {
		fmt.Printf("unable to establish a connection")
		os.Exit(1)
	}

	defer client.Close()

	// Now run the commands on the remote machine:
	if _, err := client.Exec("cat /tmp/somefile"); err != nil {
		log.Println(err)
	}

	return
}
