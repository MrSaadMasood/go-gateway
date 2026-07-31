# Persistence design

## Requirements

1. Persist request metadata: path, params, body, headers, response body/status; on failure, also reason and the pipeline step where it failed.  
2. Survive process restarts.  
3. Persist unexpected errors that crash or fail requests; if persistence is impossible, write to stdout for visibility.  
4. Do not persist gateway-internal state unrelated to the request/response lifecycle or debugging.  
5. Support reading persisted logs.  
6. On write failure, log the error to the console.  
7. On read failure, return a proper error response.  
8. Audit at request entry and when the response is sent.  

## Non-requirements

1. No UI or dashboard for log analysis.  
2. No metrics system built on top of these logs.  
3. No durability guarantees if the process is force-killed, bypassing graceful shutdown.  

## Operational notes

1. Audit logging does not block the request path.  
2. Logs are buffered and flushed to the store periodically.  
3. Log fields are collected across the lifecycle so each stage can contribute without a single choke point.  

## Invariant

The persistence store must be available at startup for the auditor to initialize correctly.
