// cmd/receiver/main.go
package main

import (
	"log"
	"net"
	"time"

	"go.bug.st/serial"
)

const (
	crsfSyncByte = 0xC8
)

func main() {
	// === НАСТРОЙКИ ===
	listenAddr := ":14550"       // слушаем на всех интерфейсах
	portName := "COM5"           // COM-порт полётного контроллера; Linux: /dev/ttyUSB1
	baudRate := 420000           // CRSF скорость

	// Открываем COM-порт для записи в полётный контроллер
	ser, err := serial.Open(portName, &serial.Mode{
		BaudRate: baudRate,
	})
	if err != nil {
		log.Fatalf("Не удалось открыть порт %s: %v", portName, err)
	}
	defer ser.Close()
	log.Printf("Порт %s открыт, бодрейт %d", portName, baudRate)

	// Открываем UDP-слушатель
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		log.Fatalf("Не удалось резолвить адрес: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Не удалось слушать UDP: %v", err)
	}
	defer conn.Close()
	log.Printf("Слушаем UDP на %s", listenAddr)

	buf := make([]byte, 256)
	recvCount := 0
	lastReport := time.Now()

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Ошибка чтения UDP: %v", err)
			continue
		}
		if n < 4 || n > 64 {
			continue // отсекаем мусор
		}
		if buf[0] != crsfSyncByte {
			continue // не CRSF
		}

		// Пишем кадр в COM-порт
		_, err = ser.Write(buf[:n])
		if err != nil {
			log.Printf("Ошибка записи в порт: %v", err)
			continue
		}

		recvCount++
		if time.Since(lastReport) >= time.Second {
			log.Printf("Получено и отправлено кадров: %d", recvCount)
			recvCount = 0
			lastReport = time.Now()
		}
	}
}
