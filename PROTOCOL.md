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
- Messages are encoded using **Protocol Buffers**
- Each datagram contains a `SurpMessage` with a `oneof` selecting
  `IS`, `SET`, or `GET`
- No handshaking or session state
- Registry is the central node; providers and consumers do not communicate directly

---

## IS Message (Inform State)

Used by providers to send register value, metadata, and TTL. Also used by the registry to forward updates to consumers.

### Format

| Field             | Type  | Description                                 |
|------------------|-------|---------------------------------------------|
| TTL              | u32   | In seconds. 0 = static (never expires)      |
| Name             | string| Register name                               |
| Value            | Value | Register value                              |
| Metadata         | list  | See metadata format                         |

If `Value` is unset, the register value is considered undefined.

Registry sends `IS` message with value`undefined` when last `IS` message from the provider expires.

---

## SET Message

Used by consumers to request a value change. Forwarded by registry to the appropriate provider.

### Format

| Field | Type  |
|-------|-------|
| Name  | string|
| Value | Value |

---

## GET Message

Used by consumers to request or subscribe to a register.

### Format

| Field | Type  | Description                            |
|-------|-------|----------------------------------------|
| TTL   | u32   | In seconds. 0 = one-shot response only |
| Name  | string| Register name                          |

- When received, registry immediately replies with an `IS` message.
- If registry does not know the register, it replies with value `undefined`.
- If TTL > 0, registry stores the subscription and forwards future `IS` updates to the consumer.
- Subscriptions expire automatically after TTL; consumers must refresh them.

---

## Value Representation

Values are encoded using a `oneof` that can carry one of:

- `bool_value`
- `u64_value`
- `s64_value`
- `f64_value`
- `str_value`

---

## Metadata Format

Each IS message includes a list of metadata entries:

| Field | Type | Description             |
|-------|------|-------------------------|
| Key   | u8   | See metadata keys below |
| Value | Value| Metadata value          |

### Metadata Keys

| Key  | Name        | Type          | Description                         |
|------|-------------|---------------|-------------------------------------|
| 0x01 | RO          | bool         | Read only flag                      |
| 0x02 | MIN         | matches reg. | Minimum allowed value               |
| 0x03 | MAX         | matches reg. | Maximum allowed value               |
| 0x04 | UNIT        | string       | Display unit (e.g., "°C")           |
| 0x05 | DESCRIPTION | string       | Human-readable description          |

---

## Provider Identification

- Registry identifies a provider by its **IP:port**
- If two different providers announce the same register name, it is considered a **conflict**
- Currently, register names must be globally unique

---

## TTL Handling

- `IS` includes a TTL. Both registry and consumers must check the TTL.
  - When TTL expires without an update:
    - Registry removes the value
    - Consumers treat the register as undefined
- `GET` uses TTL for subscriptions
  - Consumers must resend GET to maintain long-lived subscriptions

---

## Notes
- No acks or errors are defined; `SET` is a best-effort request
- Providers may clip or ignore `SET` values as appropriate
- Registry keeps no persistent state; protocol is robust to restarts
- Clean and extensible design for future message types and metadata
