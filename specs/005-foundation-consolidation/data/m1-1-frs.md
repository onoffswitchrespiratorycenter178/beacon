# M1.1 Functional Requirements (FR-M1.1-001 through FR-M1.1-035)

**Source**: `specs/004-m1-1-architectural-hardening/spec.md`
**Milestone**: M1.1 (Architectural Hardening - Socket Config, Interface Management, Security)
**Task**: T007 (R004) - Extract and convert to milestone-prefixed IDs

---

## Socket Configuration (F-9) - 10 FRs

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1.1-001 | System MUST use `net.ListenConfig` with Control function to set socket options BEFORE bind() | ✅ Implemented | internal/transport/udp.go (ListenConfig pattern) | - | tests/transport/TestSocketOptionsApplied |
| FR-M1.1-002 | System MUST set SO_REUSEADDR on Linux, macOS, and Windows for multicast binding | ✅ Implemented | internal/transport/socket_linux.go, socket_darwin.go, socket_windows.go | - | tests/transport/TestSocketOptions_Linux |
| FR-M1.1-003 | System MUST set SO_REUSEPORT on Linux (kernel >= 3.9) and macOS for port sharing with system daemons | ✅ Implemented (Linux/macOS) | internal/transport/socket_linux.go, socket_darwin.go | RFC 6762 §15 | tests/integration/TestAvahiCoexistence |
| FR-M1.1-004 | System MUST NOT use `net.ListenMulticastUDP()` due to Go Issues #73484 and #34728 | ✅ Implemented | internal/transport/udp.go (ListenConfig used instead) | - | Code review validation |
| FR-M1.1-005 | System MUST join multicast group 224.0.0.251 using `golang.org/x/net/ipv4` package per RFC 6762 §5 | ✅ Implemented | internal/transport/udp.go (uses ipv4.PacketConn) | RFC 6762 §5.2 | tests/integration/TestMulticastJoin |
| FR-M1.1-006 | System MUST set multicast TTL to 255 per RFC 6762 §11 | ✅ Implemented | internal/transport/udp.go (SetMulticastTTL(255)) | RFC 6762 §11 | tests/contract/TestMulticastTTL |
| FR-M1.1-007 | System MUST enable multicast loopback for local testing | ✅ Implemented | internal/transport/udp.go (SetMulticastLoopback(true)) | - | tests/integration/TestLocalLoopback |
| FR-M1.1-008 | System MUST detect Linux kernel version and log warning if < 3.9 (SO_REUSEPORT not guaranteed) | ✅ Implemented | internal/transport/socket_linux.go (kernel version check) | - | tests/transport/TestKernelVersionCheck_Linux |
| FR-M1.1-009 | System MUST use platform-specific files (build tags) for socket option configuration | ✅ Implemented | socket_linux.go, socket_darwin.go, socket_windows.go (build tags) | - | Build system validation |
| FR-M1.1-010 | System MUST propagate socket initialization errors with context (interface, operation, cause) | ✅ Implemented | internal/transport/udp.go (errors.NetworkError wrapping) | - | tests/integration/TestSocketInitErrors |

## Interface Management (F-10) - 12 FRs

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1.1-011 | System MUST provide `WithInterfaces([]net.Interface)` functional option for explicit interface selection | ✅ Implemented | querier/options.go:176-185 | - | tests/querier/TestWithInterfaces |
| FR-M1.1-012 | System MUST provide `WithInterfaceFilter(func(net.Interface) bool)` functional option for custom filtering | ✅ Implemented | querier/options.go:187-196 | - | tests/querier/TestWithInterfaceFilter |
| FR-M1.1-013 | System MUST implement `DefaultInterfaces()` function that returns filtered list of suitable interfaces | ✅ Implemented | internal/network/interfaces.go (DefaultInterfaces) | - | tests/network/TestDefaultInterfaces |
| FR-M1.1-014 | `DefaultInterfaces()` MUST include only interfaces with `net.FlagUp` set (active interfaces) | ✅ Implemented | internal/network/interfaces.go (flag checks) | - | tests/network/TestDefaultInterfaces_FlagUp |
| FR-M1.1-015 | `DefaultInterfaces()` MUST include only interfaces with `net.FlagMulticast` set (multicast-capable) | ✅ Implemented | internal/network/interfaces.go (flag checks) | - | tests/network/TestDefaultInterfaces_Multicast |
| FR-M1.1-016 | `DefaultInterfaces()` MUST exclude loopback interfaces (127.0.0.1, ::1) | ✅ Implemented | internal/network/interfaces.go (loopback check) | - | tests/network/TestDefaultInterfaces_ExcludeLoopback |
| FR-M1.1-017 | `DefaultInterfaces()` MUST exclude VPN interfaces by pattern: utun*, tun*, ppp*, wg*, tailscale*, wireguard* | ✅ Implemented | internal/network/interfaces.go (VPN patterns) | - | tests/network/TestDefaultInterfaces_ExcludeVPN |
| FR-M1.1-018 | `DefaultInterfaces()` MUST exclude Docker interfaces by pattern: docker0, veth*, br-* | ✅ Implemented | internal/network/interfaces.go (Docker patterns) | - | tests/network/TestDefaultInterfaces_ExcludeDocker |
| FR-M1.1-019 | System MUST validate each interface exists and is suitable for mDNS before binding | ✅ Implemented | internal/transport/udp.go (interface validation) | - | tests/integration/TestInterfaceValidation |
| FR-M1.1-020 | System MUST log interface selection decisions at debug level (interface evaluated, filter decision, reason) | ✅ Implemented | internal/network/interfaces.go (debug logging) | - | tests/network/TestInterfaceLogging_Debug |
| FR-M1.1-021 | System MUST log selected interfaces at info level (user-visible confirmation) | ✅ Implemented | querier/querier.go (info logging) | - | tests/querier/TestInterfaceLogging_Info |
| FR-M1.1-022 | When no interfaces match filters, system MUST fail initialization with error listing rejected interfaces | ✅ Implemented | querier/querier.go (error on empty interface list) | - | tests/querier/TestNoInterfacesError |

