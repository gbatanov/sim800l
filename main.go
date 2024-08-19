/*
GSM-modem SIM800l
Copyright (c) 2023-2024 GSB, Georgii Batanov gbatanov@yandex.ru

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package main

import (
	"bufio"
	"context"

	"time"

	_ "embed"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gbatanov/sim800l/modem"
)

//go:embed version.txt
var VERSION string

const PORT = "/dev/cu.usbserial-A50285BI" //"/dev/ttyUSB0"
const PHONE_NUMBER = "9250109365"

// Пример использования пакета sim800l/modem
func main() {
	log.Println("Start")
	Flag := true

	phoneNumber := PHONE_NUMBER

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := bufio.NewReader(os.Stdin)

	sigs := make(chan os.Signal, 1)
	// signal.Notify registers this channel to receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// This goroutine performs signal blocking.
	// When goroutine receives signal, it prints signal name out and then notifies the program that it can be terminated.
	go func() {
		sig := <-sigs
		log.Println(sig)
		Flag = false
		cancel()
	}()

	mdm := modem.GsmModemCreate(PORT, 9600, phoneNumber)
	err := mdm.Open(ctx)

	if err != nil {
		return
	}
	defer mdm.Stop()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for Flag {
			select {
			case <-ctx.Done():
				Flag = false
				return
			case tcmd := <-mdm.CmdToController:
				log.Println(tcmd)
			default:
				text, _ := reader.ReadString('\n') //  здесь висит
				log.Printf("%v", []byte(text))
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
				default:
					time.Sleep(time.Second * 3)
				}
			}
		} //for
		wg.Done()
	}()
	wg.Wait()

	log.Println("End")

}
