package main

import (
	"bufio"
	"fmt"

	"time"

	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gbatanov/gsm/modem"
	"github.com/matishsiao/goInfo"
)

const VERSION = "0.3.14"
const PORT = "/dev/tty.usbserial-A50285BI"

func main() {
	log.Println("Start")
	Flag := true

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Введите номер телефона в формате 1234567890, без + и 7. Пробелы не допускаются")
		return
	}
	phoneNumber := args[0]

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
		mdm := modem.GsmModemCreate(PORT, oss, 9600, phoneNumber)
		err := mdm.Open()
		//		var err error = nil
		if err == nil {
			defer mdm.Stop()
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				for Flag {
					reader := bufio.NewReader(os.Stdin)
					text, _ := reader.ReadString('\n')
					if len(text) > 1 { // text включает завершающий \n
						// log.Printf("%v", []byte(text))
						switch text {
						case "q\n":
							Flag = false
						case "balance\n":
							mdm.GetBalance()
						case "sms\n":
							mdm.SendSms("Ёлки-палки 2023 USSR")
						case "call\n":
							mdm.CallMain() // звонок на основной номер
						case "up\n":
							mdm.HangUp() // поднять трубку
						case "down\n":
							mdm.HangOut() // сбросить звонок
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
