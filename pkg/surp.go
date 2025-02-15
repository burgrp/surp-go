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
	[2 bytes] Port for unicast operations (address to be determined from the packet)

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
	consumer Consumer
	timeout  *time.Timer
	setIP    net.IP
	setPort  uint16
}

type providerWrapper struct {
	provider    Provider
	syncChannel chan struct{}
}

type RegisterGroup struct {
	name          string
	multicastPipe *UdpPipe
	unicastPipe   *UdpPipe
	rcvChannel    chan MessageAndAddr

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

	pipe, err := NewMulticastPipe(in, groupName, group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.multicastPipe = pipe

	pipe, err = NewUnicastPipe(in, group.rcvChannel)
	if err != nil {
		return nil, err
	}
	group.unicastPipe = pipe

	go group.readPipes()

	return group, nil
}

func (group *RegisterGroup) AddProviders(providers ...Provider) {

	for _, provider := range providers {

		providerWrapper := &providerWrapper{
			provider:    provider,
			syncChannel: make(chan struct{}),
		}

		group.providersMutex.Lock()
		group.providers[provider.GetName()] = providerWrapper
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

		wrapper := &consumerWrapper{
			consumer: consumer,
		}

		wrappers := group.consumers[consumer.GetName()]
		if wrappers == nil {
			wrappers = []*consumerWrapper{wrapper}
		} else {
			wrappers = append(wrappers, wrapper)
		}
		group.consumers[consumer.GetName()] = wrappers

		consumer.Attach(func(value Optional[[]byte]) {
			port := wrapper.setPort
			if port != 0 {
				message := &Message{
					SequenceNumber: group.nextSequenceNumber(),
					Type:           MessageTypeSet,
					Group:          group.name,
					Name:           wrapper.consumer.GetName(),
					Value:          value,
				}
				encoded := encodeMessage(message)
				group.unicastPipe.sndChannel <- MessageAndAddr{Message: encoded, Addr: &net.UDPAddr{IP: wrapper.setIP, Port: int(wrapper.setPort)}}
			}
		})

		group.multicastPipe.sndChannel <- MessageAndAddr{Message: encodeMessage(&Message{
			SequenceNumber: group.nextSequenceNumber(),
			Type:           MessageTypeGet,
			Group:          group.name,
			Name:           consumer.GetName(),
		}), Addr: group.multicastPipe.addr}
	}

}

func (group *RegisterGroup) Close() error {
	err := group.multicastPipe.Close()
	if err != nil {
		return err
	}

	err = group.multicastPipe.Close()
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
				wrapper.setPort = message.Port
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
	port := group.unicastPipe.conn.LocalAddr().(*net.UDPAddr).Port
	for {

		regular := time.After(MinSyncPeriod + time.Duration(rand.Intn(int(MaxSyncPeriod-MinSyncPeriod))))

		select {
		case <-regular:
			group.sendSyncMessage(providerWrapper.provider, port)
		case <-providerWrapper.syncChannel:
			group.sendSyncMessage(providerWrapper.provider, port)
		}

	}
}

func (group *RegisterGroup) sendSyncMessage(provider Provider, port int) {

	value, metadata := provider.GetEncodedValue()

	msg := &Message{
		SequenceNumber: group.nextSequenceNumber(),
		Type:           MessageTypeSync,
		Group:          group.name,
		Port:           uint16(port),
		Name:           provider.GetName(),
		Value:          value,
		Metadata:       metadata,
	}

	data := encodeMessage(msg)
	group.multicastPipe.sndChannel <- MessageAndAddr{Message: data, Addr: group.multicastPipe.addr}
}

func (group *RegisterGroup) OnSync(listener func(*Message)) {
	group.syncListener = listener
}
