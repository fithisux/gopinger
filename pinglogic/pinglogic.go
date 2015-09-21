// pinglogic.go
package pinglogic

import (
    "fmt"
    "net"
    "time"
	"os"
	"encoding/json"
	"sync"
	"errors"
)

type TimedAttempts struct {
	Timeout time.Duration
	Retries int
}


type ErrorMesg struct {
	status bool	
	msg string
	writing bool
}
 
type PingMessage struct {
    Msg  	 string `json:"msg"`
    Backcall string `json:"backcall"`
}

var mu sync.Mutex
var running bool = true
var wg sync.WaitGroup


func Passive(strSourceAddr string) {
	ServerAddr, err := net.ResolveUDPAddr("udp", strSourceAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer ServerConn.Close()
	defer wg.Done()
	buf := make([]byte, 1024)
	for {
		n, addr, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err.Error())			
		}
		pingMessage := new(PingMessage)
		err = json.Unmarshal(buf[0:n],pingMessage)
		
		if err != nil {
			fmt.Println("Stray non ping message ", string(buf[0:n]), " from ", addr)
		}
		if (pingMessage.Msg == "PING") {
			dstaddr, err := net.ResolveUDPAddr("udp", pingMessage.Backcall)
			if err !=nil {
				fmt.Println("Stray backcall ping message ", string(buf[0:n]), " from ", addr)
			}
			fmt.Println("Ping received as ", string(buf[0:n]), " from ", addr)
			
			_, err = ServerConn.WriteTo([]byte("PONG"),dstaddr)
			if err != nil {
				fmt.Println(err.Error())				
			}
		} else {
			fmt.Println("Stray ping message ", string(buf[0:n]), " from ", addr)
		}
	}
}

func Active(attempts *TimedAttempts,sourceAddr,destinationAddr *net.UDPAddr) (time.Duration,error) {
	notifications_channel := make(chan bool)
	mesg_channel := make(chan *ErrorMesg)
	startTime := time.Now()
	go writeToDestination(attempts,notifications_channel,mesg_channel,sourceAddr,destinationAddr)
	mesg := <- mesg_channel	
	strerr :=""
	if !mesg.status {
		if(mesg.writing){
			strerr="we failed on writer because "+mesg.msg
		} else {
			strerr="we failed on reader because "+mesg.msg
		}
	} else {
		if(mesg.msg!="OK")	{			
			strerr="we failed on writer because "+mesg.msg
		}
	}	
	close(mesg_channel)
	elapsed := time.Since(startTime)
	if strerr =="" {
		return elapsed,nil
	} else {
		return elapsed,errors.New(strerr)
	}
}

func writeToDestination(attempts *TimedAttempts,notifications_channel chan bool,mesg_channel chan *ErrorMesg,sourceAddr,destinationAddr *net.UDPAddr) {
	Conn, err := net.DialUDP("udp", sourceAddr,destinationAddr)	
	if err  != nil {
        mesg_channel <- &ErrorMesg{false,err.Error(),true}
		return
    }
	defer Conn.Close()
	wg.Add(1)
	go readFromDestination(Conn,notifications_channel,mesg_channel,sourceAddr,destinationAddr)
	counter := attempts.Retries		
	ticker := time.NewTicker(attempts.Timeout * time.Millisecond)	
	for {
		select {
			case b := <- notifications_channel : {
				if b {
					mesg_channel <- &ErrorMesg{true,"OK",true}
					goto Cleanmeup
				}
			}
			case <-ticker.C : {
				fmt.Println(counter)
				if counter == 0 {
					mesg_channel <- &ErrorMesg{true,"TIMEOUT",true}
					goto Cleanmeup
				}
				
				pingMsg := &PingMessage{"PING",sourceAddr.String()}
				xxx,err:=json.Marshal(pingMsg)
				if err != nil {
		            mesg_channel <- &ErrorMesg{false,err.Error(),true}
					goto Cleanmeup
		        }
				
				_,err = Conn.Write(xxx)
		        if err != nil {
					fmt.Println(err.Error())		            
		        }
				counter--
				ticker = time.NewTicker(attempts.Timeout * time.Millisecond)
			}
		}
	}	
	Cleanmeup :
		mu.Lock()
		running=false
		mu.Unlock()
		wg.Wait()
		return	
}

func readFromDestination(ServerConn *net.UDPConn,notifications_channel chan bool,mesg_channel chan *ErrorMesg,sourceAddr,destinationAddr *net.UDPAddr) {
	defer wg.Done()
	buf := make([]byte, 1024)
	for {
		
		goon := true
		mu.Lock()
		goon = running		
		mu.Unlock()
		if !goon {
			close(notifications_channel)
			break
		}
        n,addr,err := ServerConn.ReadFromUDP(buf)
		if err == nil {
            pingMessage := string(buf[0:n])
			if(pingMessage=="PONG") && (addr.String() == destinationAddr.String()) {
				fmt.Println("Received PONG from ",addr)
				notifications_channel <- true						
			} else {
				fmt.Println("Stray ping message ",string(buf[0:n]), " from ",addr)
			}
        }						        
    }
}
