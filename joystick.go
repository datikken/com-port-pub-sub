// Чтение положения стиков RadioMaster TX12 в режиме USB Joystick (HID).
// Linux, joydev-интерфейс /dev/input/jsN. Без cgo и внешних зависимостей.
//
// Запуск: go run tx12_joystick.go [/dev/input/js0]
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

// Формат события joydev (struct js_event из linux/joystick.h)
type jsEvent struct {
	Time   uint32 // мс с момента открытия устройства
	Value  int16  // оси: -32767..32767, кнопки: 0/1
	Type   uint8  // 0x01 кнопка, 0x02 ось, |0x80 — начальное состояние
	Number uint8  // номер оси/кнопки
}

const (
	typeButton = 0x01
	typeAxis   = 0x02
	typeInit   = 0x80
)

// Имена осей для стандартного маппинга EdgeTX (AETR).
// Проверь под свою модель: порядок каналов задаётся в настройках.
var axisNames = map[uint8]string{
	0: "Ail (roll) ",
	1: "Ele (pitch)",
	2: "Thr (throt)",
	3: "Rud (yaw)  ",
	4: "S1         ",
	5: "S2         ",
}

type state struct {
	mu      sync.Mutex
	axes    map[uint8]int16
	buttons map[uint8]int16
}

func (s *state) set(e jsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch e.Type &^ typeInit {
	case typeAxis:
		s.axes[e.Number] = e.Value
	case typeButton:
		s.buttons[e.Number] = e.Value
	}
}

func (s *state) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := ""
	for n := uint8(0); n < 8; n++ {
		v, ok := s.axes[n]
		if !ok {
			continue
		}
		name, has := axisNames[n]
		if !has {
			name = fmt.Sprintf("axis %d     ", n)
		}
		out += fmt.Sprintf("%s %+6d (%+7.1f%%)  %s\n",
			name, v, float64(v)/32767.0*100.0, bar(v))
	}
	for n := uint8(0); n < 16; n++ {
		if v, ok := s.buttons[n]; ok && v != 0 {
			out += fmt.Sprintf("btn %d ON\n", n)
		}
	}
	return out
}

// Псевдографическая шкала положения стика.
func bar(v int16) string {
	const w = 21
	pos := int((float64(v)/32767.0 + 1) / 2 * float64(w-1))
	if pos < 0 {
		pos = 0
	}
	if pos > w-1 {
		pos = w - 1
	}
	b := make([]byte, w)
	for i := range b {
		b[i] = '-'
	}
	b[w/2] = '|'
	b[pos] = '#'
	return "[" + string(b) + "]"
}

func main() {
	dev := "/dev/input/js0"
	if len(os.Args) > 1 {
		dev = os.Args[1]
	}

	f, err := os.Open(dev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "не открыть %s: %v\n"+
			"Проверь, что пульт в режиме USB Joystick и что есть права "+
			"(sudo usermod -aG input $USER, затем перелогиниться).\n", dev, err)
		os.Exit(1)
	}
	defer f.Close()

	st := &state{axes: map[uint8]int16{}, buttons: map[uint8]int16{}}

	for {
		var e jsEvent
		if err := binary.Read(f, binary.LittleEndian, &e); err != nil {
			if err == io.EOF {
				fmt.Fprintln(os.Stderr, "устройство отключено")
				return
			}
			fmt.Fprintf(os.Stderr, "ошибка чтения: %v\n", err)
			return
		}

		st.set(e)

		// Начальные события (|0x80) приходят пачкой при открытии — не перерисовываем.
		if e.Type&typeInit != 0 {
			continue
		}

		fmt.Print("\033[H\033[2J") // очистка экрана
		fmt.Printf("Устройство: %s\n\n%s", dev, st)
	}
}