# Logging strategy

## What logging means here

Logging records gateway events for debugging, operational insight, and observability.

## How it is handled

Two channels:

1. **Stdout** — critical paths: server errors, shutdown, persistence connection loss mid-flight, and structured fallback when request logs cannot be stored.  
2. **Database** — normal per-request audit logs when persistence is available.  

## What is logged

1. One log per request: status trail through the pipeline, failure step and reason when applicable, plus request/response metadata.  
2. Process-level events that affect the gateway: start, close, unexpected server errors.  

## Requirements

1. Record success or failure per request, including failure reason and where in the pipeline it stopped.  
2. Record unexpected errors while the gateway is running.  
3. Persist request logs when the store is available.  
4. Include the full request path for tracking.  

## Non-requirements

If the persistence layer becomes unavailable mid-run, the gateway does not block or force writes — it falls back to stdout.

## Operational notes

1. Request logs are flushed to the database on an interval.  
2. Stdout uses structured logging.  
3. On DB write/connection failure, pending request logs are dumped to stdout.  
4. Gateway start and server errors always go to stdout.  
5. On graceful shutdown, any logs still buffered for persistence are flushed to stdout.
