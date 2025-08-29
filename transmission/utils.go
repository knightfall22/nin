package transmission

import (
	"fmt"
	"log"
	"net"
	"time"
)

func GetFirstOpenPort(address string, port int) int {
	var availablePort int
	for p := port; p-port < 200; p++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, fmt.Sprint(p)), 30*time.Second)

		if conn != nil {
			conn.Close()
			continue
		} else if err != nil {
			availablePort = p
		}

	}

	return availablePort
}

func PingServer(address string) error {
	log.Printf("pinging %s", address)
	c, err := net.DialTimeout("tcp", address, 300*time.Millisecond)
	if err != nil {
		log.Println(err)
		return err
	}

	msg := Message{ID: MessagePing}
	_, err = c.Write(msg.Serialize())
	if err != nil {
		log.Println(err)
		return err
	}

	m, err := DeserializeMessageFromReader(c)
	if err != nil {
		//Fix ipv6 issue later
		// log.Println(err)
		return err
	}

	if m.ID == MessagePong {
		c.Close()
		return nil
	}

	return fmt.Errorf("no pong")
}
