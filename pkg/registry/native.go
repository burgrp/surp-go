package registry

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
)

type subscription struct {
	addr    *net.UDPAddr
	ttl     uint16
	addedAt time.Time
}

// Native bridges SURP messages over a Socket to a Registry.
type Native struct {
	socket      *surp.Socket
	registry    *Registry
	subscribers map[string][]subscription
	mu          sync.Mutex
	logger      *slog.Logger
}

// NewNative creates a Native bridge between a Socket and a Registry.
func NewNative(socket *surp.Socket, registry *Registry, logger *slog.Logger) *Native {
	n := &Native{
		socket:      socket,
		registry:    registry,
		subscribers: make(map[string][]subscription),
		logger:      logger,
	}

	registry.AddListener(n)
	return n
}

// Run begins processing messages from the socket.
func (n *Native) Start(ctx context.Context, wg *sync.WaitGroup) {
	n.logger.Debug("Native bridge started")
	wg.Add(1)
	go func() {
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case msg := <-n.socket.ReceivedMessages:
				if msg.Version != surp.ProtocolVersion1 {
					n.logger.Debug("Invalid protocol version", "version", msg.Version)
					continue
				}

				switch msg.Type {
				case surp.MsgTypeIS:
					msgIS, err := surp.DecodeMessageIS(msg.Payload)
					if err == nil {
						n.logger.Debug("Received IS", "from", msg.Sender.String(), "name", msgIS.Name)
						n.registry.UpdateFromIS(msgIS, msg.Sender)
					} else {
						n.logger.Debug("Failed to decode IS", "err", err)
					}
				case surp.MsgTypeGET:
					msgGET, err := surp.DecodeMessageGET(msg.Payload)
					if err == nil {
						n.logger.Debug("Received GET", "from", msg.Sender.String(), "name", msgGET.Name, "ttl", msgGET.TTL)
						n.handleGET(msgGET, msg.Sender)
					} else {
						n.logger.Debug("Failed to decode GET", "err", err)
					}
				case surp.MsgTypeSET:
					msgSET, err := surp.DecodeMessageSET(msg.Payload)
					if err == nil {
						n.logger.Debug("Received SET", "name", msgSET.Name)
						n.handleSET(msgSET)
					} else {
						n.logger.Debug("Failed to decode SET", "err", err)
					}
				default:
					n.logger.Debug("Unknown message type", "msgType", msg.Type)
				}
			}
		}
		n.logger.Debug("Native bridge stopped")
		wg.Done()
	}()
}

// OnRegisterUpdate is called by the Registry when a register is updated.
func (n *Native) OnRegisterUpdate(r *Register) {
	n.mu.Lock()
	defer n.mu.Unlock()

	subs := n.subscribers[r.Name]
	if len(subs) == 0 {
		return
	}

	msg := surp.MessageIS{
		TTL:       r.TTL,
		Name:      r.Name,
		ValueType: r.ValueType,
		Value:     r.Value,
		Metadata:  r.Metadata,
	}
	encoded, err := surp.EncodeMessageIS(msg)
	if err != nil {
		n.logger.Debug("Failed to encode IS for update", "name", r.Name, "err", err)
		return
	}

	n.logger.Debug("Forwarding IS update", "name", r.Name, "subs", len(subs))
	for _, s := range subs {
		n.socket.WriteMessage(s.addr, surp.MsgTypeIS, encoded)
	}
}

// OnRegisterRemove is called when a register expires or is removed.
func (n *Native) OnRegisterRemove(name string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	subs := n.subscribers[name]
	if len(subs) == 0 {
		return
	}

	msg := surp.MessageIS{
		TTL:       0,
		Name:      name,
		ValueType: surp.ValueUndefined,
	}
	encoded, err := surp.EncodeMessageIS(msg)
	if err != nil {
		n.logger.Debug("Failed to encode IS (undefined)", "name", name, "err", err)
		return
	}

	n.logger.Debug("Forwarding IS undefined", "name", name, "subs", len(subs))
	for _, s := range subs {
		n.socket.WriteMessage(s.addr, surp.MsgTypeIS, encoded)
	}
}

// handleGET processes a GET message, stores a subscription, and sends current value if known.
func (n *Native) handleGET(msg *surp.MessageGET, sender *net.UDPAddr) {
	if msg.TTL > 0 {
		n.mu.Lock()
		n.subscribers[msg.Name] = append(n.subscribers[msg.Name], subscription{
			addr:    sender,
			ttl:     msg.TTL,
			addedAt: time.Now(),
		})
		n.mu.Unlock()
		n.logger.Debug("Subscribed", "name", msg.Name, "ttl", msg.TTL, "from", sender.String())
	}

	if reg, ok := n.registry.Get(msg.Name); ok && reg.ValueType != surp.ValueUndefined {
		isMsg := surp.MessageIS{
			TTL:       reg.TTL,
			Name:      reg.Name,
			ValueType: reg.ValueType,
			Value:     reg.Value,
			Metadata:  reg.Metadata,
		}
		if encoded, err := surp.EncodeMessageIS(isMsg); err == nil {
			n.socket.WriteMessage(sender, surp.MsgTypeIS, encoded)
			n.logger.Debug("Sent immediate IS response", "name", reg.Name, "to", sender.String())
		}
	}
}

// handleSET forwards a SET message to the original provider.
func (n *Native) handleSET(msg *surp.MessageSET) {
	reg, ok := n.registry.Get(msg.Name)
	if !ok || reg.Source == nil {
		n.logger.Debug("Cannot route SET — register not found or has no provider", "name", msg.Name)
		return
	}

	if encoded, err := surp.EncodeMessageSET(*msg); err == nil {
		n.socket.WriteMessage(reg.Source, surp.MsgTypeSET, encoded)
		n.logger.Debug("Forwarded SET to provider", "name", msg.Name, "provider", reg.Source.String())
	}
}
