package wsutils

import (
	"fmt"
	"time"

	"github.com/fasthttp/websocket"
)

func KeepAlive(c *websocket.Conn, timeout time.Duration) error {
	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	for {
		err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
		if err != nil {
			return err
		}
		time.Sleep(timeout)
		if time.Since(lastResponse) > timeout {
			c.Close()
			return fmt.Errorf("connection timeout")
		}
	}

}
