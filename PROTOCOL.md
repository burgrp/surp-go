# SURP: Simple UDP Register Protocol

SURP (Simple UDP Register Protocol) is a lightweight, efficient binary protocol for transmitting named "register" values over UDP. It is designed for embedded systems, instrumentation, and control applications where devices expose internal state or accept commands over a network.

---

## Overview

A **register** is a named entity with:
- A **name** (string, globally unique)
- A **value** (bool, integer, float, string, etc.)
- Optional **metadata** (type info, unit, limits, etc.)

SURP enables:
- Providers to announce registers with values and metadata
- Consumers to query or subscribe to register updates
- Registries to track registers and route messages

---

## Roles

### Provider
- Owns and publishes registers
- Sends `IS` messages (state announcements)
- Receives `SET` messages (value update requests)

### Registry
- Central hub that:
  - Tracks registers and providers
  - Forwards updates to subscribers
  - Forwards `SET` to correct provider
  - Marks expired registers as undefined
- Identifies providers by **IP:port**
- Assumes **globally unique** register names

### Consumer
- Queries or subscribes to register state via `GET`
- Requests value changes via `SET`
- Receives `IS` messages for subscribed registers

---

## Transport

- All messages are **UDP datagrams**
- No handshaking or session state
- Registry is the central node; providers and consumers do not communicate directly

---

## Message Header (Common to All Messages)

| Field    | Size | Description                      |
|----------|------|----------------------------------|
| Version  | u8   | Protocol version (e.g. `0x01`)    |
| MsgType  | u8   | Message type (see below)         |

---

## Message Types

| MsgType | Name | Direction | Description |
|---------|------|-----------|-------------|
| 0x01    | IS   | Provider → Registry → Consumer | Register announcement or update |
| 0x02    | SET  | Consumer → Registry → Provider | Request to set register value |
| 0x03    | GET  | Consumer → Registry | Subscribe to register updates or request value |

---

## IS Message (Inform State)

Used by providers to send register value, metadata, and TTL. Also used by the registry to forward updates or mark a register as undefined.

### Format

| Field             | Type         | Description                                 |
|------------------|--------------|---------------------------------------------|
| TTL              | u16          | In seconds. 0 = static (never expires)      |
| NameLen          | u8           | Length of name                              |
| Name             | UTF-8 bytes  | Register name (max 255 bytes)               |
| ValueType        | u8           | Type of value (see below)                   |
| Value            | variable     | Encoded based on ValueType                  |
| MetadataCount    | u8           | Number of metadata entries                  |
| Metadata Entries | list         | See metadata format                         |

**Note:** If `ValueType == 0x00` (UNDEFINED), no value is included. Registry uses this to notify subscribers when a register times out.

---

## SET Message

Used by consumers to request a value change. Forwarded by registry to the appropriate provider.

### Format

| Field        | Type         |
|--------------|--------------|
| NameLen      | u8           |
| Name         | UTF-8 bytes  |
| ValueType    | u8           |
| Value        | variable     |

---

## GET Message

Used by consumers to request or subscribe to a register.

### Format

| Field       | Type         | Description                              |
|-------------|--------------|------------------------------------------|
| TTL         | u16          | In seconds. 0 = one-shot response only   |
| NameLen     | u8           |
| Name        | UTF-8 bytes  | Register name                            |

- When received, registry immediately replies with an `IS` message.
- If TTL > 0, registry stores the subscription and forwards future `IS` updates to the consumer.
- Subscriptions expire automatically after TTL; consumers must refresh them.

---

## Value Types (ValueType enum)

| Code | Type          | Encoding                          |
|------|---------------|-----------------------------------|
| 0x00 | UNDEFINED     | No value data                     |
| 0x01 | BOOL          | 1 byte: 0x00 = false, 0x01 = true |
| 0x02 | U8            | 1 byte unsigned                   |
| 0x03 | S8            | 1 byte signed                     |
| 0x04 | U16           | 2 bytes, big-endian               |
| 0x05 | S16           | 2 bytes, big-endian               |
| 0x06 | U32           | 4 bytes, big-endian               |
| 0x07 | S32           | 4 bytes, big-endian               |
| 0x08 | U64           | 8 bytes, big-endian               |
| 0x09 | S64           | 8 bytes, big-endian               |
| 0x0A | DOUBLE        | IEEE754, 8 bytes, big-endian      |
| 0x0B | SHORT_STRING  | u8 length + UTF-8 string          |
| 0x0C | LONG_STRING   | u16 length (BE) + UTF-8 string    |

---

## Metadata Format

Each IS message includes a list of metadata entries:

| Field        | Type         | Description                             |
|--------------|--------------|-----------------------------------------|
| Key          | u8           | See metadata keys below                 |
| ValueType    | u8           | Type of metadata value                  |
| Value        | variable     | Same encoding rules as register values  |

### Metadata Keys

| Key  | Name        | ValueType      | Description                         |
|------|-------------|----------------|-------------------------------------|
| 0x01 | TYPE        | ValueType enum | Redundant; explicitly defines type  |
| 0x02 | RW          | BOOL           | Read/write flag                     |
| 0x03 | MIN         | Matches reg.   | Minimum allowed value               |
| 0x04 | MAX         | Matches reg.   | Maximum allowed value               |
| 0x05 | UNIT        | SHORT_STRING   | Display unit (e.g., "°C")           |
| 0x06 | DESCRIPTION | SHORT/LONG_STRING | Human-readable description       |

---

## Provider Identification

- Registry identifies a provider by its **IP:port**
- If two different providers announce the same register name, it is considered a **conflict**
- Currently, register names must be globally unique

---

## TTL Handling

- `IS` includes a TTL (timeout for that register)
- When TTL expires without an update, registry:
  - Removes the value
  - Sends `IS` with `ValueType = UNDEFINED` to subscribers
- `GET` also uses TTL for subscriptions
  - Consumers must resend GET to maintain long-lived subscriptions

---

## Notes

- All integers are encoded in **big-endian (network order)**
- No acks or errors are defined; `SET` is a best-effort request
- Providers may clip or ignore `SET` values as appropriate
- Registry keeps no persistent state; protocol is robust to restarts
- Clean and extensible design for future message types and metadata
