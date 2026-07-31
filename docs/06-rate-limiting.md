# Rate limiting

The rate limiter is designed with the same discipline as the rest of the gateway: requirements and non-requirements first, then components, boundaries, and invariants — including consistency, concurrency, lifecycle, cleanup, and operational guarantees.

Those guarantees live in architecture notes, API contracts, and operational docs. Code enforces them; it does not invent them.

## Behavior

Requests are rate-limited by client IP at three levels:

1. Global — when the request enters the gateway  
2. Service — for the mapped backend  
3. Route — for the service path  

A limited request fails with HTTP **429**.

## Strategy

Token buckets are used at each level above. Capacity and refill rate are config-driven. A token is consumed at every level from global down to route. Consumed tokens are not returned if a later level fails.
