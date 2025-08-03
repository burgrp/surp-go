package provider

import (
	"context"
	"net"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
	pb "github.com/burgrp/surp-go/pkg/pb"
)

type Provider struct {
	socket    *surp.Socket
	registry  *net.UDPAddr
	is        pb.MessageIS
	changed   chan *pb.Value
	validator func(value *pb.Value) *pb.Value
}

func NewProvider(
	socket *surp.Socket,
	registry *net.UDPAddr,
	TTL uint32,
	Name string,
	Value *pb.Value,
	Metadata []*pb.MetadataEntry,
	validator func(value *pb.Value) *pb.Value,
) *Provider {

	provider := &Provider{
		socket:   socket,
		registry: registry,
		is: pb.MessageIS{
			Ttl:      TTL,
			Name:     Name,
			Value:    Value,
			Metadata: Metadata,
		},
		changed:   make(chan *pb.Value),
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
		case <-time.After(time.Duration(p.is.Ttl) * time.Second / 2):
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
func (p *Provider) OnMessageGET(msg *pb.MessageGET, sender *net.UDPAddr) {
}

// OnMessageIS implements surp.MessageListener.
func (p *Provider) OnMessageIS(msg *pb.MessageIS, sender *net.UDPAddr) {
}

// OnMessageSET implements surp.MessageListener.
func (p *Provider) OnMessageSET(msg *pb.MessageSET, sender *net.UDPAddr) {
	if msg.GetName() != p.is.GetName() {
		return
	}

	val := msg.GetValue()
	if p.validator != nil {
		val = p.validator(val)
	}

	p.is.Value = val

	p.sendIs()
}

func (p *Provider) SetValue(value *pb.Value) {
	p.changed <- value
}

func (p *Provider) GetValue() *pb.Value {
	return p.is.Value
}

func (p *Provider) GetName() string {
	return p.is.GetName()
}

func (p *Provider) GetTTL() uint32 {
	return p.is.GetTtl()
}

func (p *Provider) GetMetadata() []*pb.MetadataEntry {
	return p.is.GetMetadata()
}

func (p *Provider) GetMetadataByKey(key pb.MetadataKey) *pb.Value {
	return surp.GetMetadataValue(p.is.GetMetadata(), key)
}
