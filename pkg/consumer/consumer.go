package consumer

import surp "github.com/burgrp-go/surp/pkg"

type UpdateListener[T any] func(surp.Optional[T])

type Register[T comparable] struct {
	name      string
	value     surp.Optional[T]
	encoder   surp.Encoder[T]
	decoder   surp.Decoder[T]
	metadata  surp.Optional[map[string]string]
	listeners []UpdateListener[T]
	getterCh  chan surp.Optional[[]byte]
	setterCh  chan surp.Optional[[]byte]
}

func NewRegister[T comparable](name string, encoder surp.Encoder[T], decoder surp.Decoder[T], listeners ...UpdateListener[T]) *Register[T] {
	consumer := &Register[T]{
		name:      name,
		encoder:   encoder,
		decoder:   decoder,
		listeners: listeners,
		getterCh:  make(chan surp.Optional[[]byte]),
		setterCh:  make(chan surp.Optional[[]byte]),
	}

	go consumer.readUpdates()

	return consumer
}

func (p *Register[T]) GetName() string {
	return p.name
}

func (p *Register[T]) GetMetadata() surp.Optional[map[string]string] {
	return p.metadata
}

func (p *Register[T]) GetValue() surp.Optional[T] {
	return p.value
}

func (p *Register[T]) SetValue(value surp.Optional[T]) {
	if value.IsUndefined() {
		p.getterCh <- surp.NewUndefined[[]byte]()
	}
	p.getterCh <- surp.NewDefined(p.encoder(value.Get()))
}

func (p *Register[T]) SetMetadata(md map[string]string) {
	p.metadata = surp.NewDefined(md)
}

func (p *Register[T]) GetChannels() (<-chan surp.Optional[[]byte], chan<- surp.Optional[[]byte]) {
	return p.getterCh, p.setterCh
}

func (p *Register[T]) readUpdates() {
	for encodedValue := range p.setterCh {
		var newValue surp.Optional[T]
		if encodedValue.IsDefined() {
			newValue = surp.NewDefined(p.decoder(encodedValue.Get()))
		}
		if newValue != p.value {
			p.value = newValue
			for _, listener := range p.listeners {
				listener(p.value)
			}
		}
	}
}

func NewStringRegister(name string, listeners ...UpdateListener[string]) *Register[string] {
	return NewRegister[string](name, surp.EncodeString, surp.DecodeString, listeners...)
}

func NewIntRegister(name string, listeners ...UpdateListener[int]) *Register[int] {
	return NewRegister[int](name, surp.EncodeInt, surp.DecodeInt, listeners...)
}
