// test_udp_client.go
package main

import (
	"flag"
	"fmt"
	"github.com/fithisux/gopinger/pinglogic"
	"net"
	"os"
	"strings"
	"time"
	"sync"
)

func main() {
	retries := flag.Int("retries", 5, "retries")
	timeout := flag.Int("timeout", 100, "timeout in ms")
	strDestinationAddr := flag.String("dst", "", "destination udp addresses")
	strSourceAddr := flag.String("src", "", "source udp address")

	flag.Parse()

	if (timeout == nil) || (*timeout <= 0) {
		fmt.Println("Timeout must be positive")
		os.Exit(1)
	}
	if (retries == nil) || (*retries <= 0) {
		fmt.Println("Retries must be positive")
		os.Exit(1)
	}

	attempts := &pinglogic.TimedAttempts{time.Duration(*timeout) *time.Millisecond, *retries}

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
		targets[i], err = net.ResolveUDPAddr("udp", x)
		if err != nil {
			fmt.Println("Error for destinationAddr: ", err)
		}
	}

	sourceAddr, err := net.ResolveUDPAddr("udp", *strSourceAddr)
	if err != nil {
		fmt.Println("Error for sourceAddr: ", err)
	}
	
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); pinglogic.Passive(sourceAddr) }()

	elapsed, ok := pinglogic.Active(attempts, sourceAddr, targets)
	
	pinglogic.StopPassive()
	wg.Wait()

	fmt.Println("After " + elapsed.String())
	if !ok {
		fmt.Println("PING TIMEOUT")
	} else {
		fmt.Println("PING OK")
	}
}
