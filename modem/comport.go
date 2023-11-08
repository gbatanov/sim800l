package modem

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tarm/serial"
)

const SOF byte = 0xFE

type Uart struct {
	port       string
	os         string
	comport    *serial.Port
	Flag       bool
	portOpened bool
}

func init() {
	fmt.Println("Init in serial3")
	// TODO: check availability serial port
}

func UartCreate(port string, os string) *Uart {
	uart := Uart{port: port, os: os}
	return &uart
}

func (u *Uart) Open(baud int) error {
	comport, err := u.openPort(baud)
	if err != nil {
		log.Println("Error open port ", u.port)
		return err
	}
	u.Flag = true
	u.comport = comport
	u.portOpened = true
	return nil
}

// Opening the given port
func (u Uart) openPort(baud int) (*serial.Port, error) {
	c := &serial.Config{Name: u.port, Baud: baud, ReadTimeout: time.Second * 3}
	return serial.OpenPort(c)

}

func (u *Uart) Stop() {
	if u.portOpened {
		u.Flag = false
		u.comport.Flush()
		u.comport.Close()
		u.portOpened = false
		log.Println("comport closed")
	}
}

// write a sequence of bytes to serial port
func (u Uart) Write(text []byte) error {
	n, err := u.comport.Write(text)
	if err != nil {
		return err
	}
	if n != len(text) {
		return errors.New("Write error")
	}
	return nil
}

// The cycle of receiving commands from the zhub
// in this serial port library version we get chunks 64 byte size !!!
func (u *Uart) Loop(cmdinput chan []byte) {
	BufRead := make([]byte, 256)
	BufReadResult := make([]byte, 0)
	k := 0
	for u.Flag {
		n, err := u.comport.Read(BufRead)
		if err != nil {
			if n != 0 {
				u.Flag = false
			}
		} else if err == nil && n > 0 {

			BufReadResult = append(BufReadResult, BufRead[:n]...)
			k += n
			//			log.Printf("1.Received: %v \n", BufReadResult[:k])

			cnt := strings.Count(string(BufReadResult[:k]), "\r\n")
			if cnt > 1 {
				// Надо найти последнее вхождение \r\n и если после него есть
				// еще символы, их надо перенести в следующий буфер вместе с \r\n
				//
				// k-1 - последний символ
				// k-2 - предпоследний
				log.Printf("2.Received: %v \n", BufReadResult[:k])
				if BufReadResult[k-2] == '\r' && BufReadResult[k-1] == '\n' {
					cmdinput <- BufReadResult[:k]
					BufReadResult = make([]byte, 0)
					k = 0
				} else {
					z := k - 1
					for BufReadResult[z] != '\r' {
						z = z - 1
					}

					cmdinput <- BufReadResult[:z]
					BufReadResult = BufReadResult[z:k]
					k = k - z
				}
			} else {
				//проверим на > ответ на первую часть отправки СМС
				if k > 2 {
					if BufReadResult[k-2] == '>' && BufReadResult[k-1] == ' ' {
						//						log.Printf("3.Received: %v \n", BufReadResult[:k])
						cmdinput <- BufReadResult[:k]
						BufReadResult = make([]byte, 0)
						k = 0
					}
				}
			}

			// if there is no command, wait 1 sec, меньше секунды показывает нестабильный результат
			time.Sleep(1000 * time.Millisecond)
		}
		for i := 0; i < n; i++ {
			BufRead[i] = 0
		}
	}
}

func (u Uart) SendCommandToDevice(buff []byte) error {

	err := u.Write(buff)
	return err
}
