package provider

import (
	"net"

	surp "github.com/burgrp/surp-go/pkg"
)

type Provider struct {
	socket *surp.Socket
}

func NewProvider(socket *surp.Socket) *Provider {

	provider := &Provider{
		socket: socket,
	}

	socket.AddListener(provider)

	return provider
}

// OnMessageGET implements surp.MessageListener.
func (p *Provider) OnMessageGET(msg *surp.MessageGET, sender *net.UDPAddr) {
	panic("unimplemented")
}

// OnMessageIS implements surp.MessageListener.
func (p *Provider) OnMessageIS(msg *surp.MessageIS, sender *net.UDPAddr) {
	panic("unimplemented")
}

// OnMessageSET implements surp.MessageListener.
func (p *Provider) OnMessageSET(msg *surp.MessageSET, sender *net.UDPAddr) {
	panic("unimplemented")
}
