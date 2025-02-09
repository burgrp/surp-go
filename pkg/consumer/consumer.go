package consumer

import surp "github.com/burgrp-go/surp/pkg"

type UpdateListener[T any] func(surp.Optional[T])

type Register[T comparable] struct {
	name            string
	value           surp.Optional[T]
	encoder         surp.Encoder[T]
	decoder         surp.Decoder[T]
	metadata        surp.Optional[map[string]string]
	updateListeners []UpdateListener[T]
	setListener     func(surp.Optional[[]byte])
}

func NewRegister[T comparable](name string, encoder surp.Encoder[T], decoder surp.Decoder[T], listeners ...UpdateListener[T]) *Register[T] {
	consumer := &Register[T]{
		name:            name,
		encoder:         encoder,
		decoder:         decoder,
		updateListeners: listeners,
	}

	return consumer
}

func (reg *Register[T]) GetName() string {
	return reg.name
}

func (reg *Register[T]) GetMetadata() surp.Optional[map[string]string] {
	return reg.metadata
}

func (reg *Register[T]) UpdateValue(encodedValue surp.Optional[[]byte]) {

	var newValue surp.Optional[T]
	if encodedValue.IsDefined() {
		newValue = surp.NewDefined(reg.decoder(encodedValue.Get()))
	}
	if newValue != reg.value {
		reg.value = newValue
		for _, listener := range reg.updateListeners {
			listener(reg.value)
		}
	}
}

func (reg *Register[T]) Attach(setListener func(surp.Optional[[]byte])) {
	reg.setListener = setListener
}

func (reg *Register[T]) GetValue() surp.Optional[T] {
	return reg.value
}

func (reg *Register[T]) SetValue(value surp.Optional[T]) {
	if reg.setListener != nil {
		if value.IsUndefined() {
			reg.setListener(surp.NewUndefined[[]byte]())
		}
		reg.setListener(surp.NewDefined(reg.encoder(value.Get())))
	}
}

func (reg *Register[T]) SetMetadata(md map[string]string) {
	reg.metadata = surp.NewDefined(md)
}

func NewStringRegister(name string, listeners ...UpdateListener[string]) *Register[string] {
	return NewRegister[string](name, surp.EncodeString, surp.DecodeString, listeners...)
}

func NewIntRegister(name string, listeners ...UpdateListener[int64]) *Register[int64] {
	return NewRegister[int64](name, surp.EncodeInt, surp.DecodeInt, listeners...)
}

func NewBoolRegister(name string, listeners ...UpdateListener[bool]) *Register[bool] {
	return NewRegister[bool](name, surp.EncodeBool, surp.DecodeBool, listeners...)
}

func NewFloatRegister(name string, listeners ...UpdateListener[float64]) *Register[float64] {
	return NewRegister[float64](name, surp.EncodeFloat, surp.DecodeFloat, listeners...)
}
