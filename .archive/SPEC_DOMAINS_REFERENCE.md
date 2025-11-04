# Specification Domains Quick Reference

## RFC 6762 (Multicast DNS) - 10 Domains

| # | Domain Name | RFC Sections | Tier | Can Parallelize? |
|---|-------------|--------------|------|------------------|
| 1 | Message Format & Protocol Structure | 2, 17, 18, 19 | 1 | ✅ Yes - Foundation |
| 2 | Name Management & Namespace | 3, 4, 12, 16, 22 | 1 | ✅ Yes - After Domain 1 |
| 3 | Query Operations | 5, 7.1-7.3 | 2 | ✅ Yes - Coordinate with 4 |
| 4 | Response Operations | 6, 7.4 | 2 | ✅ Yes - Coordinate with 3 |
| 5 | Cache Management & Coherency | 10, 11 | 2 | ✅ Yes - Independent |
| 6 | Probing & Conflict Resolution | 8.1-8.2, 9 | 3 | ❌ No - Needs 3 & 4 first |
| 7 | Announcing & Updates | 8.3-8.4 | 3 | ❌ No - Needs 6 first |
| 8 | Traffic Optimization | 7, 6.4 | 3 | ⚠️ Partial - After 3, 4, 5 |
| 9 | Multi-Interface & Multi-Responder | 14, 15 | 4 | ✅ Yes - Specialization |
| 10 | Security Considerations | 13, 21, 22.1 | 4 | ⚠️ Review - After all others |

## RFC 6763 (DNS-SD) - 16 Domains

| # | Domain Name | RFC Sections | Tier | Can Parallelize? |
|---|-------------|--------------|------|------------------|
| 11 | Service Instance Naming | 4.1, 4.3, App E | 1 | ✅ Yes - Foundation |
| 12 | Service Type Specification | 4.1.2, 7, 7.2 | 1 | ✅ Yes - Foundation |
| 13 | Domain Enumeration | 11 | 1 | ✅ Yes - Infrastructure |
| 14 | Rationale & Architecture | 1, 3, App A, B | 1 | ✅ Yes - Documentation |
| 15 | Service Instance Enumeration | 3, 4, 4.2, App F | 2 | ✅ Yes - After 11, 12 |
| 16 | Service Instance Resolution | 5 | 2 | ✅ Yes - After 11 |
| 17 | TXT Record Format | 6, 6.1-6.8 | 2 | ✅ Yes - After 11 |
| 18 | Service Type Enumeration | 9 | 2 | ✅ Yes - After 12 |
| 19 | Flagship Naming | 8 | 2 | ✅ Yes - After 11, 12 |
| 20 | User Experience Guidelines | 4.2, App C, D | 2 | ✅ Yes - After 11, 15 |
| 21 | Selective Instance Enum (Subtypes) | 7.1 | 3 | ⚠️ Partial - After 12, 15 |
| 22 | Additional Record Generation | 12, 12.1-12.4 | 3 | ⚠️ Partial - After 15, 16, 17 |
| 23 | DNS Record Population Methods | 10 | 4 | ✅ Yes - Operations guide |
| 24 | Security Considerations | 15 | 4 | ⚠️ Review - After all others |
| 25 | IPv6 Considerations | 14 | 4 | ✅ Yes - After 16, 22 |
| 26 | IANA Considerations | 16 | 4 | ✅ Yes - After 12, 21, 19 |

## Summary by Tier

### Tier 1: Foundation (6 domains - ALL PARALLEL)
- mDNS: Domains 1, 2
- DNS-SD: Domains 11, 12, 13, 14

### Tier 2: Core Operations (9 domains - MOSTLY PARALLEL)
- mDNS: Domains 3, 4, 5
- DNS-SD: Domains 15, 16, 17, 18, 19, 20

### Tier 3: Advanced Mechanisms (5 domains - PARTIAL SERIAL)
- mDNS: Domains 6, 7, 8
- DNS-SD: Domains 21, 22

### Tier 4: Cross-Cutting (6 domains - MOSTLY PARALLEL)
- mDNS: Domains 9, 10
- DNS-SD: Domains 23, 24, 25, 26

## Total Parallelization Potential

- **Maximum concurrent agents**: 9 (in Tier 2)
- **Minimum serial bottlenecks**: 2 (Domains 6→7 in mDNS)
- **Total specification domains**: 26
- **Estimated time with full parallelization**: 4 phases vs 26 sequential phases

## Key Dependencies to Watch

1. **Domain 1** → Everything else in mDNS
2. **Domain 11** → Everything else in DNS-SD
3. **Domains 3 & 4** → Domain 6 (Probing)
4. **Domain 6** → Domain 7 (Announcing)
5. **Domains 15, 16, 17** → Domain 22 (Additional Records)

## Legend

- ✅ Yes - Fully parallelizable within tier
- ⚠️ Partial - Some constraints, coordinate with other specs
- ❌ No - Must be sequential, has strict dependencies
