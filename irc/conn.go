package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var (
	ErrBadProtocol = errors.New("This server does not speak IRC")
)

func (c *Client) Connect(address string) {
	if idx := strings.Index(address, ":"); idx < 0 {
		c.Host = address

		if c.TLS {
			address += ":6697"
		} else {
			address += ":6667"
		}
	} else {
		c.Host = address[:idx]
	}
	c.Server = address
	c.dialer = &net.Dialer{Timeout: 10 * time.Second}

	c.connChange(false, nil)
	go c.run()
}

func (c *Client) Write(data string) {
	c.out <- data + "\r\n"
}

func (c *Client) Writef(format string, a ...interface{}) {
	c.out <- fmt.Sprintf(format+"\r\n", a...)
}

func (c *Client) write(data string) {
	c.conn.Write([]byte(data + "\r\n"))
}

func (c *Client) writef(format string, a ...interface{}) {
	fmt.Fprintf(c.conn, format+"\r\n", a...)
}

func (c *Client) run() {
	c.tryConnect()

	for {
		select {
		case <-c.quit:
			if c.Connected() {
				c.disconnect()
			}

			c.sendRecv.Wait()
			close(c.Messages)
			return

		case <-c.reconnect:
			c.disconnect()
			c.connChange(false, nil)

			c.sendRecv.Wait()
			c.reconnect = make(chan struct{})

			c.tryConnect()
		}
	}
}

type ConnectionState struct {
	Connected bool
	Error     error
}

func (c *Client) connChange(connected bool, err error) {
	c.ConnectionChanged <- ConnectionState{
		Connected: connected,
		Error:     err,
	}
}

func (c *Client) disconnect() {
	c.lock.Lock()
	c.connected = false
	c.lock.Unlock()

	c.conn.Close()
}

func (c *Client) tryConnect() {
	for {
		select {
		case <-c.quit:
			return

		default:
		}

		err := c.connect()
		if err != nil {
			c.connChange(false, err)
		} else {
			c.backoff.Reset()

			c.flushChannels()
			return
		}

		time.Sleep(c.backoff.Duration())
	}
}

func (c *Client) connect() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.TLS {
		conn, err := tls.DialWithDialer(c.dialer, "tcp", c.Server, c.TLSConfig)
		if err != nil {
			return err
		}

		c.conn = conn
	} else {
		conn, err := c.dialer.Dial("tcp", c.Server)
		if err != nil {
			return err
		}

		c.conn = conn
	}

	c.connected = true
	c.connChange(true, nil)
	c.reader = bufio.NewReader(c.conn)

	c.register()

	c.sendRecv.Add(1)
	go c.recv()

	return nil
}

func (c *Client) send() {
	defer c.sendRecv.Done()

	for {
		select {
		case <-c.quit:
			return

		case <-c.reconnect:
			return

		case msg := <-c.out:
			_, err := c.conn.Write([]byte(msg))
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) recv() {
	defer c.sendRecv.Done()

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			select {
			case <-c.quit:
				return

			default:
				close(c.reconnect)
				return
			}
		}

		msg := parseMessage(line)
		if msg == nil {
			close(c.quit)
			c.connChange(false, ErrBadProtocol)
			return
		}

		switch msg.Command {
		case Ping:
			go c.write("PONG :" + msg.LastParam())

		case Join:
			if msg.Nick == c.GetNick() {
				c.addChannel(msg.Params[0])
			}

		case Nick:
			if msg.Nick == c.GetNick() {
				c.setNick(msg.LastParam())
			}

		case ReplyWelcome:
			c.setNick(msg.Params[0])
			c.sendRecv.Add(1)
			go c.send()

		case ErrNicknameInUse:
			if c.HandleNickInUse != nil {
				go c.writeNick(c.HandleNickInUse(msg.Params[1]))
			}
		}

		c.Messages <- msg
	}
}
