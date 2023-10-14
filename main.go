package main

import (
	"bufio"
	"gsm/modem"
	"time"

	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/matishsiao/goInfo"
)

const VERSION = "0.2.6"
const PORT = "/dev/tty.usbserial-A50285BI"

func main() {
	log.Println("Start")
	Flag := true

	gi, _ := goInfo.GetInfo()
	oss := gi.GoOS

	sigs := make(chan os.Signal, 1)
	//	intrpt := false // for gracefull exit
	// signal.Notify registers this channel to receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// This goroutine performs signal blocking.
	// When goroutine receives signal, it prints signal name out and then notifies the program that it can be terminated.
	go func() {
		sig := <-sigs
		log.Println(sig)
		Flag = false
		//		intrpt = true
	}()

	if Flag {
		mdm := modem.GsmModemCreate(PORT, oss)
		err := mdm.Open(9600)
		//		var err error = nil
		if err == nil {
			defer mdm.Stop()
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				for Flag {
					reader := bufio.NewReader(os.Stdin)
					text, _ := reader.ReadString('\n')
					if len(text) > 0 {
						switch []byte(text)[0] {
						case 'q':
							Flag = false
						case 'b':
							mdm.GetBalance()
						case 's':
							mdm.SendSms("Царь-надёжа OK")
						} //switch
					} else {
						time.Sleep(time.Second * 3)
					}
				} //for
				wg.Done()
			}()
			wg.Wait()
		}
		Flag = false
	}

	log.Println("End")

}
