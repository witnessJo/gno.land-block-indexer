package msgbroker

type MsgBroker interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string, handler func(message []byte) error) error
	Close() error
}
