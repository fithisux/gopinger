// pinglogic.go
package pinglogic

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type PingConf struct {
	Backaddress *net.UDPAddr
	Backchannel <-chan *Backcall
	Timingconf  *TimedAttempts
}

type TimedAttempts struct {
	Timeout time.Duration `json:"timeout"`
	Retries int           `json:"retries"`
}

type TargetedPing struct {
	ServerAddr  *net.UDPAddr
	Startoftime time.Time
}

type PingMessage struct {
	Msg        string    `json:"msg"`
	Callmeback string    `json:"callmeback"`
	Timestamp  time.Time `json:"timestamp"`
}

type Backcall struct {
	Responder_msg *PingMessage
	Responder_url string
}

type Multipingresponse struct {
	Targets map[string]*TargetedPing
	Answers map[string]*TimedAttempts
}

var mutex sync.Mutex
var running bool = true

func StopPassive() {
	mutex.Lock()
	running = false
	mutex.Unlock()
}

func Passive(ServerAddr *net.UDPAddr, backchannel chan<- *Backcall) {

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer ServerConn.Close()
	buf := make([]byte, 1024)

	for {
		goon := true
		mutex.Lock()
		goon = running
		mutex.Unlock()
		if !goon {
			break
		}

		if err := ServerConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			panic(err)
		}

		numbytes, addr, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			//fmt.Println(err.Error())
			continue
		}

		message := string(buf[0:numbytes])
		fmt.Println("Message ", message, " from ", addr)
		pingmessage := new(PingMessage)

		if err := json.Unmarshal([]byte(message), pingmessage); err != nil {
			fmt.Println("STRAY " + message)
			continue
		}
		if pingmessage.Msg == "PING" {
			fmt.Println("read some ping")
			pk2 := &PingMessage{"PONG", pingmessage.Callmeback, pingmessage.Timestamp}

			if xxx, err := json.Marshal(pk2); err == nil {
				if dstaddr, err := net.ResolveUDPAddr("udp", pingmessage.Callmeback); err == nil {
					if _, err := ServerConn.WriteTo(xxx, dstaddr); err != nil {
						fmt.Println(err.Error())
					}
				} else {
					fmt.Println(err.Error())
				}
			} else {
				panic(err.Error())
			}
		} else if pingmessage.Msg == "PONG" {
			backchannel <- &Backcall{pingmessage, addr.String()}
		} else {
			fmt.Println("STRAY " + message)
		}
	}
}

func Active(pingconf *PingConf, targets []*net.UDPAddr) (time.Duration, *Multipingresponse) {

	statuschannel := make(chan *Multipingresponse)
	startTime := time.Now()
	go writeToDestinations(pingconf, statuschannel, targets)
	ok := <-statuschannel
	fmt.Println("received " + strconv.Itoa(len(ok.Answers)) + " answers")
	close(statuschannel)
	return time.Since(startTime), ok
}

func writeToDestination(data []byte, destinationaddress *net.UDPAddr) {
	if Conn, err := net.DialUDP("udp", nil, destinationaddress); err != nil {
		fmt.Println("ZOO1")
		fmt.Println(err.Error())
		return
	} else {
		defer Conn.Close()
		if _, err := Conn.Write(data); err != nil {
			fmt.Println(err.Error())
		}
	}
}

func writeToDestinations(pingconf *PingConf, statuschannel chan<- *Multipingresponse, targets []*net.UDPAddr) {
	fmt.Println("Write to destinations")

	responses := new(Multipingresponse)
	responses.Targets = make(map[string]*TargetedPing)
	responses.Answers = make(map[string]*TimedAttempts)

	localtime := time.Now()
	for _, x := range targets {
		responses.Targets[x.String()] = &TargetedPing{x, localtime}
	}

	retries := 0
	var ticker *time.Ticker

	sendmessage := func() {
		pingmessage := &PingMessage{"PING", pingconf.Backaddress.String(), time.Now()}
		if byterepresentation, err := json.Marshal(pingmessage); err == nil {
			for _, targetedpings := range responses.Targets {
				fmt.Println("writing ping to " + targetedpings.ServerAddr.String())
				go writeToDestination(byterepresentation, targetedpings.ServerAddr)
			}
			retries++
			ticker = time.NewTicker(pingconf.Timingconf.Timeout)
		} else {
			panic(err.Error())
		}
	}

	fmt.Println("OK1")
	sendmessage()
	for {
		select {
		case b := <-pingconf.Backchannel:
			{
				if b.Responder_msg.Msg == "PONG" {
					if b.Responder_msg.Timestamp.After(responses.Targets[b.Responder_url].Startoftime) {
						delete(responses.Targets, b.Responder_url)
						responses.Answers[b.Responder_url] = &TimedAttempts{time.Since(b.Responder_msg.Timestamp), retries}
						if len(responses.Targets) == 0 {
							statuschannel <- responses
							return
						}
					} else {
						fmt.Println("Ignored from past")
					}
				}
			}
		case <-ticker.C:
			{
				fmt.Println(retries)
				if retries == pingconf.Timingconf.Retries {
					statuschannel <- responses
					return
				} else {
					sendmessage()
				}
			}
		}
	}
}
