package gmcp

import (
	"context"
	"github.com/mangohow/gowlb/tools/collection"
	"github.com/mangohow/gowlb/tools/sync"
)

type MCPServer struct {
	sessionManager collection.ConcurrentMap[string, Session]
	cfg            ServerConfig
	router         Router
	wg             sync.WaitGroup
	ctx            context.Context
}

func NewMCPServer(opts ...Option) *MCPServer {
	cfg := ServerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.transport == nil {
		cfg.transport = NewStdioTransport()
	}

	return &MCPServer{
		sessionManager: collection.NewConcurrentMap[string, Session](),
		cfg:            cfg,
	}
}

type ServerConfig struct {
	transport MCPTransport
}

type Option func(s *ServerConfig)

func WithTransport(transport MCPTransport) Option {
	return func(s *ServerConfig) {
		s.transport = transport
	}
}

func (s *MCPServer) Start() error {
	for {
		session, err := s.cfg.transport.Accept()
		if err != nil {
			return err
		}

		s.sessionManager.Set(session.SessionID(), session)
		s.wg.Go(func() {
			defer s.sessionManager.Delete(session.SessionID())

			err := session.readerLoop()
			if err != nil {
				session.close()
				return
			}
		})

		s.wg.Go(func() {
			err := session.writerLoop()
			if err != nil {
				session.close()
				return
			}
		})
	}
}

func (s *MCPServer) HandleRequest(session Session, req JSONRPCRequest) error {

}
