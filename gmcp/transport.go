package gmcp

type MCPTransport interface {
	Listen() error
	Accept() (Session, error)
	Close() error
}
