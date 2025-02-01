package surp

type InMemoryConsumer[T comparable] struct {
	name      string
	value     T
	encoder   Encoder[T]
	decoder   Decoder[T]
	metadata  map[string]string
	listeners []func(T)
	getterCh  chan []byte
	setterCh  chan []byte
}

func NewInMemoryConsumer[T comparable](name string, encoder Encoder[T], decoder Decoder[T], listeners ...func(T)) *InMemoryConsumer[T] {
	consumer := &InMemoryConsumer[T]{
		name:      name,
		encoder:   encoder,
		decoder:   decoder,
		metadata:  map[string]string{},
		listeners: listeners,
		getterCh:  make(chan []byte),
		setterCh:  make(chan []byte),
	}

	go consumer.readUpdates()

	return consumer
}

func (p *InMemoryConsumer[T]) GetName() string {
	return p.name
}

func (p *InMemoryConsumer[T]) GetMetadata() map[string]string {
	return p.metadata
}

func (p *InMemoryConsumer[T]) GetValue() T {
	return p.value
}

func (p *InMemoryConsumer[T]) SetValue(value T) {
	p.setterCh <- p.encoder(value)
}

func (p *InMemoryConsumer[T]) SetMetadata(md map[string]string) {
	p.metadata = md
}

func (p *InMemoryConsumer[T]) GetChannels() (<-chan []byte, chan<- []byte) {
	return p.getterCh, p.setterCh
}

func (p *InMemoryConsumer[T]) readUpdates() {
	for encodedValue := range p.setterCh {
		var decodedValue T
		if len(encodedValue) != 0 {
			decodedValue = p.decoder(encodedValue)
		}
		if decodedValue != p.value {
			p.value = decodedValue
			for _, listener := range p.listeners {
				listener(p.value)
			}
		}
	}
}

func NewInMemoryStringConsumer(name string, listeners ...func(string)) *InMemoryConsumer[string] {
	return NewInMemoryConsumer[string](name, encodeString, decodeString, listeners...)
}

func NewInMemoryIntConsumer(name string, listeners ...func(int)) *InMemoryConsumer[int] {
	return NewInMemoryConsumer[int](name, encodeInt, decodeInt, listeners...)
}
