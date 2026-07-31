# Startup and rehydration

Startup behavior is part of the system contract: what must be true for the process to start, what happens when it is not, and which side effects run before traffic is accepted. Behavior is documented here and covered by tests where practical.

## Rehydration

Rehydration loads any state required for correct operation before serving requests. Today that includes configuration and a connection to the database used for request logs and traffic data.

## Startup sequence

1. Register termination signals (`SIGINT`, `SIGTERM`).  
2. Load config — failure panics; the process does not start.  
3. Construct components that depend on config.  
4. Connect to the persistence layer — failure panics.  
5. Register HTTP handlers and start the server.  

## Startup invariants

### Rehydration

1. In-memory state that was never persisted is lost on restart (e.g. per-server / per-route traffic counters). It is not restored.  
2. Every restart reloads config and injects it into the system; config changes take effect on the next start.  

### Configuration

1. Config is loaded from `config.json` at the project root.  
2. Invalid config fails startup immediately; no requests are served. The error is logged to stdout.  
3. Without a config file, the gateway does not start.  

### Failure after start

1. If the persistence connection fails or drops after startup, the gateway keeps serving requests. Request metadata is not stored; errors go to stdout. Log reads also fail in that case.  
2. If configuration cannot be loaded for any reason, the gateway does not start and serves no traffic.
