/*
Simple UDP Register Protocol (SURP) - Lightweight M2M communication for IoT

Protocol Overview:
- Decentralized register-based communication using IPv6 multicast
- Three fundamental operations:
 1. Discovery: Devices advertise their registers
 2. Synchronization: Efficient value updates through multicast
 3. Control: Set operations to RW registers

Key Components:
1. Register Groups: Logical namespaces for register organization
2. Registers: Named data entities with:
  - Value: Dynamically typed payload
  - Metadata: Description, unit, data type, permissions
  - Lifecycle: Advertised → Updated → Expired

Protocol Characteristics:
- Transport: UDP/IPv6 multicast (link-local scope ff02::/16)
- MTU: Optimized for ≤512 byte payloads
- Frequency: Periodic advertisements every 2-4 seconds

Message Types:
1. Advertise (0x01): Register metadata announcements
2. Update (0x02): Broadcast value updates
4. Set0x03): Register modification attempts

Addressing Scheme:
- IPv6 multicast address: ff02::cafe:face:1dea:1
- Port: Calculated for each register group and message type in range 1024-49151

Message Structure (Binary):
Advertise Message:

	[4 bytes]  Magic number "SURP"
	[1 byte]  Message type (0x01)
	[2 bytes] Sequence number
	[2 bytes] Port for SET messages; 0 if all registers are read-only
	[1 bytes] Register count (C)
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

Update/Set Message:

	[4 bytes]  Magic number "SURP"
	[1 byte]  Message type (0x02 for update, 0x03 for set)
	[2 bytes] Sequence number
	[1 byte]  Group name length (G)
	[G bytes] Group name
	[1 byte] Register count (C)
	[C times]:
		[1 bytes] Register name length (N)
		[N bytes] Register name
		[2 bytes] Value length (V) (see Value length in Advertise message)
		[V bytes] Value

Join Message:

	[4 bytes]  Magic number "SURP"
	[1 byte]  Message type (0x04)
	[2 bytes] Sequence number
	[1 byte]  Group name length (G)
	[G bytes] Group name

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
	messageTypeJoin      = 0x04
	updateTimeout        = 10 * time.Second
	minAdvertisePeriod   = 2 * time.Second
	maxAdvertisePeriod   = 4 * time.Second
)

type Provider interface {
	GetName() string
	GetMetadata() (map[string]string, Optional[[]byte])
	SetValue(Optional[[]byte])
	Attach(updateListener func(Optional[[]byte]))
}

type Consumer interface {
	GetName() string
	SetMetadata(map[string]string)
	UpdateValue(Optional[[]byte])
	Attach(setListener func(Optional[[]byte]))
}

type Encoder[T any] func(T) []byte
type Decoder[T any] func([]byte) (T, bool)

type ConsumerWrapper struct {
	consumer Consumer
	timeout  *time.Timer
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

	sequenceNumber      uint16
	sequenceNumberMutex sync.Mutex

	joinChannel chan struct{}
}

func JoinGroup(interfaceName string, groupName string) (*RegisterGroup, error) {

	in, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	group := &RegisterGroup{
		name:        groupName,
		rcvChannel:  make(chan MessageAndAddr),
		providers:   make(map[string]Provider),
		consumers:   make(map[string][]*ConsumerWrapper),
		joinChannel: make(chan struct{}),
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
		provider.Attach(func(value Optional[[]byte]) {
			msg := &UpdateMessage{
				SequenceNumber: group.nextSequenceNumber(),
				GroupName:      group.name,
				Registers: []UpdatedRegister{
					{
						Name:  provider.GetName(),
						Value: value,
					},
				},
			}
			encoded := encodeUpdateMessage(msg)
			group.updatePipe.sndChannel <- MessageAndAddr{Message: encoded, Addr: group.updatePipe.addr}
		})
	}
}

func (group *RegisterGroup) AddConsumers(consumers ...Consumer) {
	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, consumer := range consumers {

		wrapper := &ConsumerWrapper{
			consumer: consumer,
		}

		wrappers := group.consumers[consumer.GetName()]
		if wrappers == nil {
			wrappers = []*ConsumerWrapper{wrapper}
		} else {
			wrappers = append(wrappers, wrapper)
		}
		group.consumers[consumer.GetName()] = wrappers

		consumer.Attach(func(value Optional[[]byte]) {
			port := wrapper.setPort
			if port != 0 {
				message := &SetMessage{
					SequenceNumber: group.nextSequenceNumber(),
					GroupName:      group.name,
					Registers: []UpdatedRegister{
						{
							Name:  wrapper.consumer.GetName(),
							Value: value,
						},
					},
				}
				encoded := encodeSetMessage(message)
				group.setPipe.sndChannel <- MessageAndAddr{Message: encoded, Addr: &net.UDPAddr{IP: wrapper.setIP, Port: int(wrapper.setPort)}}
			}
		})
	}

	group.advertisePipe.sndChannel <- MessageAndAddr{Message: encodeJoinMessage(&JoinMessage{
		SequenceNumber: group.nextSequenceNumber(),
		GroupName:      group.name,
	}), Addr: group.advertisePipe.addr}
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

		if len(m.Message) < 5 || string(m.Message[:4]) != "SURP" {
			continue
		}

		messageType := m.Message[4]
		messageData := m.Message[5:]

		switch messageType {
		case messageTypeAdvertise:
			advertiseMessage, ok := decodeAdvertiseMessage(messageData)
			if ok {
				group.handleAdvertiseMessage(advertiseMessage, m.Addr)
			}
		case messageTypeUpdate:
			updateMessage, ok := decodeUpdateMessage(messageData)
			if ok {
				group.handleUpdateMessage(updateMessage)
			}

		case messageTypeSet:
			setMessage, ok := decodeSetMessage(messageData)
			if ok {
				group.handleSetMessage(setMessage)
			}

		case messageTypeJoin:
			joinMessage, ok := decodeJoinMessage(messageData)
			if ok {
				group.handleJoinMessage(joinMessage, m.Addr)
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

	for _, register := range msg.Registers {
		consumers := group.consumers[register.Name]
		for _, wrapper := range consumers {
			group.updateConsumerValue(wrapper, register.Value)
		}
	}
}

func (group *RegisterGroup) handleSetMessage(msg *SetMessage) {

	if group.name != msg.GroupName {
		return
	}

	for _, register := range msg.Registers {
		provider := group.providers[register.Name]
		if provider != nil {
			provider.SetValue(register.Value)
		}
	}
}

func (group *RegisterGroup) handleJoinMessage(msg *JoinMessage, ip *net.UDPAddr) {

	if group.name != msg.GroupName {
		return
	}

	group.joinChannel <- struct{}{}
}

func (group *RegisterGroup) updateConsumerValue(wrapper *ConsumerWrapper, value Optional[[]byte]) {

	if wrapper.timeout != nil {
		wrapper.timeout.Stop()
	}
	wrapper.timeout = time.AfterFunc(updateTimeout, func() {
		wrapper.consumer.UpdateValue(NewUndefined[[]byte]())
	})

	wrapper.consumer.UpdateValue(value)
}

func (group *RegisterGroup) nextSequenceNumber() uint16 {
	group.sequenceNumberMutex.Lock()
	defer group.sequenceNumberMutex.Unlock()
	group.sequenceNumber++
	return group.sequenceNumber
}

func (group *RegisterGroup) advertiseLoop() {
	port := group.setPipe.conn.LocalAddr().(*net.UDPAddr).Port
	for {

		regular := time.After(minAdvertisePeriod + time.Duration(rand.Intn(int(maxAdvertisePeriod-minAdvertisePeriod))))

		select {

		case <-regular:
			group.sendAdvertiseMessage(port)
		case <-group.joinChannel:
			group.sendAdvertiseMessage(port)
		}

	}
}

func (group *RegisterGroup) sendAdvertiseMessage(port int) {
	//TODO send multiple registers in one message, split if necessary to fit into MTU
	for _, p := range group.providers {

		metadata, value := p.GetMetadata()

		msg := &AdvertiseMessage{
			SequenceNumber: group.nextSequenceNumber(),
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
	}
}
