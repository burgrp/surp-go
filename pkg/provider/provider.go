package provider

import (
	"fmt"

	surp "github.com/burgrp-go/surp/pkg"
)

type Register[T comparable] struct {
	name         string
	value        surp.Optional[T]
	encoder      surp.Encoder[T]
	decoder      surp.Decoder[T]
	rw           bool
	metadata     map[string]string
	setListener  SetListener[T]
	syncListener func()
}

type SetListener[T any] func(surp.Optional[T])

func NewRegister[T comparable](name string, value surp.Optional[T], encoder surp.Encoder[T], decoder surp.Decoder[T], typ string, rw bool, metadata map[string]string, setListener SetListener[T]) *Register[T] {
	if metadata == nil {
		metadata = map[string]string{}
	}

	metadata["type"] = typ
	metadata["rw"] = fmt.Sprintf("%t", rw)

	reg := &Register[T]{
		name:        name,
		value:       value,
		encoder:     encoder,
		decoder:     decoder,
		metadata:    metadata,
		rw:          rw,
		setListener: setListener,
	}

	return reg
}

func (reg *Register[T]) GetName() string {
	return reg.name
}

func (reg *Register[T]) GetValue() surp.Optional[T] {
	return reg.value
}

func (reg *Register[T]) Attach(syncListener func()) {
	reg.syncListener = syncListener
}

func (reg *Register[T]) SetEncodedValue(encodedValue surp.Optional[[]byte]) {
	if !reg.rw && reg.setListener != nil {
		return
	}

	decodedValue := surp.NewUndefined[T]()
	if encodedValue.IsDefined() {
		ev, ok := reg.decoder(encodedValue.Get())
		if ok {
			decodedValue = surp.NewDefined(ev)
		}
	}

	reg.setListener(decodedValue)
}

func (reg *Register[T]) SyncValue(value surp.Optional[T]) {
	if value != reg.value {
		reg.value = value
		if reg.syncListener != nil {
			reg.syncListener()
		}
	}
}

func (reg *Register[T]) GetEncodedValue() (surp.Optional[[]byte], map[string]string) {

	value := surp.NewUndefined[[]byte]()

	if reg.value.IsDefined() {
		value = surp.NewDefined(reg.encoder(reg.value.Get()))
	}

	return value, reg.metadata
}

func NewStringRegister(name string, value surp.Optional[string], rw bool, metadata map[string]string, listener SetListener[string]) *Register[string] {
	return NewRegister[string](name, value, surp.EncodeString, surp.DecodeString, "string", rw, metadata, listener)
}

func NewIntRegister(name string, value surp.Optional[int64], rw bool, metadata map[string]string, listener SetListener[int64]) *Register[int64] {
	return NewRegister[int64](name, value, surp.EncodeInt, surp.DecodeInt, "int", rw, metadata, listener)
}

func NewBoolRegister(name string, value surp.Optional[bool], rw bool, metadata map[string]string, listener SetListener[bool]) *Register[bool] {
	return NewRegister[bool](name, value, surp.EncodeBool, surp.DecodeBool, "bool", rw, metadata, listener)
}

func NewFloatRegister(name string, value surp.Optional[float64], rw bool, metadata map[string]string, listener SetListener[float64]) *Register[float64] {
	return NewRegister[float64](name, value, surp.EncodeFloat, surp.DecodeFloat, "float", rw, metadata, listener)
}

func NewAnyRegister(name string, value surp.Optional[any], typ string, rw bool, metadata map[string]string, listener SetListener[any]) *Register[any] {

	reg := NewRegister[any](name, value, func(value any) []byte {
		return surp.EncodeGeneric(value, typ)
	}, func(b []byte) (any, bool) {
		return surp.DecodeGeneric(b, typ)
	}, typ, rw, metadata, listener)

	return reg
}
