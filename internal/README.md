# Internal Packages

This directory contains internal implementation packages that are not part of Beacon's public API.

Go's `internal/` convention prevents external projects from importing these packages, allowing us to refactor and change internals without breaking compatibility.

## Structure

- `mdns/` - Multicast DNS (RFC 6762) implementation
- `dnssd/` - DNS Service Discovery (RFC 6763) implementation
- `protocol/` - Shared protocol primitives and utilities
