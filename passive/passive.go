// passive.go
package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/fithisux/gopinger/pinglogic"
)

func main() {
	fmt.Println("Hello World!")
	strSourceAddr := flag.String("src", "", "source udp address")
	flag.Parse()
	if (strSourceAddr == nil) || (*strSourceAddr == "") {
		panic("src must not be empty")
	}

	if ServerAddr, err := net.ResolveUDPAddr("udp", *strSourceAddr); err == nil {
		backchannel := make(chan *pinglogic.Backcall)
		go pinglogic.Passive(ServerAddr, backchannel)

		for backcall := range backchannel {
			fmt.Println("received " + backcall.Responder_msg.Msg + " from " + backcall.Responder_url)
		}
	} else {
		panic(err.Error())
	}
}
