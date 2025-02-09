package surp

import (
	"fmt"
	"log"
	"net"
	"syscall"
)

const (
	maxUdpMessageSize = 1024
	ipv6Address       = "ff02::cafe:face:1dea:1"
)

type MessageAndAddr struct {
	Message []byte
	Addr    *net.UDPAddr
}

type UdpPipe struct {
	conn       *net.UDPConn
	addr       *net.UDPAddr
	rcvChannel chan MessageAndAddr
	sndChannel chan MessageAndAddr
}

func NewMulticastPipe(netInterface *net.Interface, pipeName string, rcvChannel chan MessageAndAddr) (*UdpPipe, error) {

	port := 1024 + int(CalculateHash(pipeName)&0xBBFF) // this gives us a port in the range 1024-49151

	addrStr := fmt.Sprintf("[%s]:%d", ipv6Address, port)

	log.Printf("Creating %s UDP pipe on %s", pipeName, addrStr)

	addr, err := net.ResolveUDPAddr("udp6", addrStr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenMulticastUDP("udp6", netInterface, addr)
	if err != nil {
		return nil, err
	}

	fd, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	err = syscall.SetsockoptInt(int(fd.Fd()), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}

	err = syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 1)
	if err != nil {
		return nil, err
	}

	pipe := &UdpPipe{
		conn:       conn,
		addr:       addr,
		rcvChannel: rcvChannel,
		sndChannel: make(chan MessageAndAddr),
	}

	go pipe.read()
	go pipe.write()

	return pipe, nil
}

func NewUnicastPipe(netInterface *net.Interface, rcvChannel chan MessageAndAddr) (*UdpPipe, error) {

	conn, err := net.ListenUDP("udp6", nil)
	if err != nil {
		return nil, err
	}

	pipe := &UdpPipe{
		conn:       conn,
		rcvChannel: rcvChannel,
		sndChannel: make(chan MessageAndAddr),
	}

	go pipe.read()
	go pipe.write()

	return pipe, nil
}

func (pipe *UdpPipe) read() {
	for {
		buf := make([]byte, maxUdpMessageSize)
		n, src, err := pipe.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		pipe.rcvChannel <- MessageAndAddr{
			Message: buf[:n],
			Addr:    &net.UDPAddr{IP: src.IP, Port: src.Port},
		}
	}
}

func (pipe *UdpPipe) write() {
	for m := range pipe.sndChannel {
		_, err := pipe.conn.WriteToUDP(m.Message, m.Addr)
		if err != nil {
			return
		}
	}
}

func (pipe *UdpPipe) Close() error {
	return pipe.conn.Close()
}
