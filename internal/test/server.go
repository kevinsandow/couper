package test

import (
	"net"
	"strconv"
	"time"
)

func WaitForClosedPort(port int) {
	round := time.Duration(0)

	for {
		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if dialErr != nil {
			break
		}
		_ = conn.Close()

		round++
		if round == 20 {
			panic("port is still in use")
		}
		time.Sleep(time.Second + (time.Second*round)/2)
	}
}

func WaitForOpenPort(port int) {
	round := time.Duration(0)
	for {
		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if dialErr == nil {
			_ = conn.Close()
			return
		}

		time.Sleep(time.Second + (time.Second*round)/2)
		round++
		if round == 20 {
			panic("port is still not listening")
		}
	}
}
