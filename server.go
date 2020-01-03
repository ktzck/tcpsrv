package tcpsrv

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
)

type Server interface {
	Listen(port int, host string) error
	Close()
	Handle(conn net.Conn) error
}

type TCPServerHandler func(w io.Writer, request []byte) bool

type TCPServer struct {
	Server
	listener *net.TCPListener
	quit     chan int
	Handler  TCPServerHandler
}

func (s *TCPServer) Listen(port int, host string) error {
	addr, err := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()
	s.listener = l
	s.quit = make(chan int)

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

ListenLoop:
	for {
		cconn := make(chan *net.TCPConn)
		cerr := make(chan error)
		go func() {
			conn, err := l.AcceptTCP()
			if err != nil {
				cerr <- err
				return
			}
			cconn <- conn
		}()

		select {
		case conn := <-cconn:
			go s.Handle(conn)
		case err := <-cerr:
			fmt.Println(err)
		case <-s.quit:
			break ListenLoop
		case <-interrupt:
			break ListenLoop
		}
		cconn = nil
		cerr = nil
	}
	return nil
}

func (s *TCPServer) Handle(conn *net.TCPConn) {
	b, err := readall(conn)
	if err == nil && s.Handler != nil && s.Handler(conn, b) {
		s.Handle(conn)
	} else {
		conn.Close()
	}
}

func (s *TCPServer) Close() {
	s.quit <- 1
}

func readall(r io.Reader) ([]byte, error) {
	// nr := bufio.NewReader(r)
	_b := make([]byte, 1024)
	var b []byte
	for {
		n, err := r.Read(_b)
		// n, err := nr.Read(_b)
		if err != nil {
			return nil, err
		}
		b = append(b, _b[:n]...)
		if n < 1024 {
			break
		}
	}
	return b, nil
}
