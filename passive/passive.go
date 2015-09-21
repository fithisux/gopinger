// passive.go
package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"github.com/fithisux/gopinger/pinglogic"
)

var wg sync.WaitGroup

func main() {
	fmt.Println("Hello World!")
	strSourceAddr := flag.String("src", "", "source udp address")
	flag.Parse()
	if (strSourceAddr == nil) || (*strSourceAddr == "") {
		fmt.Println("src must not be empty")
		os.Exit(1)
	}
	listentome(*strSourceAddr)	
}


func listentome(strSourceAddr string){
	wg.Add(1)
	go pinglogic.Passive(strSourceAddr)
	wg.Wait()
}
