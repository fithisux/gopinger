// test_udp_client.go
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fithisux/gopinger/pinglogic"
)

func main() {
	retries := flag.Int("retries", 5, "retries")
	timeout := flag.Int("timeout", 100, "timeout in ms")
	strDestinationAddr := flag.String("dst", "", "destination udp urls")
	strSourceAddr := flag.String("src", "", "source udp url to callback")

	flag.Parse()

	if (timeout == nil) || (*timeout <= 0) {
		fmt.Println("Timeout must be positive")
		os.Exit(1)
	}
	if (retries == nil) || (*retries <= 0) {
		fmt.Println("Retries must be positive")
		os.Exit(1)
	}

	attempts := &pinglogic.TimedAttempts{time.Duration(*timeout) * time.Millisecond, *retries}

	if (strDestinationAddr == nil) || (*strDestinationAddr == "") {
		fmt.Println("dst must not be empty")
		os.Exit(1)
	}

	if (strSourceAddr == nil) || (*strSourceAddr == "") {
		fmt.Println("src must not be empty")
		os.Exit(1)
	}

	strTargets := strings.Split(*strDestinationAddr, ";")
	targets := make([]*net.UDPAddr, len(strTargets))

	var err error
	for i, x := range strTargets {
		if targets[i], err = net.ResolveUDPAddr("udp", x); err != nil {
			panic(err)
		}
	}

	backaddress, err := net.ResolveUDPAddr("udp", *strSourceAddr)
	if err != nil {
		fmt.Println("Error for backadress: ", err)
	}

	backchannel := make(chan *pinglogic.Backcall)
	pingconf := &pinglogic.PingConf{backaddress, backchannel, attempts}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); pinglogic.Passive(backaddress, backchannel) }()
	elapsed, _ := pinglogic.Active(pingconf, targets)
	fmt.Println("After " + elapsed.String())
	pinglogic.StopPassive()
	wg.Wait()
	close(backchannel)
}
