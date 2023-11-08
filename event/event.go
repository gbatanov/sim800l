/*
GSM-modem SIM800l
Copyright (c) GSB, Georgii Batanov gbatanov @ yandex.ru
*/
package event

import (
	"errors"
	"log"
	"time"
)

type Event struct {
	Id      string
	Message string
}

func (ev *Event) Reset() {
	ev.Message = ""
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

// Создаем пустое событие с идентификатором id
func (em *EventEmitter) CreateEvent(id string) *Event {
	event := Event{Id: id, Message: ""}
	em.Events = append(em.Events, &event)
	return &event
}

// Ищем событие по идентификатору, если еще нет, создае новое
func (em *EventEmitter) GetEvent(id string) *Event {
	for _, ev := range em.Events {
		if ev.Id == id {
			return ev
		}
	}
	return em.CreateEvent(id)
}

// Устанавливаем событие, id - код команды, msg - сообщение
func (em *EventEmitter) SetEvent(id string, msg string) {
	event := em.GetEvent(id)
	event.Message = msg
}

// Очищаем событие перед ожиданием ответа
func (em *EventEmitter) ResetEvent(id string) {
	event := em.GetEvent(id)
	event.Message = ""
}

// ждем сообщения с заданным идентификатором
// Здесь мы ожидаем ответ на отправленные нами команды
// Неинициированные нами команды не порождают событие, обрабатываются своим обработчиком
func (em *EventEmitter) WaitEvent(id string, timeout time.Duration) (string, error) {
	event := em.GetEvent(id)
	event.Reset()
	if event.Message != "" {
		return event.Message, nil
	}
	timer1 := time.NewTimer(timeout)
	for {
		select {
		case <-timer1.C:
			timer1.Stop()
			return "", errors.New("Timeout")
		default:
			if event.Message != "" {
				return event.Message, nil
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}
