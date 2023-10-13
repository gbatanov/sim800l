package event

import (
	"log"
	"time"
)

type Event struct {
	Id      string
	Message string
}

type EventEmitter struct {
	Events []*Event
}

func init() {
	log.Println("event init")
}

func EventEmitterCreate() *EventEmitter {
	evEmitter := EventEmitter{}
	evEmitter.Events = make([]*Event, 0)
	return &evEmitter
}

// Create empty event with Id = id
func (em *EventEmitter) CreateEvent(id string) *Event {
	event := Event{Id: id, Message: ""}
	em.Events = append(em.Events, &event)
	return &event
}

// Find event in emitter, if not finded< create empty
func (em *EventEmitter) GetEvent(id string) *Event {
	for _, ev := range em.Events {
		if ev.Id == id {
			return ev
		}
	}
	return em.CreateEvent(id)
}

// When a message arrives from the modem, id - command, msg - message
func (em *EventEmitter) SetEvent(id string, msg string) {
	event := em.GetEvent(id)
	event.Message = msg
}

// Call before waiting
func (em *EventEmitter) ResetEvent(id string) {
	event := em.GetEvent(id)
	event.Message = ""
}

// wait for a message with a given identifier
func (em *EventEmitter) WaitEvent(id string, timeout time.Duration) string {
	event := em.GetEvent(id)
	if event.Message != "" {
		return event.Message
	}
	timer1 := time.NewTimer(timeout)
	for {
		select {
		case <-timer1.C:
			timer1.Stop()
			return ""
		default:
			if event.Message != "" {
				return event.Message
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}
