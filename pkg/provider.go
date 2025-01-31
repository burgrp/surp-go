package surp

import "fmt"

type InMemoryProvider[T any] struct {
	name     string
	value    T
	encoder  Encoder[T]
	decoder  Decoder[T]
	rw       bool
	metadata map[string]string
}

func NewInMemoryProvider[T any](name string, encoder Encoder[T], decoder Decoder[T], typ string, rw bool, metadata map[string]string) *InMemoryProvider[T] {
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["type"] = typ
	metadata["rw"] = fmt.Sprintf("%t", rw)
	return &InMemoryProvider[T]{
		name:     name,
		encoder:  encoder,
		decoder:  decoder,
		metadata: metadata,
		rw:       rw,
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
}

func (p *InMemoryProvider[T]) GetEncodedValue() []byte {
	return []byte(p.encoder(p.value))
}

func (p *InMemoryProvider[T]) SetEncodedValue(value []byte) {
	p.value = p.decoder(value)
}

func (p *InMemoryProvider[T]) GetMetadata() map[string]string {
	return p.metadata
}

func NewInMemoryStringProvider(name string, rw bool, metadata map[string]string) *InMemoryProvider[string] {
	return NewInMemoryProvider[string](name, encodeString, decodeString, "string", rw, metadata)
}

func NewInMemoryIntProvider(name string, rw bool, metadata map[string]string) *InMemoryProvider[int] {
	return NewInMemoryProvider[int](name, encodeInt, decodeInt, "int", rw, metadata)
}
