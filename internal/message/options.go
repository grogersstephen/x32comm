package message

type Option interface {
	apply(*Message) error
}
