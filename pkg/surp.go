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
	[2 bytes] Port for SET messages; 0 if all registers are read-only
	[2 bytes] Register count (C)
	[C times]:
		[1 byte]  Group name length (G)
		[G bytes] Group name
		[1 byte]  Register name length (N)
		[N bytes] Register name
		[2 bytes] Value length (V) (-1 if undefined/null, in which case the following array is empty)
		[V bytes] Value
		[1 bytes] Metadata count (M)
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
	[2 bytes] Value length (V) (see Value length in Advertise message)
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
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	messageTypeAdvertise = 0x01
	messageTypeUpdate    = 0x02
	messageTypeSet       = 0x03
	updateTimeout        = 10 * time.Second
	minAdvertisePeriod   = 2 * time.Second
	maxAdvertisePeriod   = 4 * time.Second
)

type Provider interface {
	GetName() string
	GetChannels() (<-chan Optional[[]byte], chan<- Optional[[]byte])
	GetMetadata() (map[string]string, Optional[[]byte])
}

type Consumer interface {
	GetName() string
	GetChannels() (<-chan Optional[[]byte], chan<- Optional[[]byte])
	SetMetadata(map[string]string)
}

type Encoder[T any] func(T) []byte
type Decoder[T any] func([]byte) T

type ConsumerWrapper struct {
	consumer Consumer
	timeout  *time.Timer
	getter   <-chan Optional[[]byte]
	setter   chan<- Optional[[]byte]
	setIP    net.IP
	setPort  uint16
}

type RegisterGroup struct {
	name          string
	advertisePipe *UdpPipe
	updatePipe    *UdpPipe
	setPipe       *UdpPipe
	rcvChannel    chan MessageAndAddr

	providers      map[string]Provider
	providersMutex sync.Mutex

	consumers      map[string][]*ConsumerWrapper
	consumersMutex sync.Mutex
}

