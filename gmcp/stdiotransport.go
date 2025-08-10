package gmcp

import (
	"bufio"
	"fmt"
	"os"
)

type stdioTransport struct {
	singleSessionChan chan *stdioSession
	server            *MCPServer
}

func NewStdioTransport(server *MCPServer) MCPTransport {
	s := &stdioTransport{
		singleSessionChan: make(chan *stdioSession, 1),
		server:            server,
	}
	s.singleSessionChan <- newStdioSession(s)

	return s
}

var (
	ErrClosed = fmt.Errorf("server has been closed")
)

func (s *stdioTransport) Listen() error {
	return nil
}

func (s *stdioTransport) Accept() (Session, error) {
	session, closed := <-s.singleSessionChan
	if closed {
		return nil, ErrClosed
	}

	return session, nil
}

func (s *stdioTransport) Close() error {
	close(s.singleSessionChan)

	return nil
}

type stdioSession struct {
	transport   *stdioTransport
	done        chan struct{}
	messageChan chan []byte
}

func newStdioSession(transport *stdioTransport) *stdioSession {
	return &stdioSession{
		transport:   transport,
		done:        make(chan struct{}),
		messageChan: make(chan []byte, 1024),
	}
}

func (s *stdioSession) readerLoop() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-s.done:
			return nil
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
	}
}

func (s *stdioSession) writerLoop() error {
	for {
		select {
		case <-s.done:
			return nil
		case message := <-s.messageChan:
			fmt.Fprintf(os.Stdout, "%s\n", message)
		}
	}
}

func (s *stdioSession) close() error {
	select {
	case _, closed := <-s.done:
		if closed {
			return nil
		}
	default:
	}

	close(s.done)
	return nil
}
