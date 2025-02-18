/*
Simple UDP Register Protocol (SURP) - Lightweight M2M communication for IoT

Protocol Overview:
- Decentralized register-based communication using IPv6 multicast
- Three fundamental operations:
 1. Efficient value synchronization and register advertisement through multicast
 2. Set operations to RW registers
 3. Get operations to query RW/RO registers

Key Components:
1. Register Groups: Logical namespaces for register organization
2. Registers: Named data entities with:
  - Value: Dynamically typed payload
  - Metadata: Description, unit, data type, etc.
  - Lifecycle: Synced → Expired

Protocol Characteristics:
- Transport: UDP/IPv6 multicast (link-local scope ff02::/16)
- MTU: Optimized for ≤512 byte payloads
- Frequency: Periodic synchronization every 2-4 seconds or on value changes

Message Types:
- Sync (0x01): Broadcast value syncs
- Set (0x02): Register modification attempts
- Get (0x03): Challenge to send sync message

Addressing Scheme:
- IPv6 multicast address: ff02::cafe:face:1dea:1
- Port: Calculated for each register group and message type in range 1024-49151

Message Structure (Binary):

	[4 bytes]  Magic number "SURP"
	[1 byte]  Message type
	[2 bytes] Sequence number
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

	All messages share the same encoding.
	Sync message sets all fields.
	Set message has no metadata and port (ends after value).
	Get message has no value, metadata, or port (ends after register name).

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
	MessageTypeSync = 0x01
	MessageTypeSet  = 0x02
	MessageTypeGet  = 0x03
	SyncTimeout     = 10 * time.Second
	MinSyncPeriod   = 2 * time.Second
	MaxSyncPeriod   = 4 * time.Second
)

type Provider interface {
	GetName() string
	GetEncodedValue() (Optional[[]byte], map[string]string)
	SetEncodedValue(Optional[[]byte])
	Attach(syncListener func())
}

type Consumer interface {
	GetName() string
	SetMetadata(map[string]string)
	SyncValue(Optional[[]byte])
	Attach(setListener func(Optional[[]byte]))
}

type Encoder[T any] func(T) []byte
type Decoder[T any] func([]byte) (T, bool)

type consumerWrapper struct {
	consumer      Consumer
	timeout       *time.Timer
	setIP         net.IP
	setPort       uint16
	multicastAddr *net.UDPAddr
}

type providerWrapper struct {
	provider      Provider
	syncChannel   chan struct{}
	multicastAddr *net.UDPAddr
}

type RegisterGroup struct {
	name string

	multicastAddr   *net.UDPAddr
	multicastReader <-chan MessageAndAddr
	multicastClose  func() error

	unicastReader <-chan MessageAndAddr
	unicastWriter chan<- MessageAndAddr
	unicastClose  func() error

	rcvChannel chan MessageAndAddr

	providers      map[string]*providerWrapper
	providersMutex sync.Mutex

	consumers      map[string][]*consumerWrapper
	consumersMutex sync.Mutex

	sequenceNumber      uint16
	sequenceNumberMutex sync.Mutex

	syncListener func(*Message)
}

func JoinGroup(interfaceName string, groupName string) (*RegisterGroup, error) {

	in, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	group := &RegisterGroup{
		name:       groupName,
		rcvChannel: make(chan MessageAndAddr),
		providers:  make(map[string]*providerWrapper),
		consumers:  make(map[string][]*consumerWrapper),
	}

	group.multicastAddr = stringToMulticastAddr(groupName)

	group.multicastReader, group.multicastClose, err = listenMulticast(in, group.multicastAddr)
	if err != nil {
		return nil, err
	}

	group.unicastReader, group.unicastWriter, group.unicastClose, err = listenUnicast(in)
	if err != nil {
		return nil, err
	}

	go group.readPipes()

	return group, nil
}

func (group *RegisterGroup) AddProviders(providers ...Provider) {

	for _, provider := range providers {

		name := provider.GetName()

		providerWrapper := &providerWrapper{
			provider:      provider,
			syncChannel:   make(chan struct{}),
			multicastAddr: stringToMulticastAddr(group.name + ":" + name),
		}

		group.providersMutex.Lock()
		group.providers[name] = providerWrapper
		group.providersMutex.Unlock()

		provider.Attach(func() {
			providerWrapper.syncChannel <- struct{}{}
		})

		go group.syncLoop(providerWrapper)
	}
}

func (group *RegisterGroup) AddConsumers(consumers ...Consumer) {
	group.consumersMutex.Lock()
	defer group.consumersMutex.Unlock()

	for _, consumer := range consumers {

		name := consumer.GetName()

		wrapper := &consumerWrapper{
			consumer:      consumer,
			multicastAddr: stringToMulticastAddr(group.name + ":" + name),
		}

		wrappers := group.consumers[name]
		if wrappers == nil {
			wrappers = []*consumerWrapper{wrapper}
		} else {
			wrappers = append(wrappers, wrapper)
		}
		group.consumers[name] = wrappers

		consumer.Attach(func(value Optional[[]byte]) {
			port := wrapper.setPort
			if port != 0 {
				message := &Message{
					SequenceNumber: group.nextSequenceNumber(),
					Type:           MessageTypeSet,
					Group:          group.name,
					Name:           name,
					Value:          value,
				}
				encoded := encodeMessage(message)
				group.unicastWriter <- MessageAndAddr{Message: encoded, Addr: &net.UDPAddr{IP: wrapper.setIP, Port: int(wrapper.setPort)}}
			}
		})

		encoded := encodeMessage(&Message{
			SequenceNumber: group.nextSequenceNumber(),
			Type:           MessageTypeGet,
			Group:          group.name,
			Name:           name,
		})

		group.unicastWriter <- MessageAndAddr{Message: encoded, Addr: group.multicastAddr}
		group.unicastWriter <- MessageAndAddr{Message: encoded, Addr: wrapper.multicastAddr}
	}

}

func (group *RegisterGroup) Close() error {

	err := group.multicastClose()
	if err != nil {
		return err
	}

	err = group.unicastClose()
	if err != nil {
		return err
	}

	return nil
}

func (group *RegisterGroup) readPipes() {
	for m := range group.rcvChannel {

		if len(m.Message) < 4 || string(m.Message[:4]) != "SURP" {
			continue
		}

		message, ok := decodeMessage(m.Message[4:])
		if !ok {
			continue
		}

		if message.Group != group.name {
			continue
		}

		switch message.Type {
		case MessageTypeSync:

			group.consumersMutex.Lock()
			consumers := group.consumers[message.Name]
			for _, wrapper := range consumers {
				wrapper.setIP = m.Addr.IP
				wrapper.setPort = uint16(m.Addr.Port)
				wrapper.consumer.SetMetadata(message.Metadata)
				group.syncConsumerValue(wrapper, message.Value)
			}
			group.consumersMutex.Unlock()

			if group.syncListener != nil {
				group.syncListener(message)
			}

		case MessageTypeSet:
			group.providersMutex.Lock()
			providerWrapper := group.providers[message.Name]
			group.providersMutex.Unlock()

			if providerWrapper != nil {
				providerWrapper.provider.SetEncodedValue(message.Value)
			}

		case MessageTypeGet:
			group.providersMutex.Lock()
			providerWrapper := group.providers[message.Name]
			group.providersMutex.Unlock()

			if providerWrapper != nil {
				providerWrapper.syncChannel <- struct{}{}
			}

		default:
			fmt.Println("Received unknown message type")
		}
	}
}

func (group *RegisterGroup) syncConsumerValue(wrapper *consumerWrapper, value Optional[[]byte]) {

	if wrapper.timeout != nil {
		wrapper.timeout.Stop()
	}
	wrapper.timeout = time.AfterFunc(SyncTimeout, func() {
		wrapper.consumer.SyncValue(NewUndefined[[]byte]())
	})

	wrapper.consumer.SyncValue(value)
}

func (group *RegisterGroup) nextSequenceNumber() uint16 {
	group.sequenceNumberMutex.Lock()
	defer group.sequenceNumberMutex.Unlock()
	group.sequenceNumber++
	return group.sequenceNumber
}

func (group *RegisterGroup) syncLoop(providerWrapper *providerWrapper) {
	for {

		regular := time.After(MinSyncPeriod + time.Duration(rand.Intn(int(MaxSyncPeriod-MinSyncPeriod))))

		select {
		case <-regular:
			group.sendSyncMessage(providerWrapper)
		case <-providerWrapper.syncChannel:
			group.sendSyncMessage(providerWrapper)
		}

	}
}

func (group *RegisterGroup) sendSyncMessage(providerWrapper *providerWrapper) {

	name := providerWrapper.provider.GetName()

	value, metadata := providerWrapper.provider.GetEncodedValue()

	encoded := encodeMessage(&Message{
		SequenceNumber: group.nextSequenceNumber(),
		Type:           MessageTypeSync,
		Group:          group.name,
		Name:           name,
		Value:          value,
		Metadata:       metadata,
	})

	group.unicastWriter <- MessageAndAddr{Message: encoded, Addr: group.multicastAddr}
	group.unicastWriter <- MessageAndAddr{Message: encoded, Addr: providerWrapper.multicastAddr}
}

func (group *RegisterGroup) OnSync(listener func(*Message)) {
	group.syncListener = listener
}
