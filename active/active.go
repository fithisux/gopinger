// test_udp_client.go
package main
 
import (
    "fmt"
    "net"
    "time"
	"flag"
	"os"
	"github.com/fithisux/gopinger/pinglogic"
)


func main() {
	retries := flag.Int("retries", 5, "retries")
    timeout := flag.Int("timeout", 100, "timeout in ms")
	strDestinationAddr := flag.String("dst", "", "destination udp address")
	strSourceAddr := flag.String("src", "", "source udp address")
	
	flag.Parse()
	
	if (timeout==nil) || (*timeout <= 0) {
		fmt.Println("Timeout must be positive")
		os.Exit(1)
	}
	if (retries==nil) || (*retries <= 0) {
		fmt.Println("Retries must be positive")
		os.Exit(1)
	}
	
	attempts := &pinglogic.TimedAttempts{time.Duration(*timeout),*retries}
	
	if((strDestinationAddr==nil)|| (*strDestinationAddr=="")){
		fmt.Println("dst must not be empty")
		os.Exit(1)
	}
	
	if((strSourceAddr==nil)|| (*strSourceAddr=="")){
		fmt.Println("src must not be empty")
		os.Exit(1)
	}
		
    destinationAddr,err := net.ResolveUDPAddr("udp",*strDestinationAddr)
    if err  != nil {
        fmt.Println("Error for destinationAddr: " , err)
    }
	
    sourceAddr,err := net.ResolveUDPAddr("udp",*strSourceAddr)
    if err  != nil {
        fmt.Println("Error for sourceAddr: " , err)
    } 
	
	elapsed,err := pinglogic.Active(attempts,sourceAddr,destinationAddr)
	
	if err != nil {
		fmt.Println("After "+elapsed.String()+" we have error : "+err.Error())
	} else {
		fmt.Println("After "+elapsed.String()+" pinged normally")
	}
}