package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/BurntSushi/toml"
)

type endPoint struct {
	IPAddr string
	Port   int
}

type forward struct {
	UseTx2 bool `toml:"use_tx2"`
}

type config struct {
	Rx1AsSender   bool `toml:"rx1_as_sender"`
	Tx2AsReceiver bool `toml:"tx2_as_receiver"`
	RX1           endPoint
	TX1           endPoint
	RX2           endPoint
	TX2           endPoint
}

func listen(ep endPoint) net.PacketConn {
	conn, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", ep.IPAddr, ep.Port))
	if err != nil {
		panic(err)
	}
	return conn
}

func dial(ep endPoint) net.Conn {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ep.IPAddr, ep.Port))
	if err != nil {
		panic(err)
	}
	return conn
}

func sendToRx(tx net.PacketConn, chDat chan []byte, chRmtAddr chan net.Addr) {
	var addr net.Addr
	go func() {
		for {
			addr = <-chRmtAddr
		}
	}()
	for {
		dat := <-chDat
		fmt.Printf("sendToRx received %v from chan\n", dat)
		tx.WriteTo(dat, addr)
	}
}

func sendToTx(name string, tx net.Conn, chDat chan []byte) {
	fmt.Printf("%s(%v) sender start\n", name, tx.RemoteAddr())
	for {
		dat := <-chDat
		fmt.Printf("%s received %v from chan\n", name, dat)
		tx.Write(dat)
	}
}

func recv(name string, rx net.PacketConn, chDat chan []byte, chRmtAddr chan net.Addr) {
	buf := make([]byte, 1500)
	fmt.Printf("%s(%v) receiver start\n", name, rx.LocalAddr())
	for {
		length, rmtAddr, _ := rx.ReadFrom(buf)
		fmt.Printf("%s(%v) received from %v\n", name, rmtAddr, rx.LocalAddr())
		chRmtAddr <- rmtAddr
		chDat <- buf[:length]
	}
}

func main() {
	var config config
	_, err := toml.DecodeFile("fwudp.toml", &config)
	if err != nil {
		panic(err)
	}

	rx1 := listen(config.RX1)
	tx1 := dial(config.TX1)

	rx2 := listen(config.RX2)
	tx2 := dial(config.TX2)

	chRx1RmtAddr := make(chan net.Addr)
	chRx2RmtAddr := make(chan net.Addr)

	chRx1Dat := make(chan []byte)
	chRx2Dat := make(chan []byte)

	go sendToTx("tx1", tx1, chRx1Dat)

	go recv("rx1", rx1, chRx1Dat, chRx1RmtAddr)

	go recv("rx2", rx2, chRx2Dat, chRx2RmtAddr)

	if config.Rx1AsSender {
		go sendToRx(rx1, chRx2Dat, chRx1RmtAddr)
	} else {
		go sendToTx("tx2", tx2, chRx2Dat)
		go func() {
			for {
				<-chRx1RmtAddr
			}
		}()
	}

	go func() {
		for {
			<-chRx2RmtAddr
		}
	}()

	scan := bufio.NewScanner(os.Stdin)
	scan.Scan()
	fmt.Println("終了します")
}
