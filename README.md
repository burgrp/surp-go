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

TODO

## Wireshark
To analyze SURP protocol messages with Wireshark, set the WIRESHARK_PLUGIN_DIR environment variable to the wireshark directory in this repository and start Wireshark:

```sh
WIRESHARK_PLUGIN_DIR=$PWD/wireshark wireshark
```

## CLI

### Installation

Download binary for your platform from the latest [release](https://github.com/burgrp/surp-go).

### Compilation

```sh
go mod tidy
go build -o surp cmd/surp/main.go
```

### Reference

#### surp

surp is a command line tool for working with registers over SURP protocol.

##### Synopsis

The surp command is a command line tool for working with registers over SURP protocol.
It allows you to read, write and list registers.
Furthermore it can provide a 'virtual' register which is convenient for debugging of consumers of the register.

Two environment variables are required:
- SURP_IF: The network interface to bind to
- SURP_GROUP: The SURP group name to join

For more information on registers over SURP, see: https://github.com/burgrp/surp-go .

##### Options

```
  -h, --help   help for surp
```

##### SEE ALSO

* [surp get](#surp-get)	 - Read a register
* [surp list](#surp-list)	 - List all known registers
* [surp provide](#surp-provide)	 - Provide a register
* [surp set](#surp-set)	 - Write a register
* [surp version](#surp-version)	 - Show version

#### surp get

Read a register

##### Synopsis

Reads the specified register.
	With --stay flag, the command will remain connected and write any changes to stdout.

```
surp get <register> [flags]
```

##### Options

```
  -h, --help   help for get
  -s, --stay   Stay connected and write changes to stdout
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

#### surp help

Help about any command

##### Synopsis

Help provides help for any command in the application.
Simply type surp help [path to command] for full details.

```
surp help [command] [flags]
```

##### Options

```
  -h, --help   help for help
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

#### surp list

List all known registers

##### Synopsis

Lists all known registers.
	With --stay flag, the command will remain connected and write any changes to stdout.
	If registers are specified, only those will be listed.

```
surp list [<reg1> <reg2> ...] [flags]
```

##### Options

```
  -h, --help               help for list
  -m, --meta               Do not print metadata
  -s, --stay               Stay connected infinitely and write changes to stdout
  -t, --timeout duration   Timeout for waiting for the registers (default 10s)
  -v, --values             Do not print values
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

#### surp provide

Provide a register

##### Synopsis

Provides a register with the specified name, value and metadata.
Subsequent values are read from stdin and are written to stdout.
Default type is int, if not specified otherwise in metadata.

```
surp provide <name> <value> [meta-key:meta-value ...] [flags]
```

##### Options

```
  -h, --help        help for provide
  -r, --read-only   Make the register read-only.
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

#### surp set

Write a register

##### Synopsis

Writes the specified register.
With --stay flag, the command will remain connected, read values from stdin and write any changes to stdout.
Values are specified as JSON expressions, e.g. true, false, 3.14, "hello world" or null.

```
surp set <register> <value> [flags]
```

##### Options

```
  -h, --help               help for set
  -s, --stay               Stay connected, read values from stdin and write changes to stdout
  -o, --timeout duration   Timeout for waiting for the register to be set (default 10s)
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

#### surp version

Show version

##### Synopsis

Shows version of reg command line tool.

```
surp version [flags]
```

##### Options

```
  -h, --help   help for version
```

##### SEE ALSO

* [surp](#surp)	 - surp is a command line tool for working with registers over SURP protocol.

