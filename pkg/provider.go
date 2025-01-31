package surp

import "fmt"

type InMemoryProvider[T any] struct {
	name     string
	value    T
	encoder  Encoder[T]
	decoder  Decoder[T]
	rw       bool
	metadata map[string]string
	getterCh chan []byte
	setterCh chan []byte
}

func NewInMemoryProvider[T any](name string, value T, encoder Encoder[T], decoder Decoder[T], typ string, rw bool, metadata map[string]string) *InMemoryProvider[T] {
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["type"] = typ
	metadata["rw"] = fmt.Sprintf("%t", rw)
	return &InMemoryProvider[T]{
		name:     name,
		value:    value,
		encoder:  encoder,
		decoder:  decoder,
		metadata: metadata,
		rw:       rw,
		getterCh: make(chan []byte),
		setterCh: make(chan []byte),
	}
}

func (p *InMemoryProvider[T]) GetName() string {
	return p.name
}

func (p *InMemoryProvider[T]) GetValue() T {
	return p.value
}

func (p *InMemoryProvider[T]) SetValue(value T) {
	p.value = value
	p.getterCh <- p.encoder(p.value)
}

func (p *InMemoryProvider[T]) GetMetadata() (map[string]string, []byte) {
	return p.metadata, p.encoder(p.value)
}

func (p *InMemoryProvider[T]) GetChannels() (<-chan []byte, chan<- []byte) {
	return p.getterCh, p.setterCh
}

func NewInMemoryStringProvider(name string, value string, rw bool, metadata map[string]string) *InMemoryProvider[string] {
	return NewInMemoryProvider[string](name, value, encodeString, decodeString, "string", rw, metadata)
}

func NewInMemoryIntProvider(name string, value int, rw bool, metadata map[string]string) *InMemoryProvider[int] {
	return NewInMemoryProvider[int](name, value, encodeInt, decodeInt, "int", rw, metadata)
}
