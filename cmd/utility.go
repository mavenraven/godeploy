package cmd

import (
	"fmt"
	"github.com/fatih/color"
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
	color.Green("%v. %v\n", *counter, beginDesc)
	action()
	*counter++
	color.Green("   success!\n")
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
