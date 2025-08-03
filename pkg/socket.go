package surp

import (
	"context"
	"log/slog"
	"net"
	"sync"
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
	OnMessageIS(msg *MessageIS, sender *net.UDPAddr)
	OnMessageGET(msg *MessageGET, sender *net.UDPAddr)
	OnMessageSET(msg *MessageSET, sender *net.UDPAddr)
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
			if n < 2 {
				s.logger.Debug("Discarding short packet", "bytes", n, "from", sender.String())
				continue
			}

			version, msgType, err := DecodeMessageHeader(s.readBuf[:n])
			if err != nil {
				s.logger.Debug("Invalid header", "err", err)
				continue
			}

			if version != ProtocolVersion1 {
				s.logger.Debug("Invalid protocol version", "version", version)
				continue
			}

			payload := s.readBuf[2:n]

			switch msgType {
			case MsgTypeIS:
				msgIS, err := DecodeMessageIS(payload)
				if err == nil {
					s.logger.Debug("Received IS", "from", sender.String(), "name", msgIS.Name)
					s.handleMessageIS(msgIS, sender)
				} else {
					s.logger.Debug("Failed to decode IS", "err", err)
				}
			case MsgTypeGET:
				msgGET, err := DecodeMessageGET(payload)
				if err == nil {
					s.logger.Debug("Received GET", "from", sender.String(), "name", msgGET.Name, "ttl", msgGET.TTL)
					s.handleMessageGET(msgGET, sender)
				} else {
					s.logger.Debug("Failed to decode GET", "err", err)
				}
			case MsgTypeSET:
				msgSET, err := DecodeMessageSET(payload)
				if err == nil {
					s.logger.Debug("Received SET", "name", msgSET.Name)
					s.handleMessageSET(msgSET, sender)
				} else {
					s.logger.Debug("Failed to decode SET", "err", err)
				}
			default:
				s.logger.Debug("Unknown message type", "msgType", msgType)
			}
		}
	}()

	<-ctx.Done()
	s.closing = true
	s.conn.Close()
	s.logger.Debug("Socket stopped")

	return nil
}

func (s *Socket) SendMessageIS(addr *net.UDPAddr, msg *MessageIS) error {
	encoded, err := EncodeMessageIS(msg)
	if err != nil {
		return err
	}
	return s.sendMessage(addr, MsgTypeIS, encoded)
}

func (s *Socket) SendMessageGET(addr *net.UDPAddr, msg *MessageGET) error {
	encoded, err := EncodeMessageGET(msg)
	if err != nil {
		return err
	}
	return s.sendMessage(addr, MsgTypeGET, encoded)
}

func (s *Socket) SendMessageSET(addr *net.UDPAddr, msg *MessageSET) error {
	encoded, err := EncodeMessageSET(msg)
	if err != nil {
		return err
	}
	return s.sendMessage(addr, MsgTypeSET, encoded)
}

func (s *Socket) sendMessage(addr *net.UDPAddr, msgType MsgType, body []byte) error {
	packet := make([]byte, 2+len(body))
	packet[0] = 0x01 // version
	packet[1] = byte(msgType)
	copy(packet[2:], body)
	_, err := s.conn.WriteToUDP(packet, addr)

	if err != nil {
		s.logger.Debug("Failed to send packet", "to", addr.String(), "err", err)
	} else {
		s.logger.Debug("Message sent", "type", msgType, "to", addr.String(), "size", len(packet))
	}
	return err
}

func (s *Socket) AddListener(listener MessageListener) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *Socket) handleMessageIS(msg *MessageIS, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageIS(msg, sender)
	}
}

func (s *Socket) handleMessageGET(msg *MessageGET, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageGET(msg, sender)
	}
}

func (s *Socket) handleMessageSET(msg *MessageSET, sender *net.UDPAddr) {
	s.listenersMu.RLock()
	defer s.listenersMu.RUnlock()
	for _, listener := range s.listeners {
		listener.OnMessageSET(msg, sender)
	}
}