## Security (F-11) - 13 FRs

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1.1-023 | System MUST validate source IP is link-local (169.254.0.0/16) OR same subnet as receiving interface | ✅ Implemented | internal/security/source_filter.go (IsValidSource) | RFC 6762 §2, §11 | tests/security/TestSourceFilter_LinkLocal |
| FR-M1.1-024 | System MUST silently drop packets with non-link-local source IPs BEFORE parsing | ✅ Implemented | querier/querier.go (filter in receive loop before ParseMessage) | - | tests/querier/TestDropNonLinkLocalSources |
| FR-M1.1-025 | System MUST log dropped packets at debug level with source IP and reason | ✅ Implemented | querier/querier.go (debug log on drop) | - | tests/querier/TestDroppedPacketLogging |
| FR-M1.1-026 | System MUST implement per-source-IP rate limiting with sliding window (1 second) | ✅ Implemented | internal/security/rate_limiter.go (RateLimiter.Allow) | RFC 6762 §6 (traffic reduction) | tests/security/TestRateLimiter_SlidingWindow |
| FR-M1.1-027 | Rate limiter MUST have configurable threshold (default: 100 queries/second) | ✅ Implemented | internal/security/rate_limiter.go (threshold parameter) | - | tests/security/TestRateLimiter_ConfigurableThreshold |
| FR-M1.1-028 | Rate limiter MUST have configurable cooldown duration (default: 60 seconds) | ✅ Implemented | internal/security/rate_limiter.go (cooldown parameter) | - | tests/security/TestRateLimiter_Cooldown |
| FR-M1.1-029 | Rate limiter MUST drop packets from flooding sources during cooldown without parsing | ✅ Implemented | querier/querier.go (rate limit check before ParseMessage) | - | tests/querier/TestRateLimitDropsDuringCooldown |
| FR-M1.1-030 | Rate limiter MUST log first violation at warn level, subsequent at debug level | ✅ Implemented | internal/security/rate_limiter.go (log level logic) | - | tests/security/TestRateLimiter_LogLevels |
| FR-M1.1-031 | Rate limiter MUST periodically clean up expired cooldown entries (every 5 minutes) | ✅ Implemented | internal/security/rate_limiter.go (cleanup goroutine) | - | tests/security/TestRateLimiter_Cleanup |
| FR-M1.1-032 | Rate limiter MUST bound cooldown map size (max 10,000 entries, LRU eviction) | ✅ Implemented | internal/security/rate_limiter.go (map size limit) | - | tests/security/TestRateLimiter_BoundedSize |
| FR-M1.1-033 | System MUST provide `WithRateLimit(bool)` option to enable/disable rate limiting (default: enabled) | ✅ Implemented | querier/options.go (WithRateLimit option) | - | tests/querier/TestWithRateLimit |
| FR-M1.1-034 | System MUST reject packets larger than 9000 bytes per RFC 6762 §17 | ✅ Implemented | querier/querier.go (size check in receive loop) | RFC 6762 §17 | tests/querier/TestRejectOversizedPackets |
| FR-M1.1-035 | System MUST never panic on malformed packets (already implemented in M1, validated in fuzzing) | ✅ Implemented | internal/message/parser.go (error returns, not panics) | RFC 6762 §18.3 | tests/fuzz/FuzzParseMessage (M1 fuzz tests continue passing) |

---

## Summary

- **Total FRs**: 35 (note: spec.md mentioned 33 initially, but full list shows 35)
- **Status**: All ✅ Implemented (M1.1 complete)
- **Functional Areas**:
  - Socket Configuration (F-9): 10 FRs
  - Interface Management (F-10): 12 FRs
  - Security (F-11): 13 FRs
- **Platform Status**:
  - Linux: ✅ Fully validated (Avahi coexistence tests pass)
  - macOS: ⚠️ Code-complete, untested (per INCOMPLETE_TASKS_ANALYSIS.md)
  - Windows: ⚠️ Code-complete, untested (per INCOMPLETE_TASKS_ANALYSIS.md)

## Notes

- M1.1 focused on production-readiness: socket configuration for Avahi/Bonjour coexistence, interface management for VPN privacy, and security features (rate limiting, source filtering)
- All 35 FRs map to Foundation specs: F-9 (Socket Config), F-10 (Interface Management), F-11 (Security)
- Zero functional regressions - all M1 tests continue passing (SC-008)

---

**Generated**: 2025-11-02
**Next**: Use this data in T033 (aggregate M1.1 FRs into FR matrix)
