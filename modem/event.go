/*
GSM-modem SIM800l
Copyright (c) 2023-2025 GSB, Georgii Batanov gbatanov@yandex.ru

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
package modem

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

// Ищем событие по идентификатору, если еще нет, создаем новое
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

// Ждем сообщения с заданным идентификатором
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
