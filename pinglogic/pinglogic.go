// pinglogic.go
package pinglogic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type TimedAttempts struct {
	Timeout time.Duration
	Retries int
}

type ErrorMesg struct {
	status  bool
	msg     string
	writing bool
}

type PingMessage struct {
	Msg      string `json:"msg"`
	Backcall string `json:"backcall"`
}

type PingMessageChannel struct {
	Mychannel chan *PingMessage
}

var mu sync.Mutex
var running bool = true
var Messagechannel *PingMessageChannel=nil

func Passive(ServerAddr *net.UDPAddr) {	

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer ServerConn.Close()
	buf := make([]byte, 1024)
	
	for {
		goon := true
		mu.Lock()
		goon = running
		mu.Unlock()
		if !goon {
			break
		}		
		err = ServerConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			panic(err)
		}
		n, addr, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			//fmt.Println(err.Error())
			continue
		}

		message := string(buf[0:n])
		fmt.Println("Message ", message, " from ", addr)

		pk1 := new(PingMessage)
		err = json.Unmarshal(buf[0:n], pk1)
		if err != nil {
			fmt.Println("STRAY " + message)
			continue
		}
		if pk1.Msg == "PING" {
			fmt.Println("read some ping")
			pk2 := &PingMessage{"PONG", ServerAddr.String()}
			xxx, err := json.Marshal(pk2)
			if err == nil {
				dstaddr, err := net.ResolveUDPAddr("udp", pk1.Backcall)
				if err == nil {
					_, err = ServerConn.WriteTo(xxx, dstaddr)
					if err != nil {
						fmt.Println(err.Error())
					}
				} else {
					fmt.Println(err.Error())
				}
			} else {
				panic(err.Error())
			}
		} else if pk1.Msg == "PONG" {
			var temp *PingMessageChannel
			mu.Lock()
			temp = Messagechannel
			mu.Unlock()			
			if temp != nil {
				temp.Mychannel <- pk1				
			}
		} else {
			fmt.Println("STRAY " + message)
		}
	}
}

func Active(attempts *TimedAttempts, inAddr *net.UDPAddr, targets []*net.UDPAddr) (time.Duration, error) {
	mesg_channel := make(chan *ErrorMesg)
	startTime := time.Now()
	go writeToDestinations(attempts, mesg_channel, inAddr, targets)
	mesg := <-mesg_channel
	strerr := ""
	if !mesg.status {
		strerr = "we failed because " + mesg.msg
	} else {
		if mesg.msg != "OK" {
			strerr = "we failed on writer because " + mesg.msg
		}
	}
	close(mesg_channel)
	elapsed := time.Since(startTime)
	if strerr == "" {
		return elapsed, nil
	} else {
		return elapsed, errors.New(strerr)
	}
}

func writeToDestination(data []byte, dstAddr *net.UDPAddr) {
	Conn, err := net.DialUDP("udp", nil, dstAddr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer Conn.Close()
	_, err = Conn.Write(data)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func writeToDestinations(attempts *TimedAttempts, mesg_channel chan *ErrorMesg, inAddr *net.UDPAddr, targets []*net.UDPAddr) {	
	fmt.Println("Write to destinations")
	Messagechannel = new(PingMessageChannel)
	Messagechannel.Mychannel = make(chan *PingMessage)
	
	cc := Messagechannel.Mychannel
	mymap := make(map[string]*net.UDPAddr)
	for _, x := range targets {
		mymap[x.String()] = x
	}

	counter := attempts.Retries
	ticker := time.NewTicker(attempts.Timeout)
	fmt.Println("OK1")
	for {
		select {
		case b := <-cc:
			{
				if b.Msg == "PONG" {
					delete(mymap, b.Backcall)
					if len(mymap) == 0 {
						mesg_channel <- &ErrorMesg{true, "OK", true}
						goto Cleanmeup
					}
				}
			}
		case <-ticker.C:
			{
				fmt.Println(counter)
				if counter == 0 {
					mesg_channel <- &ErrorMesg{true, "TIMEOUT", true}
					goto Cleanmeup
				}

				pingMsg := &PingMessage{"PING", inAddr.String()}
				xxx, err := json.Marshal(pingMsg)
				if err != nil {
					mesg_channel <- &ErrorMesg{false, err.Error(), true}
					goto Cleanmeup
				}
				for _, value := range mymap {
					go writeToDestination(xxx, value)
				}
				fmt.Println("writing ping")
				counter--
				ticker = time.NewTicker(attempts.Timeout * time.Millisecond)
			}
		}
	}
Cleanmeup:
	mu.Lock()
	close(Messagechannel.Mychannel)
	Messagechannel=nil
	mu.Unlock()
	return
}

func StopPassive(){
	mu.Lock()
	running = false
	mu.Unlock()
}
