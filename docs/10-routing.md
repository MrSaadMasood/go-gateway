# Routing

Systems are specified by behavior first. Implementation follows those behaviors.

Design focuses on:
1. How the system behaves in the common case and under edge conditions  
2. Why a behavior exists  
3. What must hold for that behavior to be valid  
4. Assumptions behind it  
5. Sub-behaviors and the criteria that trigger them  

## What routing means here

Routing redirects a request to a different path than the one received, based on configuration match rules.

## Why it exists

- Smooth backward-compatible API versioning for old and new clients  
- Handle obsolete and deprecated routes without heavy process overhead  
- Support dynamic, config-driven path matching  

## Terminology

**Route** — the request path only (not the host/domain).

## Invariant

All valid requests are routed according to configuration.

## Requirements

1. Evaluate path, headers, query parameters, and related match fields (see config docs).  
2. Ignore inputs that are not part of the routing decision.  
3. On a matching rule, redirect to the configured target path.  
4. When no rule matches, forward to the default target URI after stripping the service segment from the path; the upstream handles the rest.  

## Non-requirements

The router does not proxy. It only evaluates the request and returns the URI the proxy should call.

## Operational notes

Match options are defined in configuration. On any matching option set, the request is redirected to the configured path.

## Router behavior

1. First match wins. Match order is fixed in code, so conflicts are rare.  
2. Routing is deterministic for a given request and config.
