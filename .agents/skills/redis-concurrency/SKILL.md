---
name: redis-concurrency
description: >-
  Recipes and guidelines for implementing Redis high-traffic mechanisms in QuickTix:
  Sorted Set Virtual Waiting Room (FR4), Distributed Rate Limiter (FR5), and Distributed Locks/TTL Ticket Holds (FR6).
---

# Redis Concurrency & High-Traffic Architecture Guide

Use this skill when implementing virtual queue admission control, token-bucket rate limiters, or distributed reservation locks.

## 1. Virtual Waiting Room (FR4)
- **Data Structure**: Redis Sorted Set (`ZADD queue:<event_id> <timestamp> <user_id>`).
- **Position & Wait Time Calculation**:
  - `ZRANK queue:<event_id> <user_id>` returns user's 0-based queue index.
  - Estimated Wait Time = `(position / rate_per_minute) * 60 seconds`.
- **Session Admission TTL**:
  - Admitted users get a active booking session key in Redis: `session:<event_id>:<user_id>` with 3-minute TTL (`SET key val EX 180`).

## 2. Distributed Rate Limiter (FR5)
- **Data Structure**: Redis Token Bucket implemented via atomic Lua Script.
- **Lua Logic**:
  - Key: `ratelimit:<user_id_or_ip>:<endpoint>`
  - Evaluates remaining tokens, capacity limit, and fill rate.
- **Headers Returned**:
  - `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.
  - HTTP `429 Too Many Requests` + `Retry-After: <seconds>` on limit breach.

## 3. Ticket Reservation & Distributed Locking (FR6)
- **Overselling Guarantee**:
  - Distributed Lock: `SET lock:ticket_type:<ticket_type_id> <uuid> NX PX 5000` (5-second lock duration).
  - Quantity Hold Key: `hold:<user_id>:<ticket_type_id>` set with 10-minute TTL.
- **Auto-Release on Expiry**:
  - Uses Redis Keyspace Notifications (`__keyevent@0__:expired`) or active polling worker to return held tickets back to available inventory pool upon 10-minute expiry.
- **Idempotency Key**:
  - Check idempotency key in Redis (`idempotency:<key>`) before creating reservation or charge to guarantee duplicate requests return cached results safely.
