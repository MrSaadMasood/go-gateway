# Timeouts and cancellation

Execution boundaries define how long work may run and who may cancel it.

Design questions answered here:
- Where execution boundaries sit in the gateway
- What kinds of boundaries exist (request context, proxy timeout, shutdown)
- Who controls them (config, per-service overrides, process signals)
- Which parts of the system are bound by them, and which are not
- Whether boundaries are global, nested, or independent per subsystem

The implementation aligns with those boundaries: in-flight work respects context cancellation and configured timeouts rather than running unbounded.
