package cmd

import "github.com/fatih/color"

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

func Step(counter *int, desc string, action func()) {
	color.Green("%v. [%v]\n", *counter, desc)
	action()
	*counter++
}
