package surp

type InMemoryConsumer[T any] struct {
	Name      string
	Value     T
	encoder   Encoder[T]
	decoder   Decoder[T]
	metadata  map[string]string
	listeners []func(T)
}

func NewInMemoryConsumer[T any](name string, encoder Encoder[T], decoder Decoder[T], listeners ...func(T)) *InMemoryConsumer[T] {
	return &InMemoryConsumer[T]{
		Name:      name,
		encoder:   encoder,
		decoder:   decoder,
		metadata:  map[string]string{},
		listeners: listeners,
	}
}

func (p *InMemoryConsumer[T]) GetName() string {
	return p.Name
}

func (p *InMemoryConsumer[T]) GetEncodedValue() []byte {
	return []byte(p.encoder(p.Value))
}

func (p *InMemoryConsumer[T]) SetEncodedValue(value []byte) {
	p.Value = p.decoder(value)
	for _, listener := range p.listeners {
		listener(p.Value)
	}
}

func (p *InMemoryConsumer[T]) SetMetadata(md map[string]string) {
	p.metadata = md
}

func NewInMemoryStringConsumer(name string, listeners ...func(string)) *InMemoryConsumer[string] {
	return NewInMemoryConsumer[string](name, encodeString, decodeString, listeners...)
}

func NewInMemoryIntConsumer(name string, listeners ...func(int)) *InMemoryConsumer[int] {
	return NewInMemoryConsumer[int](name, encodeInt, decodeInt, listeners...)
}
