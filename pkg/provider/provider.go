package provider

import (
	"context"
	"net"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
)

type Provider struct {
	socket    *surp.Socket
	registry  *net.UDPAddr
	is        surp.MessageIS
	changed   chan any
	validator func(value any) any
}

func NewProvider(
	socket *surp.Socket,
	registry *net.UDPAddr,
	TTL uint16,
	Name string,
	ValueType surp.ValueType,
	Value any,
	Metadata []surp.MetadataEntry,
	validator func(value any) any,
) *Provider {

	provider := &Provider{
		socket:   socket,
		registry: registry,
		is: surp.MessageIS{
			TTL:       TTL,
			Name:      Name,
			ValueType: ValueType,
			Value:     Value,
			Metadata:  Metadata,
		},
		changed:   make(chan any),
		validator: validator,
	}

	socket.AddListener(provider)

	return provider
}

// Run starts the provider.
func (p *Provider) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(p.is.TTL) * time.Second / 2):
			p.sendIs()
		case newValue := <-p.changed:
			p.is.Value = newValue
			p.sendIs()
		}
	}
}

func (p *Provider) sendIs() {
	msg := p.is
	p.socket.SendMessageIS(p.registry, &msg)
}

// OnMessageGET implements surp.MessageListener.
func (p *Provider) OnMessageGET(msg *surp.MessageGET, sender *net.UDPAddr) {
}

// OnMessageIS implements surp.MessageListener.
func (p *Provider) OnMessageIS(msg *surp.MessageIS, sender *net.UDPAddr) {
}

// OnMessageSET implements surp.MessageListener.
func (p *Provider) OnMessageSET(msg *surp.MessageSET, sender *net.UDPAddr) {
	if msg.Name != p.is.Name {
		return
	}

	if msg.ValueType != p.is.ValueType {
		return
	}

	if p.validator != nil {
		msg.Value = p.validator(msg.Value)
	}

	p.is.Value = msg.Value

	p.sendIs()
}

func (p *Provider) SetValue(value any) {
	p.changed <- value
}

func (p *Provider) GetValue() any {
	return p.is.Value
}

func (p *Provider) GetName() string {
	return p.is.Name
}

func (p *Provider) GetTTL() uint16 {
	return p.is.TTL
}

func (p *Provider) GetValueType() surp.ValueType {
	return p.is.ValueType
}

func (p *Provider) GetMetadata() []surp.MetadataEntry {
	return p.is.Metadata
}

func (p *Provider) GetMetadataByKey(key surp.MetadataKey) any {
	for _, entry := range p.is.Metadata {
		if entry.Key == key {
			return entry.Value
		}
	}
	return nil
}
