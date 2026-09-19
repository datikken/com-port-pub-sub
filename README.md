# Sender (запускается на компьютере, к которому подключён пульт)
go build -o bin/sender ./sender

# Receiver (запускается на компьютере/raspberry Pi у дрона)
go build -o bin/receiver ./receiver

https://github.com/Bluewave2/Drift-Script

# Com ports list
sudo dmesg | grep -i tty
