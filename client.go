package tcpsrv

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
)

type Client interface {
	Connect(port int, host string, connected func(w io.Writer)) error
	Close()
}

type TCPClientHandler func(response []byte) bool

type TCPClient struct {
	Client
	Handler TCPClientHandler
	Conn    *net.TCPConn
	quit    chan int
}

func (c *TCPClient) Connect(port int, host string, connected func()) error {
	raddr, err := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return err
	}
	c.Conn = conn
	defer c.Conn.Close()

	c.quit = make(chan int)

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	go connected()
Loop:
	for {
		cres := make(chan []byte)
		cerr := make(chan error)
		go func(conn *net.TCPConn, cres chan []byte, cerr chan error) {
			b, err := readall(conn)
			if err != nil {
				cerr <- err
				return
			}
			cres <- b
		}(c.Conn, cres, cerr)

		select {
		case res := <-cres:
			if c.Handler != nil && !c.Handler(res) {
				break Loop
			}
		case err := <-cerr:
			fmt.Println(err)
			break Loop
		case <-c.quit:
			break Loop
		case <-interrupt:
			break Loop
		}
	}
	return nil
}

func (c *TCPClient) Close() {
	c.quit <- 1
}
