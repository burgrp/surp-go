package surp

import (
	"fmt"
	"net"
	"syscall"
	"time"
)

const (
	maxUdpMessageSize = 1024
	ipv6Address       = "ff02::cafe:face:1dea:1"
)

type MessageAndAddr struct {
	Message []byte
	Addr    *net.UDPAddr
}

func stringToMulticastAddr(pipeName string) (*net.UDPAddr, error) {
	port := 1024 + int(CalculateHash(pipeName)&0xBBFF) // this gives us a port in the range 1024-49151
	addrStr := fmt.Sprintf("[%s]:%d", ipv6Address, port)

	addr, err := net.ResolveUDPAddr("udp6", addrStr)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

func listenMulticast(netInterface *net.Interface, addr *net.UDPAddr) (<-chan MessageAndAddr, error) {

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

	rcvChannel := make(chan MessageAndAddr)

	go func() {
		for {
			buf := make([]byte, maxUdpMessageSize)
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(rcvChannel)
				return
			}
			rcvChannel <- MessageAndAddr{
				Message: buf[:n],
				Addr:    &net.UDPAddr{IP: src.IP, Port: src.Port},
			}
		}
	}()

	return rcvChannel, nil
}

func listenUnicast(netInterface *net.Interface) (<-chan MessageAndAddr, chan<- MessageAndAddr, error) {

	addrs, err := netInterface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	var ip net.IP
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() == nil {
				ip = ipNet.IP
				break
			}
		}
	}

	conn, err := net.ListenUDP("udp6", &net.UDPAddr{IP: ip, Zone: netInterface.Name})
	if err != nil {
		return nil, nil, err
	}

	rcvChannel := make(chan MessageAndAddr)
	sndChannel := make(chan MessageAndAddr)

	go func() {
		for m := range sndChannel {
			conn.WriteToUDP(m.Message, m.Addr)
		}
	}()

	go func() {
		for {
			buf := make([]byte, maxUdpMessageSize)
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			rcvChannel <- MessageAndAddr{
				Message: buf[:n],
				Addr:    &net.UDPAddr{IP: src.IP, Port: src.Port},
			}
		}
	}()

	return rcvChannel, sndChannel, nil
}
