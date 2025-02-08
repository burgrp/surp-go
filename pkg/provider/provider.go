package provider

import (
	"fmt"

	surp "github.com/burgrp-go/surp/pkg"
)

type Register[T any] struct {
	name     string
	value    surp.Optional[T]
	encoder  surp.Encoder[T]
	decoder  surp.Decoder[T]
	rw       bool
	metadata map[string]string
	getterCh chan surp.Optional[[]byte]
	setterCh chan surp.Optional[[]byte]
	listener SetListener[T]
}

type SetListener[T any] func(surp.Optional[T])

func NewRegister[T any](name string, value surp.Optional[T], encoder surp.Encoder[T], decoder surp.Decoder[T], typ string, rw bool, metadata map[string]string, listener SetListener[T]) *Register[T] {
	if metadata == nil {
		metadata = map[string]string{}
	}

	metadata["type"] = typ
	metadata["rw"] = fmt.Sprintf("%t", rw)

	reg := &Register[T]{
		name:     name,
		value:    value,
		encoder:  encoder,
		decoder:  decoder,
		metadata: metadata,
		rw:       rw,
		getterCh: make(chan surp.Optional[[]byte]),
		setterCh: make(chan surp.Optional[[]byte]),
		listener: listener,
	}

	go reg.readSets()

	return reg
}

func (reg *Register[T]) readSets() {
	for encodedValue := range reg.setterCh {
		if !reg.rw && reg.listener != nil {
			continue
		}

		decodedValue := surp.NewInvalid[T]()
		if encodedValue.IsValid() {
			decodedValue = surp.NewValid(reg.decoder(encodedValue.Get()))
		}

		reg.listener(decodedValue)
	}
}

func (reg *Register[T]) GetName() string {
	return reg.name
}

func (reg *Register[T]) GetValue() surp.Optional[T] {
	return reg.value
}

func (reg *Register[T]) getEncodedValue() surp.Optional[[]byte] {
	if !reg.value.IsValid() {
		return surp.NewInvalid[[]byte]()
	}
	return surp.NewValid(reg.encoder(reg.value.Get()))
}

func (reg *Register[T]) SetValue(value surp.Optional[T]) {
	reg.value = value
	reg.getterCh <- reg.getEncodedValue()
}

func (reg *Register[T]) GetMetadata() (map[string]string, surp.Optional[[]byte]) {
	return reg.metadata, reg.getEncodedValue()
}

func (reg *Register[T]) GetChannels() (<-chan surp.Optional[[]byte], chan<- surp.Optional[[]byte]) {
	return reg.getterCh, reg.setterCh
}

func NewStringRegister(name string, value surp.Optional[string], rw bool, metadata map[string]string, listener SetListener[string]) *Register[string] {
	return NewRegister[string](name, value, surp.EncodeString, surp.DecodeString, "string", rw, metadata, listener)
}

func NewIntRegister(name string, value surp.Optional[int], rw bool, metadata map[string]string, listener SetListener[int]) *Register[int] {
	return NewRegister[int](name, value, surp.EncodeInt, surp.DecodeInt, "int", rw, metadata, listener)
}
