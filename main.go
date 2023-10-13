package main

import (
	"gsm/event"
	"log"
	"sync"
	"time"
)

const VERSION = "0.0.2"

func main() {
	log.Println("Start")
	em := event.EventEmitterCreate()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		em.ResetEvent("AT")
		result := em.WaitEvent("AT", time.Second*10)
		log.Println(result)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		em.ResetEvent("AT+")
		result := em.WaitEvent("AT+", time.Second*3)
		log.Println(result)
		wg.Done()
	}()
	time.Sleep(time.Second * 1)
	em.SetEvent("AT+", "OK+")
	time.Sleep(time.Second * 5)
	em.SetEvent("AT", "OK")
	wg.Wait()
	log.Println("End")
}