func JoinGroup(interfaceName string, groupName string) (*RegisterGroup, error) {

	in, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	group := &RegisterGroup{
		name:       groupName,
		rcvChannel: make(chan MessageAndAddr),
		providers:  make(map[string]Provider),
		consumers:  make(map[string][]*ConsumerWrapper),
	}

	pipe, err := NewMulticastPipe(in, fmt.Sprintf("%s:advertise", groupName), group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.advertisePipe = pipe

	pipe, err = NewMulticastPipe(in, fmt.Sprintf("%s:update", groupName), group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.updatePipe = pipe

	pipe, err = NewUnicastPipe(in, group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.setPipe = pipe

	go group.readPipes()
	go group.advertiseLoop()

	return group, nil
}

func (group *RegisterGroup) AddProviders(providers ...Provider) {
	group.providersMutex.Lock()
	defer group.providersMutex.Unlock()
	for _, provider := range providers {
		group.providers[provider.GetName()] = provider
		go group.handleProvider(provider)
	}
}

func (group *RegisterGroup) handleProvider(provider Provider) {
	getterChannel, _ := provider.GetChannels()
	for {
		data := <-getterChannel
		msg := &UpdateMessage{
			SequenceNumber: 0,
			GroupName:      group.name,
			RegisterName:   provider.GetName(),
			Value:          data,
		}
		encoded := encodeUpdateMessage(msg)
		group.updatePipe.sndChannel <- MessageAndAddr{Message: encoded, Addr: group.updatePipe.addr}
	}
}

func (group *RegisterGroup) AddConsumers(consumers ...Consumer) {
	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, consumer := range consumers {

		getter, setter := consumer.GetChannels()

		wrapper := &ConsumerWrapper{
			consumer: consumer,
			getter:   getter,
			setter:   setter,
		}

		go group.handleConsumerWrapper(wrapper)

		wrappers := group.consumers[consumer.GetName()]
		if wrappers == nil {
			wrappers = []*ConsumerWrapper{wrapper}
		} else {
			wrappers = append(wrappers, wrapper)
		}
		group.consumers[consumer.GetName()] = wrappers
	}
}

func (group *RegisterGroup) handleConsumerWrapper(wrapper *ConsumerWrapper) {
	for value := range wrapper.getter {
		port := wrapper.setPort
		if port != 0 {
			message := &SetMessage{
				SequenceNumber: 0,
				GroupName:      group.name,
				RegisterName:   wrapper.consumer.GetName(),
				Value:          value,
			}
			encoded := encodeSetMessage(message)
			group.setPipe.sndChannel <- MessageAndAddr{Message: encoded, Addr: &net.UDPAddr{IP: wrapper.setIP, Port: int(wrapper.setPort)}}
		}
	}
}

func (group *RegisterGroup) Close() error {
	err := group.advertisePipe.Close()
	if err != nil {
		return err
	}

	err = group.updatePipe.Close()
	if err != nil {
		return err
	}

	return nil
}

func (group *RegisterGroup) readPipes() {
	for m := range group.rcvChannel {
		switch m.Message[0] {
		case messageTypeAdvertise:
			advertiseMessage, ok := decodeAdvertiseMessage(m.Message)
			if ok {
				group.handleAdvertiseMessage(advertiseMessage, m.Addr)
			}
		case messageTypeUpdate:
			updateMessage, ok := decodeUpdateMessage(m.Message)
			if ok {
				group.handleUpdateMessage(updateMessage)
			}

		case messageTypeSet:
			setMessage, ok := decodeSetMessage(m.Message)
			if ok {
				group.handleSetMessage(setMessage)
			}

		default:
			fmt.Println("Received unknown message type")
		}
	}
}

func (group *RegisterGroup) handleAdvertiseMessage(msg *AdvertiseMessage, ip *net.UDPAddr) {

	if group.name != msg.GroupName {
		return
	}

	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, register := range msg.Registers {
		consumers := group.consumers[register.Name]
		for _, wrapper := range consumers {
			wrapper.setIP = ip.IP
			wrapper.setPort = msg.Port
			wrapper.consumer.SetMetadata(register.Metadata)
			group.updateConsumerValue(wrapper, register.Value)
		}
	}
}

func (group *RegisterGroup) handleUpdateMessage(msg *UpdateMessage) {

	if group.name != msg.GroupName {
		return
	}

	consumers := group.consumers[msg.RegisterName]
	for _, wrapper := range consumers {
		group.updateConsumerValue(wrapper, msg.Value)
	}
}

func (group *RegisterGroup) handleSetMessage(msg *SetMessage) {

	if group.name != msg.GroupName {
		return
	}

	provider := group.providers[msg.RegisterName]
	if provider != nil {
		_, setter := provider.GetChannels()
		setter <- msg.Value
	}
}

func (group *RegisterGroup) updateConsumerValue(wrapper *ConsumerWrapper, value Optional[[]byte]) {

	if wrapper.timeout != nil {
		wrapper.timeout.Stop()
	}
	wrapper.timeout = time.AfterFunc(updateTimeout, func() {
		wrapper.setter <- NewInvalid[[]byte]()
	})

	wrapper.setter <- value
}

func (group *RegisterGroup) advertiseLoop() {
	port := group.setPipe.conn.LocalAddr().(*net.UDPAddr).Port
	seq := uint16(0)
	for {

		//TODO send multiple registers in one message, split if necessary to fit into MTU
		for _, p := range group.providers {

			metadata, value := p.GetMetadata()

			msg := &AdvertiseMessage{
				SequenceNumber: seq,
				GroupName:      group.name,
				Port:           uint16(port),
				Registers: []AdvertisedRegister{
					{
						Name:     p.GetName(),
						Value:    value,
						Metadata: metadata,
					},
				},
			}

			data := encodeAdvertiseMessage(msg)
			group.advertisePipe.sndChannel <- MessageAndAddr{Message: data, Addr: group.advertisePipe.addr}

			seq++
		}

		sleep := minAdvertisePeriod + time.Duration(rand.Intn(int(maxAdvertisePeriod-minAdvertisePeriod)))
		time.Sleep(sleep)
	}
}
