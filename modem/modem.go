/*
GSM-modem SIM800l
Copyright (c) 2023-24 GSB, Georgii Batanov gbatanov@yandex.ru

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
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const RX_BUFFER_SIZE = 1024
const TX_BUFFER_SIZE = 256

var CmdInput chan []byte = make(chan []byte, 256)

type GsmModem struct {
	uart             *Uart
	rxBuff           []byte
	txBuff           []byte
	isCall           bool
	toneCmd          string
	toneCmdStarted   bool
	Flag             bool
	em               *EventEmitter
	myPhoneNumber    string
	myPhoneNumberSms string
	CmdToController  chan string
}

func GsmModemCreate(port string, baud int, phoneNumber string) *GsmModem {
	gsm := GsmModem{}
	gsm.uart = UartCreate(port, baud)
	gsm.rxBuff = make([]byte, 0, RX_BUFFER_SIZE)
	gsm.txBuff = make([]byte, 0, TX_BUFFER_SIZE)
	gsm.isCall = false
	gsm.toneCmd = ""
	gsm.toneCmdStarted = false
	gsm.Flag = true
	gsm.em = EventEmitterCreate()
	gsm.myPhoneNumber = phoneNumber
	gsm.CmdToController = make(chan string, 1)

	phSms := []byte("7" + phoneNumber)
	if len(phSms)%2 != 0 {
		phSms = append(phSms, 'F')
	}
	for i := 0; i < len(phSms); i = i + 2 {
		a := phSms[i]
		phSms[i] = phSms[i+1]
		phSms[i+1] = a
	}
	gsm.myPhoneNumberSms = string(phSms)

	return &gsm
}

func (mdm *GsmModem) Open() error {
	err := mdm.uart.Open()
	if err != nil {
		return err
	}

	go mdm.uart.Loop(CmdInput)

	// Цикл обработки принятых сообщений
	go func() {

		for mdm.Flag {
			buff := <-CmdInput

			if strings.Contains(string(buff), "\r\r\n") { // Ответ на команду
				answer := string(buff)
				parts := strings.Split(answer, "\r\r\n")
				//
				//				log.Printf("Answer on command: %s %s ", parts[0], parts[1])
				if strings.HasPrefix(parts[1], "+CMGR") { // Прочитано смс-сообщение
					go func(msg string) {
						mdm.handleSms(msg)
					}(answer)
				} else if strings.HasPrefix(parts[1], "+CMGS") { // Передано смс-сообщение
					// Реально сюда не придет, потому что в качестве команды будет отправленное сообщение,
					// оно не завершается отправкой \r, а завершается \z, которое не приходит обратно
					mdm.em.SetEvent(parts[0]+"\r", "OK\r\n")
				} else if strings.HasPrefix(parts[1], "+COLP") { // ответ на вызов
					if strings.Contains(parts[1], "\r\n\r\n") {
						partsAnsw := strings.Split(parts[1], "\r\n\r\n")
						mdm.em.SetEvent(parts[0]+"\r", partsAnsw[1])
					}
				} else {
					mdm.em.SetEvent(parts[0]+"\r", parts[1])
				}
			} else if strings.HasPrefix(string(buff), "\r\n+") { //+CMTI, +CLIP, +CUSD, +CMGR
				// Сообщения, которые могут придти в произвольное время
				answer := string(buff[2:])
				log.Printf("Unexpected command: %s", answer)
				if strings.HasPrefix(answer, "+CUSD") || strings.HasPrefix(answer, "0011000B919752109063F50008C10C") { // ответ на запрос баланса
					go func(msg string) {
						mdm.showBalance(msg)
					}(answer)
				} else if strings.HasPrefix(answer, "+CMTI") { // Пришло смс-сообщение
					go func(msg string) {
						mdm.readSms(msg)
					}(answer)
				} else if strings.HasPrefix(answer, "+CMGR") { // Прочитано смс-сообщение
					go func(msg string) {
						mdm.handleSms(msg)
					}(answer)
				} else if strings.HasPrefix(answer, "+CLIP") { // АОН на входящий звонок
					go func(msg string) {
						mdm.checkNumber(msg)
					}(answer)
				} else if strings.HasPrefix(answer, "+DTMF") { // тоновые команды, обработаем синхронно
					if mdm.isCall {
						// в ответе может придти несколько команд +DTMF
						dtmfCount := strings.Count(answer, "+DTMF: ")
						for dtmfCount > 0 {
							pos := strings.Index(answer, "+DTMF: ")
							cmd := string([]byte(answer)[pos+7 : pos+8])
							answer = string([]byte(answer)[pos+8:])
							//							log.Printf("%v\n", []byte(cmd))
							if mdm.toneCmdStarted {
								if cmd == "#" {
									mdm.toneCmdStarted = false
									// дать отбой и выполнить команду
									go func() {
										mdm.executeToneComand()
									}()
								} else {
									mdm.toneCmd += cmd
								}
							} else if cmd == "*" {
								mdm.toneCmdStarted = true
							}
							dtmfCount = strings.Count(answer, "+DTMF: ")
						}
					}
				}
			} else if strings.HasPrefix(string(buff), "\r\nRING") { //RING входящий вызов (после него идет +CLIP c номером: +CLIP: "+71234567890",145,"",0,"",0)
				// можно поделить на две "нормальных" команды - если подряд идут \r\n\r\n, по этому месту делим на две команды
				answer := string(buff[2:])
				log.Printf("Unexpected command: %s", answer)

				if strings.Contains(answer, "+CLIP:") { // Входящий звонок, АОН пришел сразу
					if strings.Contains(answer, "\r\n\r\n") {
						answerParts := strings.Split(answer, "\r\n\r\n")
						answer = "\r\n" + answerParts[1]
					}
					go func(msg string) {
						mdm.checkNumber(msg)
					}(answer)
				}
			} else if strings.Contains(string(buff), "\r\n+CMGS") { // Ответ на окончательную отправку СМС
				log.Printf("SMS sended: %s", string(buff))
				pos := strings.Index(string(buff), "\r\n+CMGS")
				buff = append(buff[:pos], 0x1A)
				mdm.em.SetEvent(string(buff), "OK\r\n")
			} else { // NO CARRIER (положили трубку),
				if string(buff) == "\r\nNO CARRIER\r\n" {
					log.Println("Не берут трубку")
				} else {
					log.Printf("Other: %s", string(buff))
				}
			}
			time.Sleep(time.Second * 2)
		}
	}()
	// инициализацию модема делаем синхронно
	mdm.InitModem()

	return nil
}
func (mdm *GsmModem) Stop() {
	mdm.uart.Stop()
	log.Printf("Modem stop")
}

// Проверяем номер звонящего.
// Если это номер хозяина, поднимаем трубку, иначе прекращаем звонок.
// \r\n+CLIP: "+71234567890",145,"",0,"",0\r\n
func (mdm *GsmModem) checkNumber(answer string) {
	if strings.Contains(answer, mdm.myPhoneNumber) {
		mdm.HangUp()
	} else {
		mdm.HangOut()
	}
}

// Стартовая инициализация модема:
// устанавливаем ответ с эхом
// включаем АОН
// текстовый режим СМС
// включаем тональный режим для приема команд
// Очищаем очередь смс-сообщений
func (mdm *GsmModem) InitModem() {
	mdm.sendCommand("AT\r", "OK")
	mdm.setEcho(true)
	mdm.setAon()
	mdm.sendCommand("AT+CMGF=1\r", "OK")
	mdm.sendCommand("AT+DDET=1,0,0\r", "OK")
	mdm.sendCommand("AT+CMGD=1,4\r", "OK") // Удаление всех сообщений, второй вариант
	mdm.sendCommand("AT+COLP=1\r", "OK")
	log.Println("Модем инициализирован")
}

func (mdm *GsmModem) setEcho(echo bool) bool {
	res := true
	if echo {
		res, _ = mdm.sendCommand("ATE1\r", "ATE1")
	} else {
		res, _ = mdm.sendCommand("ATE0\r", "ATE0")
	}
	return res
}

func (mdm *GsmModem) setAon() bool {
	cmd := "AT+CLIP=1\r"
	res, _ := mdm.sendCommand(cmd, "OK")
	return res
}

// Синхронные команды
// waitAnswer - ожидаемый положительный ответ
// Если ответ не совпал, в ошибке вернется принятый ответ, это имеет значение при голосовых вызовах
func (mdm *GsmModem) sendCommand(cmd string, waitAnswer string) (bool, error) {
	log.Printf("Send command %s", cmd)
	err := mdm.uart.Write([]byte(cmd))
	if err != nil {
		return false, err
	}
	result, err := mdm.em.WaitEvent(cmd, time.Second*60) // минута нужна для правильного ответа на исходящий звонок,
	//														иначе всегда будет Timeout, если не берут трубку
	//	log.Println("Result1:" + result)
	if err != nil {
		return false, err
	}
	//	log.Println("Result2:" + result)
	if waitAnswer != "> " { // Приглашение на ввод СМС не завершается \r\n
		waitAnswer = waitAnswer + "\r\n"
	}
	if result == waitAnswer {
		//		log.Println("Result sendCommand: Ответ совпал")
		return true, nil
	}
	return false, errors.New(result)
}

// Асинхронные команды
func (mdm *GsmModem) sendCommandNoWait(cmd string) bool {
	//	log.Println("Send command without waiting", cmd)
	err := mdm.uart.Write([]byte(cmd))

	return err == nil
}

// сам запрос баланса синхронный, асинхронный ответ придет с +CUSD
func (mdm *GsmModem) GetBalance() bool {
	mdm.sendCommand("AT+CMGF=1\r", "OK")
	time.Sleep(time.Second * 3)
	cmd := "AT+CUSD=1,\"*100#\"\r"
	res, _ := mdm.sendCommand(cmd, "OK")
	return res
}

// Проверено только для абонентов Мегафона
func (mdm *GsmModem) showBalance(msg string) {
	var res string
	if strings.HasPrefix(msg, "0011000B919752109063F50008C10C") { // Новый формат ответа?
		res = strings.ReplaceAll(msg, "0011000B919752109063F50008C10C", "")
	} else {
		pos := strings.Index(msg, "\"")
		res = string([]byte(msg)[pos+1:])
		pos2 := strings.Index(res, "00200440002E")
		res = string([]byte(res)[:pos2])
	}
	//	log.Printf("Balance1: %s\n", res)
	res = strings.Replace(res, "003", "", -1) // -1 - all
	//	log.Printf("Balance2: %s\n", res)
	res = strings.Replace(res, "002E", ".", -1)
	log.Printf("Balance: %s\n", res)
	// Отправляем баланс в контроллер
	mdm.CmdToController <- "/balance " + res

}

// Тут еще не сама смс, а уведомление о ее поступлении и номером в буфере.
// Нужно отправить команду на чтение по этому номеру из буфера сообщений.
//
//	CMTI: "SM",4\r\n
//
// (Получено СМС сообщение)  +CMTI: "SM",4 Уведомление о приходе СМС.
//
//	Второй параметр - номер пришедшего сообщения
func (mdm *GsmModem) readSms(msg string) {
	pos := strings.Index(msg, ",")
	answer := string([]byte(msg[pos+1:]))
	log.Println(answer)
	answer = strings.ReplaceAll(answer, "\r\n", "")
	log.Println(answer)
	num_sms, _ := strconv.Atoi(answer)
	if num_sms > 0 {
		cmd := "AT+CMGR=" + answer + "\r"
		mdm.sendCommandNoWait(cmd)
	}
}

func (mdm *GsmModem) handleSms(msg string) uint16 {
	// принимаю команды пока только со своего телефона
	//	log.Printf("SMS: %s \n", msg)
	cmdCode := 0
	if strings.Contains(msg, "7"+mdm.myPhoneNumber) && strings.Contains(msg, "/cmnd") {
		pos := strings.Index(msg, "/cmnd")
		answer := string([]byte(msg)[pos+5:])
		//		log.Printf("SMS2: %s \n", answer)
		pos = strings.Index(answer, "\r\n\r\n")
		answer = string([]byte(answer)[:pos])
		log.Printf("SMS3: %s \n", answer)
		answer = strings.ReplaceAll(answer, " ", "")
		//		cmdCode, _ = strconv.Atoi(answer)
		mdm.toneCmd = answer // тоновые и смс команды унифицированы
		mdm.executeToneComand()
	}
	// AT+CMGD=1,4
	mdm.sendCommand("AT+CMGD=1,4\r", "OK") // Удаление всех сообщений, второй вариант
	return uint16(cmdCode)
}

// AT+CMGS="+70250109365"         >                       Отправка СМС на номер (в кавычках), после кавычек передаем LF (13)
//>Water leak 1                  +CMGS: 15               Модуль ответит >, передаем сообщение, в конце передаем символ SUB (26)
//                                OK
//  Если в сообщении есть кириллица, нужно сменить текстовый режим на UCS2
//  и перекодировать сообщение

func (mdm *GsmModem) SendSms(sms string) bool {

	var cmd string = ""
	var res bool

	cmd = "AT+CMGF=0\r"
	mdm.sendCommand(cmd, "OK")
	time.Sleep(time.Second * 3)
	sms = mdm.convertSms(sms)
	txtLen := len(sms) / 2
	msgLen := txtLen + 14
	buff := fmt.Sprintf("%02X", txtLen)
	//
	// 00 - Длина и номер SMS центра. 0 - означает, что будет использоваться дефолтный номер.
	// 11 - SMS-SUBMIT
	// 00 - Длина и номер отправителя. 0 - означает что будет использоваться дефолтный номер.
	// 0B - Длина номера получателя (11 цифр)
	// 91 - Тип-адреса. (91 указывает международный формат телефонного номера, 81 - местный формат).
	//
	//  - Телефонный номер получателя в международном формате. (Пары цифр переставлены местами, если номер с нечетным количеством цифр, добавляется F)
	// 00 - Идентификатор протокола
	// 08 - Старший полубайт означает сохранять SMS у получателя или нет (Flash SMS),  Младший полубайт - кодировка(0-латиница 8-кирилица).
	// C1 - Срок доставки сообщения. С1 - неделя
	// 46 - Длина текста сообщения
	// Далее само сообщение в кодировке UCS2 (35 символов кириллицы, 70 байт, 2 байта на символ)

	msg := "0011000B91" + mdm.myPhoneNumberSms + "0008C1"
	msg = msg + buff + sms

	cmd = "AT+CMGS=" + strconv.Itoa(msgLen) + "\r"
	mdm.sendCommand(cmd, "> ") //62 32
	//	if res {
	//		log.Println("Ответ > получен")
	//	} else {
	//		log.Println("Ответ > не получен")
	//	}
	cmdByte := []byte(msg)
	cmdByte = append(cmdByte, 0x1A)
	msg = string(cmdByte)
	res, _ = mdm.sendCommand(msg, "OK")
	if res {
		log.Println("Сообщение отправлено")
	}

	cmd = "AT+CMGF=1\r"
	mdm.sendCommand(cmd, "OK")
	return res
}

// Перекодировка кириллицы
// ёЁ Ё(0xd0 0x81) 0401 ё(0xd1 0x91) 0451
func (mdm *GsmModem) convertSms(msg string) string {

	result := ""
	msgBytes := []byte(msg)
	//	log.Printf("%v", msgBytes)
	for i := 0; i < len(msgBytes); i++ {
		sym := msgBytes[i]

		if sym == 0xD0 { //208
			result = result + "04"
			i++
			sym = msgBytes[i] - 0x80
		} else if sym == 0xD1 { //209
			result = result + "04"
			i++
			sym = msgBytes[i] - 0x40
		} else {
			result = result + "00"
		}

		symHex := fmt.Sprintf("%02x", sym)
		result = result + symHex
	}
	//	log.Println(result)
	return result
}

// Звонок на основной номер
// Если трубку не берут, он звонит еще раз, если и второй раз не берут, модем отвечает NO CARRIER
// Мой модем на все отвечает NO CARRIER - не берут трубку, положили трубку ((
// Нет признака, что трубку взяли
// Но в моем случае важен сам факт звонка, ибо система делает звонок только при аварийных ситуациях
// которые можно уточнить, запросив статус через телеграм или смс
func (mdm *GsmModem) CallMain() {
	go func() {
		cmd := "ATD+7" + mdm.myPhoneNumber + ";\r"
		_, err := mdm.sendCommand(cmd, "OK")
		if err != nil {
			// Здесь будет ответ, отличный от "OK"
			log.Println(err.Error())
		}
	}()
}

// Повесить трубку
// ATH0             OK
func (mdm *GsmModem) HangOut() {
	cmd := "ATH0\r"
	mdm.sendCommand(cmd, "OK")
	mdm.isCall = false
	log.Println("Вешаем трубку")
}

// Поднять трубку
// ATA            OK
func (mdm *GsmModem) HangUp() {
	cmd := "ATA\r"
	_, err := mdm.sendCommand(cmd, "OK")
	if err == nil {
		mdm.isCall = true // включааем признак, позволяющий принимать тоновые команды
		log.Println("Отвечаем на звонок")
	}
}

func (mdm *GsmModem) executeToneComand() {
	mdm.HangOut()
	//	log.Printf("Исполняем команду %s \n", mdm.toneCmd)
	mdm.CmdToController <- mdm.toneCmd
	mdm.toneCmd = ""
}
