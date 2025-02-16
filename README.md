# SURP Implementation in Go

SURP (Simple UDP Register Protocol) is a lightweight M2M communication protocol designed for IoT devices. This repository contains the Go implementation of SURP.

## Protocol Overview

SURP is a decentralized register-based communication protocol using IPv6 multicast. It supports three fundamental operations:
1. Efficient value synchronization and register advertisement through multicast
2. Set operations to RW registers
3. Get operations to query RW/RO registers

### Key Components

1. **Register Groups**: Logical namespaces for register organization
2. **Registers**: Named data entities with:
   - Value: Dynamically typed payload
   - Metadata: Description, unit, data type, etc.
   - Lifecycle: Synced → Expired

### Protocol Characteristics

- **Transport**: UDP/IPv6 multicast (link-local scope ff02::/16)
- **MTU**: Optimized for ≤512 byte payloads
- **Frequency**: Periodic synchronization every 2-4 seconds or on value changes

### Message Types

- **Sync (0x01)**: Broadcast value syncs
- **Set (0x02)**: Register modification attempts
- **Get (0x03)**: Challenge to send sync message

### Addressing Scheme

- **IPv6 multicast address**: ff02::cafe:face:1dea:1
- **Port**: Calculated for each register group and message type in range 1024-49151

### Message Structure (Binary)

- `[4 bytes]`  Magic number "SURP"
- `[1 byte]`  Message type
- `[2 bytes]` Sequence number
- `[1 byte]`  Group name length (G)
- `[G bytes]` Group name
- `[1 byte]`  Register name length (N)
- `[N bytes]` Register name
- `[2 bytes]` Value length (V) (-1 if undefined/null, in which case the following array is empty)
- `[V bytes]` Value
- `[1 byte]` Metadata count (M)
- `[M times]`:
  - `[1 byte]`  Key length (K)
  - `[K bytes]` Key
  - `[1 byte]`  Value length (V)
  - `[V bytes]` Value
- `[2 bytes]` Port for unicast operations (address to be determined from the packet)

All messages share the same encoding. Sync message sets all fields. Set message has no metadata and port (ends after value). Get message has no value, metadata, or port (ends after register name).

### Implementation Notes

1. Security model assumes protected network layer
2. Requires multicast-enabled IPv6 network
3. CRC16 collisions handled via full name validation
4. Optimized for constrained devices (ESP32/RPi)
5. No QoS guarantees - application-layer reliability

## Library Usage

To use the SURP library in your Go application, import the `surp` package and follow the examples below:

```go
import "github.com/burgrp/surp-go/pkg/surp"

// Example usage for creating and joining a register group
group, err := surp.JoinGroup("eth0", "exampleGroup")
if err != nil {
    log.Fatalf("Failed to join group: %v", err)
}

// Adding providers and consumers to the group
group.AddProviders(provider1, provider2)
group.AddConsumers(consumer1, consumer2)
```

## CLI
The SURP CLI can be used to interact with the protocol from the command line.

## Wireshark
To analyze SURP protocol messages with Wireshark, set the WIRESHARK_PLUGIN_DIR environment variable to the wireshark directory in this repository and start Wireshark:

```sh
WIRESHARK_PLUGIN_DIR=$PWD/wireshark wireshark
```