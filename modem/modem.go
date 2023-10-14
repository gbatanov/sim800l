package modem

import (
	"fmt"
	"gsm/event"
	"log"
	"strconv"
	"strings"
	"time"
)

const RX_BUFFER_SIZE = 1024
const TX_BUFFER_SIZE = 256
const MY_PHONE_NUMBER = "9250109365"

var CmdInput chan []byte = make(chan []byte, 256)

type GsmModem struct {
	uart           *Uart
	rxBuff         []byte
	txBuff         []byte
	isCall         bool
	toneCmd        string
	toneCmdStarted bool
	Flag           bool
	em             *event.EventEmitter
}

func GsmModemCreate(port string, os string) *GsmModem {
	gsm := GsmModem{}
	gsm.uart = UartCreate(port, os)
	gsm.rxBuff = make([]byte, 0, RX_BUFFER_SIZE)
	gsm.txBuff = make([]byte, 0, TX_BUFFER_SIZE)
	gsm.isCall = false
	gsm.toneCmd = ""
	gsm.toneCmdStarted = false
	gsm.Flag = true
	gsm.em = event.EventEmitterCreate()

	return &gsm
}

func (mdm *GsmModem) Open(baud int) error {
	err := mdm.uart.Open(baud)
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
				log.Printf("Answer on command: %s %s ", parts[0], parts[1])
				if strings.HasPrefix(parts[1], "+CMGR") { // Прочитано смс-сообщение
					go func(msg string) {
						mdm.handleSms(msg)
					}(answer)
				} else if strings.HasPrefix(parts[1], "+CMGS") { // Передано смс-сообщение
					mdm.em.SetEvent(parts[0]+"\r", "OK")
				} else {
					mdm.em.SetEvent(parts[0]+"\r", parts[1])
				}
			} else if strings.HasPrefix(string(buff), "\r\n+") { //+CMTI, +CLIP, +CUSD, +CMGR
				answer := string(buff[2:])
				log.Printf("Unexpected command: %s", answer)
				if strings.HasPrefix(answer, "+CUSD") { // ответ на запрос баланса
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
				}
			} else if strings.HasPrefix(string(buff), "\r\nRING") { //RING входящий вызов (после него идет +CLIP c номером: +CLIP: "+79250109365",145,"",0,"",0)
				// можно поделить на две "нормальных" команды - если подряд идут \r\n\r\n, по этому месту делим на две команды
				answer := string(buff[2:])
				log.Printf("Unexpected command: %s", answer)
			} else { // NO CARRIER (положили трубку),
				log.Printf("Other: %s", string(buff))
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
func (mdm *GsmModem) sendCommand(cmd string, waitAnswer string) (bool, error) {
	log.Printf("Send command %s", cmd)
	err := mdm.uart.Write([]byte(cmd))
	if err != nil {
		return false, err
	}
	result, err := mdm.em.WaitEvent(cmd, time.Second*15)
	log.Println("Result1:" + result)
	if err != nil {
		return false, err
	}
	log.Println("Result2:" + result)
	waitAnswer = waitAnswer + "\r\n"
	if result == waitAnswer {
		log.Println("Result3: Ответ совпал")
	}
	return result == waitAnswer, nil
}

// Асинхронные команды
func (mdm *GsmModem) sendCommandNoWait(cmd string) bool {
	log.Println("Send command without waiting", cmd)
	err := mdm.uart.Write([]byte(cmd))

	return err == nil
}

// сам запрос баланса синхронный, асинхронный ответ придет с +CUSD
func (mdm *GsmModem) GetBalance() bool {
	mdm.sendCommand("AT+CMGF=1\r", "AT+CMGF")
	cmd := "AT+CUSD=1,\"*100#\"\r"
	res, _ := mdm.sendCommand(cmd, "OK")
	return res
}

// Проверено только для абонентов Мегафона
func (mdm *GsmModem) showBalance(msg string) {
	pos := strings.Index(msg, "\"")
	res := string([]byte(msg)[pos+1:])
	pos2 := strings.Index(res, "00200440002E")
	res = string([]byte(res)[:pos2])
	log.Printf("Balance1: %s\n", res)
	res = strings.Replace(res, "003", "", -1) // -1 - all
	log.Printf("Balance2: %s\n", res)
	res = strings.Replace(res, "002E", ".", -1)
	log.Printf("Balance: %s\n", res)
	/*
		balance_ = res
		// отправляем баланс в телеграм
		if (app->withTlg)
		 app->tlg32->send_message("Баланс: " + res + " руб.");

		// если была команда запроса баланса с тонового набора
		if (balance_to_sms)
		{
		  // отправляем баланс в смс
		  send_sms("Balance " + res + " rub");
		  balance_to_sms = false;
		}
		}
	*/
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

// ||CMGR: "REC UNREAD","+70250109365","","22/09/03,12:42:54+12"||test 5||||OK||
// ||+CMGR: "REC UNREAD","+79050109365","","23/06/08,17:32:30+12"||/cmnd401||||OK||
func (mdm *GsmModem) handleSms(msg string) uint16 {
	// принимаю команды пока только со своего телефона
	log.Printf("SMS: %s \n", msg)
	cmdCode := 0
	if strings.Index(msg, "7"+MY_PHONE_NUMBER) != -1 && strings.Index(msg, "/cmnd") != -1 {
		// answer = gsbstring::remove_before(answer, "/cmnd");
		pos := strings.Index(msg, "/cmnd")
		answer := string([]byte(msg)[pos+5:])
		log.Printf("SMS2: %s \n", answer)
		// answer = gsbstring::remove_after(answer, "||||");
		pos = strings.Index(answer, "\r\n\r\n")
		answer = string([]byte(answer)[:pos])
		log.Printf("SMS3: %s \n", answer)
		answer = strings.ReplaceAll(answer, " ", "")
		cmdCode, _ = strconv.Atoi(answer)
		//	  execute_tone_command(answer);
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

	cmd := ""
	res := true

	cmd = "AT+CMGF=0\r"
	res, _ = mdm.sendCommand(cmd, "OK")

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
	// 9752109063F5 - Телефонный номер получателя в международном формате. (Пары цифр переставлены местами, если номер с нечетным количеством цифр, добавляется F) 79250109365 -> 9752109063F5
	// 00 - Идентификатор протокола
	// 08 - Старший полубайт означает сохранять SMS у получателя или нет (Flash SMS),  Младший полубайт - кодировка(0-латиница 8-кирилица).
	// C1 - Срок доставки сообщения. С1 - неделя
	// 46 - Длина текста сообщения
	// Далее само сообщение в кодировке UCS2 (35 символов кириллицы, 70 байт, 2 байта на символ)

	msg := "0011000B91" + "9752109063F5" + "0008C1"
	msg = msg + buff + sms

	buff = fmt.Sprintf("%02X", msgLen)

	cmd = "AT+CMGS=" + strconv.Itoa(msgLen) + "\r"
	res, _ = mdm.sendCommand(cmd, "> ") //62 32

	//+ std::string("46")+ std::string("043F0440043804320435044200200445043004310440002C0020044D0442043E00200442043504410442043E0432043E043500200441043E043E043104490435043D04380435");
	//                                  043F0440043804320435044200200445043004310440002C0020044D0442043E00200442043504410442043E0432043E043500200441043E043E043104490435043D04380435//
	cmdByte := []byte(msg)
	cmdByte = append(cmdByte, 0x1A)
	msg = string(cmdByte)
	res, _ = mdm.sendCommand(msg, "OK")
	cmd = "AT+CMGF=1\r"
	mdm.sendCommand(cmd, "OK")
	if res {
		log.Println("Сообщение отправлено")
	}
	return res
}

// Перекодировка кириллицы
// ёЁ Ё(0xd0 0x81) 0401 ё(0xd1 0x91) 0451
func (mdm *GsmModem) convertSms(msg string) string {

	result := ""
	msgBytes := []byte(msg)
	log.Printf("%v", msgBytes)
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
	log.Println(result)
	return result
}
