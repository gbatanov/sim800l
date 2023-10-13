package main

import (
	"gsm/event"
	"log"
	"sync"
	"time"
)

const VERSION = "0.1.2"

func main() {
	log.Println("Start")
	em := event.EventEmitterCreate()
	var wg sync.WaitGroup
	cmds := []string{"AT", "AT+", "AT-"}
	for _, cmd := range cmds {
		wg.Add(1)
		go sendCmd(em, cmd, &wg)
	}

	time.Sleep(time.Second * 1)
	em.SetEvent("AT+", "OK+")
	em.SetEvent("AT-", "OK-")
	time.Sleep(time.Second * 5)
	em.SetEvent("AT", "OK")
	wg.Wait()
	log.Println("End")
}

func sendCmd(em *event.EventEmitter, cmd string, wg *sync.WaitGroup) {
	em.ResetEvent(cmd)
	// Тут типа отправляем команду и начинаемждать ответ
	result, err := em.WaitEvent(cmd, time.Second*3)
	if err == nil {
		log.Println(result)
	} else {
		log.Println(err.Error())
	}
	wg.Done()
}
