package registry

import (
	"context"
	"log/slog"
	"net"
	"reflect"
	"sync"
	"time"

	pb "github.com/burgrp/surp-go/pkg/pb"
)

// Register represents the state of a single SURP register.
type Register struct {
	Name      string
	Value     *pb.Value
	Metadata  []*pb.MetadataEntry
	TTL       uint32
	UpdatedAt time.Time
	Source    *net.UDPAddr
}

// Listener receives updates when register state changes or is removed.
type Listener interface {
	OnRegisterUpdate(r *Register)
	OnRegisterRemove(name string)
}

// Registry is an in-memory store for SURP registers.
type Registry struct {
	mu        sync.RWMutex
	entries   map[string]*Register
	listeners []Listener
	ticker    *time.Ticker
	logger    *slog.Logger
}

// NewRegistry creates a new register store and starts the TTL reaper.
func NewRegistry(logger *slog.Logger) *Registry {
	r := &Registry{
		entries:   make(map[string]*Register),
		listeners: []Listener{},
		ticker:    time.NewTicker(time.Second),
		logger:    logger,
	}
	return r
}

// AddListener registers a listener to receive change events.
func (r *Registry) AddListener(l Listener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = append(r.listeners, l)
}

// UpdateFromIS inserts or updates a register based on an IS message.
func (r *Registry) UpdateFromIS(msg *pb.MessageIS, source *net.UDPAddr) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	existing, found := r.entries[msg.GetName()]

	entry := &Register{
		Name:      msg.GetName(),
		Value:     msg.GetValue(),
		Metadata:  msg.GetMetadata(),
		TTL:       msg.GetTtl(),
		UpdatedAt: now,
		Source:    source,
	}

	r.entries[msg.GetName()] = entry

	changed := !found || !reflect.DeepEqual(existing.Value, msg.GetValue())

	if changed {
		if !found {
			r.logger.Debug("Register added", "name", msg.GetName())
		} else {
			r.logger.Debug("Register updated", "name", msg.GetName())
		}
		for _, l := range r.listeners {
			l.OnRegisterUpdate(entry)
		}
	}
}

// Get retrieves a register by name.
func (r *Registry) Get(name string) (*Register, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.entries[name]
	return reg, ok
}

// Close stops the TTL reaper and shuts down the registry.
func (r *Registry) Run(ctx context.Context) error {
	r.logger.Debug("Registry started")
loop:
	for {
		select {
		case <-r.ticker.C:
			r.expireStaleRegisters()
		case <-ctx.Done():
			break loop
		}
	}
	r.ticker.Stop()
	r.logger.Debug("Registry stopped")

	return nil
}

// Internal: Scan for and remove expired registers.
func (r *Registry) expireStaleRegisters() {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, entry := range r.entries {
		if entry.TTL == 0 {
			continue
		}
		if now.Sub(entry.UpdatedAt) > time.Duration(entry.TTL)*time.Second {
			r.logger.Debug("Register expired", "name", name)
			delete(r.entries, name)
			for _, l := range r.listeners {
				l.OnRegisterRemove(name)
			}
		}
	}
}
