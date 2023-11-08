# gsm
Gsm modem, standalone version


## Модем SIM800l

Важно! Cим-карту вставлять ключом наружу!!!

Управление модемом:

| Команда   | Ответ        |   Описание |
|-----------|--------------|--------------------------------|
| AT+CPAS   | +CPAS:0      | <p>Информация о состоянии модуля: |
|           | OK        | 0 - готов к работе            |
|           |               | 2 - неизвестно |
|           |               |3 - входящий звонок |
|           |              | 4 - голосовое соединение |
| AT+CSQ    | +CSQ: 17,0   | Уровень сигнала |
|           | OK           | 0 - < -115 дБ |
|           |              |   1 - -112 дБ |
|           |              |    2-30 -110...-54 дБ|
|           |              |    31 - > -52 дБ|
|           |              |    99 - нет сигнала|
| AT+CBC    | +CBC: 0,95,4134 | Монитор напряжения питания модуля|
|           | OK              | Первый параметр:|
|           |                 | 0 - не заряжается|
|           |                 | 1 - заряжается|
|           |                 | Второй параметр:|
|           |                 | Процент заряда батареи|
|           |                 | Третий параметр:|
|           |                 | Напряжение питания модуля в милливольтах|
| AT+CLIP=1 | OK              | АОН 1 - вкл, 0 - выкл|
| AT+CCLK="13/09/25,13:25:33+05" | ОК | Установка часов «yy/mm/dd,hh:mm:ss+zz»|
| ATD+70250109365; | OK        |   Позвонить на номер +70250109365|
|                 | NO DIALTONE | Нет сигнала|
|                 | BUSY        | Вызов отклонен|
|                 | NO CARRIER  | Повесили трубку,  Не берут трубку|
|                 | NO ANSWER   | Нет ответа|
| ATA              | OK          | Ответить на звонок|
| ATH0             | OK          | Повесить трубку|
| (Входящий звонок) | RING        |                        Входящий звонок|
|                    |             |                     При включенном АОН:|
|                  | +CLIP: "+70250109365",145,"",,"",0 | Номер телефона,(другие параметры мне не интересны)|
| AT+CMGF=1       | OK          |  1 - Включить текстовый режим|
| AT+CSCS="GSM"  | OK           | Кодировка GSM|
| AT+CMGS="+70250109365" |        >  |                     Отправка СМС на номер (в кавычках), после кавычек передаем LF (13)|
| >Water leak 1           |       +CMGS: 15 |              Модуль ответит >, передаем сообщение, в конце передаем символ SUB (26)|
|                          |     OK | |
| (Получено СМС сообщение)  |     +CMTI: "SM",4 |          Уведомление о приходе СМС.|
|                            |                   |        Второй параметр - номер пришедшего сообщения|
| AT+CMGR=2                   |   +CMGR: "REC READ","+790XXXXXXXX","","13/09/21,11:57:46+24"| Чтение СМС сообщений. |
|                              | cmnd1                                                     | В параметре передается номер сообщения.|
|                              | OK                                                        | В ответе передается группа сообщений, |
|                              |                                                           | номер телефона отправителя,|
|                              |                                                           | дата и время отправки, текст сообщения|
| AT+CMGDA="DEL ALL"            | OK      |                 Удаление всех сообщений |
| AT+CMGD=4                     | ОК       |                Удаление указанного сообщения|
| ATD*100#;                     | OK        |                Запрос баланса, баланс приходит в ответе.|
|                              | +CUSD: 0,"Balance:240,68r ", | |
| AT+DDET=<mode>[,<interval>][,<reportMode>][,<ssdet>] | OK  |Включение режима DTMF|
|                                                      |    | mode:  0 - выключен, 1 - включен|
|                                                      |   | interval - минимальный интервал в миллисекундах между двумя нажатиями одной и той же клавиши (диапазон допустимых значений 0-10000). По умолчанию — 0.|
|                                                      |   | reportMode: режим предоставления информации:0 — только код нажатой кнопки, 1 — код нажатой кнопки и время удержания нажатия, в мс|
|                                                      |   | ssdet - не используем|
|                                                      |   |В ответе:|
|                                                      |   |Если <reportMode>=0, то: +DTMF: <key>|
|                                                      |   |Если <reportMode>=1, то: +DTMF: <key>,<last time>|
|                                                      |   |<key> — идентификатор нажатой кнопки (0-9, *, #, A, B, C, D)|
|                                                      |   |<last time> — продолжительность удержания нажатой кнопки, в мс|


| Уведомление |	                Описание	|                                        Пример |
|-------------|-----------------------------|-----------------------------------------------|
| RING	   |                 Уведомление входящего вызова	 |                   RING|
| +CMTI	   |                 Уведомление прихода нового SMS-сообщения	|        +CMTI: "SM",2|
| +CLIP	   |                 Автоопределитель номера во время входящего звонка|	+CLIP: "+78004522441",145,"",0,"",0|
| +CUSD	   |                 Получение ответа на отправленный USSD-запрос	   | +CUSD: 0, " Vash balans 198.02 r.|
|          |                                                                    |  Dlya Vas — nedelya besplatnogo SMS-obsh'eniya s druz'yami! Podkl.: *319#", 15|
| UNDER-VOLTAGE POWER DOWN | | |
| UNDER-VOLTAGE WARNNING | | |
| OVER-VOLTAGE POWER DOWN | | |
| OVER-VOLTAGE WARNNING	    | Сообщения о некорректном напряжении модуля	 |  |    
| UNDER-VOLTAGE WARNNING| | |
| +CMTE	                   | Сообщения о некорректной температуре модуля	  |      +CMTE: 1|

## Примечания к командам
- На длительность нажатия на тональную кнопку ориентироваться не стоит, сильно нестабильно и незакономерно.
- Ответы с модема начинаются и заканчиваются с "\r\n". Если включен режим "Эхо", то перед ответом придет отправленная команда. Разделить команду и ответ можно по сочетанию "\r\r\n".
