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

type UdpPipe struct {
	addr       *net.UDPAddr
	conn       *net.UDPConn
	rcvChannel chan []byte
	sndChannel chan []byte
}

func NewUdpPipe(netInterface *net.Interface, pipeName string, rcvChannel chan []byte) (*UdpPipe, error) {

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
		addr:       addr,
		conn:       conn,
		rcvChannel: rcvChannel,
		sndChannel: make(chan []byte),
	}

	go pipe.read()
	go pipe.write()

	return pipe, nil
}

func (pipe *UdpPipe) read() {
	for {
		buf := make([]byte, maxUdpMessageSize)
		n, _, err := pipe.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		pipe.rcvChannel <- buf[:n]
	}
}

func (pipe *UdpPipe) write() {
	for {
		buf := <-pipe.sndChannel
		_, err := pipe.conn.WriteToUDP(buf, pipe.addr)
		if err != nil {
			return
		}
	}
}

func (pipe *UdpPipe) Close() error {
	return pipe.conn.Close()
}
