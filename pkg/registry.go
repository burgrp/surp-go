package surp

import (
	"context"
	"log/slog"
	"net"
	"reflect"
	"sync"
	"time"
)

// Register represents the state of a single SURP register.
type Register struct {
	Name      string
	ValueType ValueType
	Value     any
	Metadata  []MetadataEntry
	TTL       uint16
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
func (r *Registry) UpdateFromIS(msg *MessageIS, source *net.UDPAddr) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	existing, found := r.entries[msg.Name]

	entry := &Register{
		Name:      msg.Name,
		ValueType: msg.ValueType,
		Value:     msg.Value,
		Metadata:  msg.Metadata,
		TTL:       msg.TTL,
		UpdatedAt: now,
		Source:    source,
	}

	r.entries[msg.Name] = entry

	changed := !found || existing.ValueType != msg.ValueType || !reflect.DeepEqual(existing.Value, msg.Value)

	if changed {
		if !found {
			r.logger.Debug("Register added", "name", msg.Name, "type", msg.ValueType)
		} else {
			r.logger.Debug("Register updated", "name", msg.Name, "type", msg.ValueType)
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
func (r *Registry) Start(ctx context.Context, wg *sync.WaitGroup) {
	r.logger.Debug("Registry started")
	wg.Add(1)
	go func() {
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
		wg.Done()
	}()
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
