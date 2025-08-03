package registry

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
	pb "github.com/burgrp/surp-go/pkg/pb"
)

type subscription struct {
	addr    *net.UDPAddr
	ttl     uint32
	addedAt time.Time
}

// Binding bridges SURP messages over a Socket to a Registry.
type Binding struct {
	socket      *surp.Socket
	registry    *Registry
	subscribers map[string][]subscription
	mu          sync.Mutex
	logger      *slog.Logger
}

// NewBinding creates a binding between a Socket and a Registry.
func NewBinding(socket *surp.Socket, registry *Registry, logger *slog.Logger) *Binding {
	n := &Binding{
		socket:      socket,
		registry:    registry,
		subscribers: make(map[string][]subscription),
		logger:      logger,
	}

	registry.AddListener(n)
	socket.AddListener(n)
	return n
}

// Run begins processing messages from the socket.
func (b *Binding) Run(ctx context.Context) error {
	b.logger.Debug("Binding started")
	<-ctx.Done()
	b.logger.Debug("Binding bridge stopped")
	return nil
}

// OnRegisterUpdate is called by the Registry when a register is updated.
func (b *Binding) OnRegisterUpdate(r *Register) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[r.Name]
	if len(subs) == 0 {
		return
	}

	msg := &pb.MessageIS{
		Ttl:      r.TTL,
		Name:     r.Name,
		Value:    r.Value,
		Metadata: r.Metadata,
	}

	for _, s := range subs {
		err := b.socket.SendMessageIS(s.addr, msg)
		if err != nil {
			b.logger.Debug("Failed to send IS undefined", "name", r.Name, "to", s.addr.String(), "err", err)
			continue
		}
	}
}

// OnRegisterRemove is called when a register expires or is removed.
func (b *Binding) OnRegisterRemove(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[name]
	if len(subs) == 0 {
		return
	}

	msg := &pb.MessageIS{
		Ttl:  0,
		Name: name,
	}

	for _, s := range subs {
		err := b.socket.SendMessageIS(s.addr, msg)
		if err != nil {
			b.logger.Debug("Failed to send IS undefined", "name", name, "to", s.addr.String(), "err", err)
			continue
		}
	}
}

func (b *Binding) OnMessageGET(msg *pb.MessageGET, sender *net.UDPAddr) {
	if msg.GetTtl() > 0 {
		b.mu.Lock()
		b.subscribers[msg.GetName()] = append(b.subscribers[msg.GetName()], subscription{
			addr:    sender,
			ttl:     msg.GetTtl(),
			addedAt: time.Now(),
		})
		b.mu.Unlock()
		b.logger.Debug("Subscribed", "name", msg.GetName(), "ttl", msg.GetTtl(), "from", sender.String())
	}

	if reg, ok := b.registry.Get(msg.GetName()); ok && reg.Value != nil {
		isMsg := &pb.MessageIS{
			Ttl:      reg.TTL,
			Name:     reg.Name,
			Value:    reg.Value,
			Metadata: reg.Metadata,
		}
		err := b.socket.SendMessageIS(sender, isMsg)
		if err != nil {
			b.logger.Debug("Failed to send IS to subscriber", "name", msg.GetName(), "to", sender.String(), "err", err)
			return
		}
		b.logger.Debug("Sent IS to subscriber", "name", msg.GetName(), "to", sender.String())
	}
}

func (b *Binding) OnMessageSET(msg *pb.MessageSET, sender *net.UDPAddr) {
	reg, ok := b.registry.Get(msg.Name)
	if !ok || reg.Source == nil {
		b.logger.Debug("Cannot route SET — register not found or has no provider", "name", msg.Name)
		return
	}

	err := b.socket.SendMessageSET(reg.Source, msg)
	if err != nil {
		b.logger.Debug("Failed to send SET to provider", "name", msg.Name, "provider", reg.Source.String(), "err", err)
		return
	}
	b.logger.Debug("Sent SET to provider", "name", msg.Name, "provider", reg.Source.String())
}

func (b *Binding) OnMessageIS(msg *pb.MessageIS, sender *net.UDPAddr) {
}
