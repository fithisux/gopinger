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

type TimedAttempts struct {
	Timeout time.Duration `json:"timeout"`
	Retries int           `json:"retries"`
}

type PingMessage struct {
	Msg      string `json:"msg"`
	Backcall string `json:"backcall"`
}

type PingMessageChannel struct {
	Pingmessagechannel chan *PingMessage
}

type Multipingresponse struct {
	Targets map[string]*net.UDPAddr
	Answers map[string]int
}

var mutex sync.Mutex
var running bool = true
var Messagechannel *PingMessageChannel = nil

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
		mutex.Lock()
		goon = running
		mutex.Unlock()
		if !goon {
			break
		}
		err = ServerConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
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
		err = json.Unmarshal([]byte(message), pingmessage)
		if err != nil {
			fmt.Println("STRAY " + message)
			continue
		}
		if pingmessage.Msg == "PING" {
			fmt.Println("read some ping")
			pk2 := &PingMessage{"PONG", ServerAddr.String()}
			xxx, err := json.Marshal(pk2)
			if err == nil {
				dstaddr, err := net.ResolveUDPAddr("udp", pingmessage.Backcall)
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
		} else if pingmessage.Msg == "PONG" {
			var temp *PingMessageChannel
			mutex.Lock()
			temp = Messagechannel
			mutex.Unlock()
			if temp != nil {
				temp.Pingmessagechannel <- pingmessage
			}
		} else {
			fmt.Println("STRAY " + message)
		}
	}
}

func Active(timedattempts *TimedAttempts, sourceaddress *net.UDPAddr, targets []*net.UDPAddr) (time.Duration, *Multipingresponse) {
	statuschannel := make(chan *Multipingresponse)
	startTime := time.Now()
	go writeToDestinations(timedattempts, statuschannel, sourceaddress, targets)
	ok := <-statuschannel
	fmt.Println("received " + strconv.Itoa(len(ok.Answers)) + " answers")
	close(statuschannel)
	return time.Since(startTime), ok
}

func writeToDestination(data []byte, destinationaddress *net.UDPAddr) {
	Conn, err := net.DialUDP("udp", nil, destinationaddress)
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

func writeToDestinations(timedattempts *TimedAttempts, statuschannel chan *Multipingresponse, sourceaddress *net.UDPAddr, targets []*net.UDPAddr) {
	fmt.Println("Write to destinations")
	Messagechannel = new(PingMessageChannel)
	Messagechannel.Pingmessagechannel = make(chan *PingMessage)
	responses := new(Multipingresponse)

	responses.Targets = make(map[string]*net.UDPAddr)
	for _, x := range targets {
		responses.Targets[x.String()] = x
	}

	retries := 1
	ticker := time.NewTicker(timedattempts.Timeout)
	fmt.Println("OK1")
	for {
		select {
		case b := <-Messagechannel.Pingmessagechannel:
			{
				if b.Msg == "PONG" {
					delete(responses.Targets, b.Backcall)
					responses.Answers[b.Backcall] = retries
					if len(responses.Targets) == 0 {
						statuschannel <- responses
						goto Cleanmeup
					}
				}
			}
		case <-ticker.C:
			{
				fmt.Println(retries)
				if retries == timedattempts.Retries {
					statuschannel <- responses
					goto Cleanmeup
				}

				pingmessage := &PingMessage{"PING", sourceaddress.String()}
				byterepresentation, err := json.Marshal(pingmessage)
				if err != nil {
					panic(err.Error())
				}
				for _, destinationaddress := range responses.Targets {
					go writeToDestination(byterepresentation, destinationaddress)
				}
				fmt.Println("writing ping from " + sourceaddress.String())
				retries++
				ticker = time.NewTicker(timedattempts.Timeout)
			}
		}
	}

Cleanmeup:
	fmt.Println("Cleanmeup")
	mutex.Lock()
	close(Messagechannel.Pingmessagechannel)
	Messagechannel = nil
	mutex.Unlock()
	return
}

func StopPassive() {
	mutex.Lock()
	running = false
	mutex.Unlock()
}
