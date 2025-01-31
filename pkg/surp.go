/*
Simple UDP Register Protocol (SURP) - Lightweight M2M communication for IoT

Protocol Overview:
- Decentralized register-based communication using IPv6 multicast
- Three fundamental operations:
 1. Discovery: Devices advertise their registers
 2. Synchronization: Efficient value updates through multicast
 3. Control: Write operations to writable registers

Key Components:
1. Register Groups: Logical namespaces for register organization
2. Registers: Named data entities with:
  - Value: Dynamically typed payload
  - Metadata: Description, unit, data type, permissions
  - Lifecycle: Advertised → Updated → Expired

Protocol Characteristics:
- Transport: UDP/IPv6 multicast (link-local scope ff02::/16)
- Port Range: 5070-6070 (IANA reserved for experimental use)
- MTU: Optimized for ≤512 byte payloads
- Frequency: Periodic advertisements every 2-4 seconds

Message Types:
1. Advertise (0x01): Register metadata announcements
2. Update (0x02): Broadcast value updates
4. Write (0x03): Register modification attempts

Addressing Scheme:
- Base multicast address: ff02::[group_crc16]:[type]:[register_crc16]
  - group_crc16: CRC16-CCITT of register group name
  - type: 1=Control, 2=Data

- Predefined addresses:
  - Advertise:    ff02::[group_crc16]:1:1:5070
  - Update:       ff02::[group_crc16]:1:1:5071

Message Structure (Binary):
Advertise Message:

		[4 bytes]  Magic number "SURP"
		[1 byte]  Message type (0x01)
		[2 bytes] Sequence number
		[2 bytes] Register count (C)
		[C times]:
		  [1 byte]  Group name length (G)
		  [G bytes] Group name
		  [1 byte]  Register name length (N)
		  [N bytes] Register name
	      [2 bytes] Value length (V)
	      [V bytes] Value
		  [2 bytes] Metadata count (M)
		  [M times]:
		    [1 byte]  Key length (K)
		    [K bytes] Key
		    [1 byte]  Value length (V)
		    [V bytes] Value

Update Message:

	[1 byte]  Message type (0x02)
	[2 bytes] Sequence number
	[1 byte]  Group name length (G)
	[G bytes] Group name
	[2 bytes] Register name length (N)
	[N bytes] Register name
	[2 bytes] Value length (V)
	[V bytes] Value

Implementation Notes:
1. Security model assumes protected network layer
2. Requires multicast-enabled IPv6 network
3. CRC16 collisions handled via full name validation
4. Optimized for constrained devices (ESP32/RPi)
5. No QoS guarantees - application-layer reliability

See SURP specification for full protocol details.
*/
package surp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	messageTypeAdvertise = 0x01
	messageTypeUpdate    = 0x02
)

type Provider interface {
	GetName() string
	GetEncodedValue() []byte
	//SetEncodedValue([]byte)
	GetMetadata() map[string]string
}

type Consumer interface {
	GetName() string
	//GetEncodedValue() []byte
	SetEncodedValue([]byte)
	SetMetadata(map[string]string)
}

type Encoder[T any] func(T) []byte
type Decoder[T any] func([]byte) T

type RegisterGroup struct {
	name       string
	advertise  *UdpPipe
	updateConn *UdpPipe
	rcvChannel chan []byte

	providers      map[string]Provider
	providersMutex sync.Mutex

	consumers      map[string][]Consumer
	consumersMutex sync.Mutex
}

type AdvertisedRegister struct {
	RegisterName string
	Value        []byte
	Metadata     map[string]string
}

type AdvertiseMessage struct {
	SequenceNumber uint16
	GroupName      string
	Registers      []AdvertisedRegister
}

