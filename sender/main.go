// cmd/sender/main.go
package main

import (
	"log"
	"net"
	"time"

	"go.bug.st/serial"
)

const (
	crsfSyncByte = 0xC8
	crsfMaxFrame = 64 // максимальный размер CRSF-кадра
)

func main() {
	// === НАСТРОЙКИ ===
	portName := "/dev/ttyUSB0" // Windows; Linux: /dev/ttyUSB0
	// portName := "COM3" // Windows; Linux: /dev/ttyUSB0
	baudRate := 420000 // CRSF стандарт
	udpAddr := "192.168.1.100:14550" // куда отправляем

	// Открываем COM-порт
	ser, err := serial.Open(portName, &serial.Mode{
		BaudRate: baudRate,
	})
	if err != nil {
		log.Fatalf("Не удалось открыть порт %s: %v", portName, err)
	}
	defer ser.Close()
	log.Printf("Порт %s открыт, бодрейт %d", portName, baudRate)

	// Открываем UDP-соединение
	udpConn, err := net.Dial("udp", udpAddr)
	if err != nil {
		log.Fatalf("Не удалось открыть UDP: %v", err)
	}
	defer udpConn.Close()
	log.Printf("UDP отправка на %s", udpAddr)

	// Буфер для накопления сырых данных и для собранного кадра
	rawBuf := make([]byte, 256)
	var frame []byte // собираемый кадр
	var frameLen int // ожидаемая длина кадра (после чтения байта длины)
	state := 0       // 0 — ждём sync, 1 — читаем длину, 2 — читаем тело кадра

	sentCount := 0
	lastReport := time.Now()

	for {
		n, err := ser.Read(rawBuf)
		if err != nil {
			log.Printf("Ошибка чтения: %v", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if n == 0 {
			continue
		}

		for i := 0; i < n; i++ {
			b := rawBuf[i]

			switch state {
			case 0: // ищем sync-байт
				if b == crsfSyncByte {
					frame = []byte{b}
					state = 1
				}

			case 1: // читаем байт длины
				frame = append(frame, b)
				frameLen = int(b) + 2 // +2: sync + length byte
				if frameLen > crsfMaxFrame || frameLen < 4 {
					// Некорректная длина — сброс
					state = 0
					frame = nil
					continue
				}
				state = 2

			case 2: // читаем тело кадра
				frame = append(frame, b)
				if len(frame) == frameLen {
					// Кадр собран — отправляем по UDP
					_, err := udpConn.Write(frame)
					if err != nil {
						log.Printf("Ошибка отправки UDP: %v", err)
					}
					sentCount++
					if time.Since(lastReport) >= time.Second {
						log.Printf("Отправлено кадров: %d", sentCount)
						sentCount = 0
						lastReport = time.Now()
					}
					state = 0
					frame = nil
				}
			}
		}
	}
}
