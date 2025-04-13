package main

import (
	"bufio"
	"context"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/gbatanov/sim800l/modem"
)

type Config struct {
	MyPhoneNumber string
	ModemPort     string
}

func main() {
	ctx := context.Background()
	cmdPtr := flag.String("p", "", "phone number without +7 (for Russia)")
	flag.Parse()

	var config Config
	config.MyPhoneNumber = *cmdPtr
	config.ModemPort = "/dev/tty.usbserial-A50285BI"
	mdm := modem.GsmModemCreate(config.ModemPort, 9600, config.MyPhoneNumber)
	err := mdm.Open(ctx)
	if err != nil {
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	// получение команд из консоли
	go func() {
		defer wg.Done()
		for {
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')

			if len(text) > 0 {
				switch []byte(text)[0] {
				case 'q':
					return
				case 'b':
					mdm.GetBalance()
				} //switch
			}
			time.Sleep(1 * time.Second)
		} //for

	}()
	wg.Wait()

	mdm.Stop()
}
