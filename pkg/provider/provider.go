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
}

func NewRegister[T any](name string, value surp.Optional[T], encoder surp.Encoder[T], decoder surp.Decoder[T], typ string, rw bool, metadata map[string]string) *Register[T] {
	if metadata == nil {
		metadata = map[string]string{}
	}

	metadata["type"] = typ
	metadata["rw"] = fmt.Sprintf("%t", rw)
	return &Register[T]{
		name:     name,
		value:    value,
		encoder:  encoder,
		decoder:  decoder,
		metadata: metadata,
		rw:       rw,
		getterCh: make(chan surp.Optional[[]byte]),
		setterCh: make(chan surp.Optional[[]byte]),
	}
}

func (p *Register[T]) GetName() string {
	return p.name
}

func (p *Register[T]) GetValue() surp.Optional[T] {
	return p.value
}

func (p *Register[T]) getEncodedValue() surp.Optional[[]byte] {
	if !p.value.IsValid() {
		return surp.NewInvalid[[]byte]()
	}
	return surp.NewValid(p.encoder(p.value.Get()))
}

func (p *Register[T]) SetValue(value surp.Optional[T]) {
	p.value = value
	p.getterCh <- p.getEncodedValue()
}

func (p *Register[T]) GetMetadata() (map[string]string, surp.Optional[[]byte]) {
	return p.metadata, p.getEncodedValue()
}

func (p *Register[T]) GetChannels() (<-chan surp.Optional[[]byte], chan<- surp.Optional[[]byte]) {
	return p.getterCh, p.setterCh
}

func NewStringRegister(name string, value surp.Optional[string], rw bool, metadata map[string]string) *Register[string] {
	return NewRegister[string](name, value, surp.EncodeString, surp.DecodeString, "string", rw, metadata)
}

func NewIntRegister(name string, value surp.Optional[int], rw bool, metadata map[string]string) *Register[int] {
	return NewRegister[int](name, value, surp.EncodeInt, surp.DecodeInt, "int", rw, metadata)
}
