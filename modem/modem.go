package modem

import (
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
				mdm.em.SetEvent(parts[0], parts[1])
				log.Printf("Answer on command: %s %s ", parts[0], parts[1])
				if strings.HasPrefix(parts[1], "+CMGR") { // Прочитано смс-сообщение
					go func(msg string) {
						mdm.handleSms(msg)
					}(answer)
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
	mdm.sendCommand("AT+CMGF=1\r", "AT+CMGF")
	mdm.sendCommand("AT+DDET=1,0,0\r", "AT+DDET")
	mdm.sendCommand("AT+CMGD=1,4\r", "AT+CMGD") // Удаление всех сообщений, второй вариант
	mdm.sendCommand("AT+COLP=1\r", "AT+COLP")
}

func (mdm *GsmModem) setEcho(echo bool) bool {
	cmd := "ATE"
	if echo {
		cmd = cmd + "1\r"
	} else {
		cmd = cmd + "0\r"
	}
	res, _ := mdm.sendCommand(cmd, "ATE")
	return res
}

func (mdm *GsmModem) setAon() bool {
	cmd := "AT+CLIP=1\r"
	res, _ := mdm.sendCommand(cmd, "AT+CLIP")
	return res
}

// Синхронные команды
func (mdm *GsmModem) sendCommand(cmd string, waitAnswer string) (bool, error) {
	log.Println("Send command ", cmd)
	err := mdm.uart.Write([]byte(cmd))
	if err != nil {
		return false, err
	}
	result, err := mdm.em.WaitEvent(cmd, time.Second*5)

	if err != nil {
		return false, err
	}
	log.Println("Result:" + result)
	waitAnswer = cmd + "\r\n" + waitAnswer + "\r\n;"

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
	mdm.sendCommand("AT+CMGD=1,4\r", "AT+CMGD") // Удаление всех сообщений, второй вариант
	return uint16(cmdCode)
}
