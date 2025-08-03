package surp

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"

	pb "github.com/burgrp/surp-go/pkg/pb"
)

// Socket handles sending and receiving SURP messages via UDP.
type Socket struct {
	conn        *net.UDPConn
	readBuf     []byte
	logger      *slog.Logger
	listeners   []MessageListener
	listenersMu sync.RWMutex
	closing     bool
}

// Socket calls these methods synchronously when a message is received.
// It is the responsibility of the listener to not block for too long.
// All errors should be handled within the listener.
// The listener should not modify the message.
type MessageListener interface {
	OnMessageIS(msg *pb.MessageIS, sender *net.UDPAddr)
	OnMessageGET(msg *pb.MessageGET, sender *net.UDPAddr)
	OnMessageSET(msg *pb.MessageSET, sender *net.UDPAddr)
}

// NewSocket creates and binds a UDP socket to the given address.
func NewSocket(listenAddr string, logger *slog.Logger) (*Socket, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	logger.Debug("Socket bound", "addr", listenAddr)

	return &Socket{
		conn:      conn,
		readBuf:   make([]byte, 1472), // max UDP packet size
		logger:    logger,
		listeners: []MessageListener{},
	}, nil
}

// StartReceiving starts reading messages and sends them to the returned channel.
func (s *Socket) Run(ctx context.Context) error {
	s.logger.Debug("Socket started")

	go func() {
		for {
			n, sender, err := s.conn.ReadFromUDP(s.readBuf)
			if err != nil {
				if !s.closing {
					s.logger.Error("Socket read error", "err", err)
				}
				break
			}
			var msg pb.SurpMessage
			if err := proto.Unmarshal(s.readBuf[:n], &msg); err != nil {
				s.logger.Debug("Failed to decode message", "err", err)
				continue
			}
			switch m := msg.Msg.(type) {
			case *pb.SurpMessage_Is:
				s.logger.Debug("Received IS", "from", sender.String(), "name", m.Is.GetName())
				s.handleMessageIS(m.Is, sender)
			case *pb.SurpMessage_Get:
				s.logger.Debug("Received GET", "from", sender.String(), "name", m.Get.GetName(), "ttl", m.Get.GetTtl())
				s.handleMessageGET(m.Get, sender)
			case *pb.SurpMessage_Set:
				s.logger.Debug("Received SET", "name", m.Set.GetName())
				s.handleMessageSET(m.Set, sender)
			default:
				s.logger.Debug("Unknown message type")
			}
		}
	}()

	<-ctx.Done()
	s.closing = true
	s.conn.Close()
	s.logger.Debug("Socket stopped")

	return nil
}

func (s *Socket) SendMessageIS(addr *net.UDPAddr, msg *pb.MessageIS) error {
	packet, err := proto.Marshal(&pb.SurpMessage{Msg: &pb.SurpMessage_Is{Is: msg}})
	if err != nil {
		return err
	}
	return s.sendPacket(addr, packet)
}

func (s *Socket) SendMessageGET(addr *net.UDPAddr, msg *pb.MessageGET) error {
	packet, err := proto.Marshal(&pb.SurpMessage{Msg: &pb.SurpMessage_Get{Get: msg}})
	if err != nil {
		return err
	}
	return s.sendPacket(addr, packet)
}

func (s *Socket) SendMessageSET(addr *net.UDPAddr, msg *pb.MessageSET) error {
	packet, err := proto.Marshal(&pb.SurpMessage{Msg: &pb.SurpMessage_Set{Set: msg}})
	if err != nil {
		return err
	}
	return s.sendPacket(addr, packet)
}

func (s *Socket) sendPacket(addr *net.UDPAddr, packet []byte) error {
	_, err := s.conn.WriteToUDP(packet, addr)
	if err != nil {
		s.logger.Debug("Failed to send packet", "to", addr.String(), "err", err)
	} else {
		s.logger.Debug("Message sent", "to", addr.String(), "size", len(packet))
	}
	return err
}

func (s *Socket) AddListener(listener MessageListener) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *Socket) handleMessageIS(msg *pb.MessageIS, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageIS(msg, sender)
	}
}

func (s *Socket) handleMessageGET(msg *pb.MessageGET, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageGET(msg, sender)
	}
}

func (s *Socket) handleMessageSET(msg *pb.MessageSET, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageSET(msg, sender)
	}
}
