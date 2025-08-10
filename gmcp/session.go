package gmcp

type Session interface {
	readerLoop() error
	writerLoop() error
	close() error
	SessionID() string
}