func JoinGroup(intf string, name string) (*RegisterGroup, error) {

	in, err := net.InterfaceByName(intf)
	if err != nil {
		return nil, err
	}

	group := &RegisterGroup{
		name:       name,
		rcvChannel: make(chan []byte),
		providers:  make(map[string]Provider),
		consumers:  make(map[string][]Consumer),
	}

	pipe, err := NewUdpPipe(in, fmt.Sprintf("%s:advertise", name), group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.advertise = pipe

	pipe, err = NewUdpPipe(in, fmt.Sprintf("%s:update", name), group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.updateConn = pipe

	go group.readPipes()
	go group.advertiseLoop()

	return group, nil
}

func (group *RegisterGroup) AddProviders(providers ...Provider) {
	group.providersMutex.Lock()
	defer group.providersMutex.Unlock()
	for _, provider := range providers {
		group.providers[provider.GetName()] = provider
	}
}

func (group *RegisterGroup) AddConsumers(consumers ...Consumer) {
	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, consumer := range consumers {
		cons := group.consumers[consumer.GetName()]
		if cons == nil {
			cons = []Consumer{
				consumer,
			}
		} else {
			cons = append(cons, consumer)
		}
		group.consumers[consumer.GetName()] = cons
	}
}

func (group *RegisterGroup) Close() error {
	err := group.advertise.Close()
	if err != nil {
		return err
	}

	err = group.updateConn.Close()
	if err != nil {
		return err
	}

	return nil
}

func (group *RegisterGroup) readPipes() {
	for msg := range group.rcvChannel {
		switch msg[0] {
		case messageTypeAdvertise:
			advertiseMessage, ok := decodeAdvertiseMessage(msg)
			if ok {
				group.handleAdvertiseMessage(advertiseMessage)
			}
		case messageTypeUpdate:
			fmt.Println("Received Update message")
		default:
			fmt.Println("Received unknown message type")
		}
	}
}

func (group *RegisterGroup) handleAdvertiseMessage(msg *AdvertiseMessage) {

	if group.name != msg.GroupName {
		return
	}

	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, r := range msg.Registers {
		consumers := group.consumers[r.RegisterName]
		for _, c := range consumers {
			c.SetMetadata(r.Metadata)
			c.SetEncodedValue(r.Value)
		}
	}
}

func decodeAdvertiseMessage(data []byte) (*AdvertiseMessage, bool) {

	remaining := data[:]

	if len(remaining) < 1 {
		return nil, false
	}
	if remaining[0] != messageTypeAdvertise {
		return nil, false
	}
	remaining = remaining[1:]

	msg := &AdvertiseMessage{}

	if len(remaining) < 2 {
		return nil, false
	}
	msg.SequenceNumber = binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	if len(remaining) < 1 {
		return nil, false
	}
	groupNameLen := int(remaining[0])
	remaining = remaining[1:]
	if len(remaining) < groupNameLen {
		return nil, false
	}
	msg.GroupName = string(remaining[:groupNameLen])
	remaining = remaining[groupNameLen:]

	registerCount := binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	for i := 0; i < int(registerCount); i++ {
		if len(remaining) < 1 {
			return nil, false
		}
		registerNameLen := int(remaining[0])
		remaining = remaining[1:]
		if len(remaining) < registerNameLen {
			return nil, false
		}
		registerName := string(remaining[:registerNameLen])
		remaining = remaining[registerNameLen:]

		if len(remaining) < 2 {
			return nil, false
		}
		valueLen := binary.BigEndian.Uint16(remaining[:2])
		remaining = remaining[2:]
		if len(remaining) < int(valueLen) {
			return nil, false
		}
		value := remaining[:valueLen]
		remaining = remaining[valueLen:]

		if len(remaining) < 2 {
			return nil, false
		}
		metadataCount := binary.BigEndian.Uint16(remaining[:2])
		remaining = remaining[2:]
		metadata := make(map[string]string, metadataCount)

		for j := 0; j < int(metadataCount); j++ {
			if len(remaining) < 1 {
				return nil, false
			}
			keyLen := int(remaining[0])
			remaining = remaining[1:]
			if len(remaining) < keyLen {
				return nil, false
			}
			k := string(remaining[:keyLen])
			remaining = remaining[keyLen:]

			if len(remaining) < 1 {
				return nil, false
			}
			valLen := int(remaining[0])
			remaining = remaining[1:]
			if len(remaining) < valLen {
				return nil, false
			}
			v := string(remaining[:valLen])
			remaining = remaining[valLen:]
			metadata[k] = v
		}

		msg.Registers = append(msg.Registers, AdvertisedRegister{
			RegisterName: registerName,
			Value:        value,
			Metadata:     metadata,
		})
	}

	return msg, true
}

func encodeAdvertiseMessage(msg *AdvertiseMessage) []byte {
	var buf bytes.Buffer

	buf.WriteByte(messageTypeAdvertise)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.GroupName)))
	buf.WriteString(msg.GroupName)
	binary.Write(&buf, binary.BigEndian, uint16(len(msg.Registers)))

	for _, r := range msg.Registers {
		buf.WriteByte(byte(len(r.RegisterName)))
		buf.WriteString(r.RegisterName)
		binary.Write(&buf, binary.BigEndian, uint16(len(r.Value)))
		buf.Write(r.Value)
		binary.Write(&buf, binary.BigEndian, uint16(len(r.Metadata)))
		for k, v := range r.Metadata {
			buf.WriteByte(byte(len(k)))
			buf.WriteString(k)
			buf.WriteByte(byte(len(v)))
			buf.WriteString(v)
		}
	}

	return buf.Bytes()
}

func (group *RegisterGroup) advertiseLoop() {
	seq := uint16(0)
	for {

		//TODO send multiple registers in one message, split if necessary to fit into MTU
		for _, p := range group.providers {

			msg := &AdvertiseMessage{
				SequenceNumber: seq,
				GroupName:      group.name,
				Registers: []AdvertisedRegister{
					{
						RegisterName: p.GetName(),
						Value:        p.GetEncodedValue(),
						Metadata:     p.GetMetadata(),
					},
				},
			}

			data := encodeAdvertiseMessage(msg)
			group.advertise.sndChannel <- data

			seq++
		}

		sleep := 2*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond
		time.Sleep(sleep)
	}
}
