package modem

import (
	"gsm/event"
	"log"
	"strings"
	"time"
)

const RX_BUFFER_SIZE = 1024
const TX_BUFFER_SIZE = 256

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
				mdm.em.SetEvent(parts[0], parts[1])
				log.Printf("Answer on command: %s %s ", parts[0], parts[1])

			} else if strings.HasPrefix(string(buff), "\r\n+") { //+CMTI, +CLIP, +CUSD,
				answer := string(buff[2:])
				log.Printf("Unexpected command: %s", answer)
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
