package surp

import "fmt"

type InMemoryProvider[T any] struct {
	name     string
	value    Optional[T]
	encoder  Encoder[T]
	decoder  Decoder[T]
	rw       bool
	metadata map[string]string
	getterCh chan Optional[[]byte]
	setterCh chan Optional[[]byte]
}

func NewInMemoryProvider[T any](name string, value Optional[T], encoder Encoder[T], decoder Decoder[T], typ string, rw bool, metadata map[string]string) *InMemoryProvider[T] {
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
		getterCh: make(chan Optional[[]byte]),
		setterCh: make(chan Optional[[]byte]),
	}
}

func (p *InMemoryProvider[T]) GetName() string {
	return p.name
}

func (p *InMemoryProvider[T]) GetValue() Optional[T] {
	return p.value
}

func (p *InMemoryProvider[T]) getEncodedValue() Optional[[]byte] {
	if !p.value.IsValid() {
		return NewInvalid[[]byte]()
	}
	return NewValid(p.encoder(p.value.Get()))
}

func (p *InMemoryProvider[T]) SetValue(value Optional[T]) {
	p.value = value
	p.getterCh <- p.getEncodedValue()
}

func (p *InMemoryProvider[T]) GetMetadata() (map[string]string, Optional[[]byte]) {
	return p.metadata, p.getEncodedValue()
}

func (p *InMemoryProvider[T]) GetChannels() (<-chan Optional[[]byte], chan<- Optional[[]byte]) {
	return p.getterCh, p.setterCh
}

func NewInMemoryStringProvider(name string, value Optional[string], rw bool, metadata map[string]string) *InMemoryProvider[string] {
	return NewInMemoryProvider[string](name, value, encodeString, decodeString, "string", rw, metadata)
}

func NewInMemoryIntProvider(name string, value Optional[int], rw bool, metadata map[string]string) *InMemoryProvider[int] {
	return NewInMemoryProvider[int](name, value, encodeInt, decodeInt, "int", rw, metadata)
}
