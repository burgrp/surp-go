package surp

import (
	"context"
	"log/slog"
	"net"
	"slices"
	"sync"
	"time"
)

// Message represents a parsed SURP message.
type Message struct {
	Version uint8
	Type    MsgType
	Payload []byte
	Sender  *net.UDPAddr
	Err     error
}

// Socket handles sending and receiving SURP messages via UDP.
type Socket struct {
	conn             *net.UDPConn
	readBuf          []byte
	logger           *slog.Logger
	ReceivedMessages chan Message
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
		conn:    conn,
		readBuf: make([]byte, 1472), // max UDP packet size
		logger:  logger,
	}, nil
}

// StartReceiving starts reading messages and sends them to the returned channel.
func (s *Socket) Start(ctx context.Context, wg *sync.WaitGroup) {
	out := make(chan Message)
	s.ReceivedMessages = out
	s.logger.Info("Socket started")

	go func() {
		for {
			n, sender, err := s.conn.ReadFromUDP(s.readBuf)
			if err != nil {
				if err == net.ErrClosed {
					s.logger.Debug("Socket closed")
					break
				}
				s.logger.Debug("Socket read error", "err", err)
				continue
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

			payload := slices.Clone(s.readBuf[2:n])

			out <- Message{
				Version: version,
				Type:    msgType,
				Payload: payload,
				Sender:  sender,
				Err:     err,
			}
		}
	}()

	wg.Add(1)
	go func() {
		<-ctx.Done()
		s.logger.Info("Socket stopped")
		close(out)
		s.conn.Close()
		wg.Done()
	}()
}

// WriteMessage encodes and sends a SURP message to the target address.
func (s *Socket) WriteMessage(addr *net.UDPAddr, msgType MsgType, body []byte) error {
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

// Close closes the underlying UDP connection.
func (s *Socket) Close() error {
	s.logger.Info("Socket closed")
	return s.conn.Close()
}

// SetReadDeadline sets a read timeout.
func (s *Socket) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}
