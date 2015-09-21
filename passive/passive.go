// passive.go
package main

import (
	"flag"
	"fmt"
	"github.com/fithisux/gopinger/pinglogic"
	"net"
	"os"
)

func main() {
	fmt.Println("Hello World!")
	strSourceAddr := flag.String("src", "", "source udp address")
	flag.Parse()
	if (strSourceAddr == nil) || (*strSourceAddr == "") {
		fmt.Println("src must not be empty")
		os.Exit(1)
	}

	ServerAddr, err := net.ResolveUDPAddr("udp", *strSourceAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	listentome(ServerAddr)
}

func listentome(ServerAddr *net.UDPAddr) {
	result_chan := make(chan *pinglogic.PingMessage)
	go pinglogic.Passive(ServerAddr, result_chan)

	for x := range result_chan {
		fmt.Println("received " + x.Msg + " from " + x.Backcall)
	}
}
