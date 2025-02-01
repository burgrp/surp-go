package surp

type UpdateListener[T any] func(Optional[T])

type InMemoryConsumer[T comparable] struct {
	name      string
	value     Optional[T]
	encoder   Encoder[T]
	decoder   Decoder[T]
	metadata  Optional[map[string]string]
	listeners []UpdateListener[T]
	getterCh  chan Optional[[]byte]
	setterCh  chan Optional[[]byte]
}

func NewInMemoryConsumer[T comparable](name string, encoder Encoder[T], decoder Decoder[T], listeners ...UpdateListener[T]) *InMemoryConsumer[T] {
	consumer := &InMemoryConsumer[T]{
		name:      name,
		encoder:   encoder,
		decoder:   decoder,
		listeners: listeners,
		getterCh:  make(chan Optional[[]byte]),
		setterCh:  make(chan Optional[[]byte]),
	}

	go consumer.readUpdates()

	return consumer
}

func (p *InMemoryConsumer[T]) GetName() string {
	return p.name
}

func (p *InMemoryConsumer[T]) GetMetadata() Optional[map[string]string] {
	return p.metadata
}

func (p *InMemoryConsumer[T]) GetValue() Optional[T] {
	return p.value
}

func (p *InMemoryConsumer[T]) SetValue(value Optional[T]) {
	if !value.IsValid() {
		p.setterCh <- NewInvalid[[]byte]()
	}
	p.setterCh <- NewValid(p.encoder(value.Get()))
}

func (p *InMemoryConsumer[T]) SetMetadata(md map[string]string) {
	p.metadata = NewValid(md)
}

func (p *InMemoryConsumer[T]) GetChannels() (<-chan Optional[[]byte], chan<- Optional[[]byte]) {
	return p.getterCh, p.setterCh
}

func (p *InMemoryConsumer[T]) readUpdates() {
	for encodedValue := range p.setterCh {
		var newValue Optional[T]
		if encodedValue.IsValid() {
			newValue = NewValid(p.decoder(encodedValue.Get()))
		}
		if newValue != p.value {
			p.value = newValue
			for _, listener := range p.listeners {
				listener(p.value)
			}
		}
	}
}

func NewInMemoryStringConsumer(name string, listeners ...UpdateListener[string]) *InMemoryConsumer[string] {
	return NewInMemoryConsumer[string](name, encodeString, decodeString, listeners...)
}

func NewInMemoryIntConsumer(name string, listeners ...UpdateListener[int]) *InMemoryConsumer[int] {
	return NewInMemoryConsumer[int](name, encodeInt, decodeInt, listeners...)
}
