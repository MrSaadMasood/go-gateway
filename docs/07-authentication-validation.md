# Authentication and validation

## Invariant

Only requests with an allowed IP, allowed headers, an optional bearer token when configured, a non-obsolete URL, and a supported service version pass validation.

## Requirements

1. Block IPs at global, service, and route level  
2. Block origins at global and service level  
3. Enforce a maximum request body size  
4. Control required, restricted, and allowed headers  
5. Optionally require a bearer token on protected routes  
6. Surface deprecated / obsolete URLs and headers for a service  
7. Reject requests targeting unsupported service versions  

## Non-requirements

1. The gateway does not inspect bearer token contents — only presence on protected routes. Token semantics belong to each service.  
2. Service-specific auth that needs domain models stays out of the gateway.  
3. Origins are not blocked at route level.  

Failed validation returns an error response. Checks run after the request is mapped to a service.

## Operational notes

1. Global allowed origins are enforced via the global CORS policy.  
2. Service-level origins (and allowed headers) are checked during service validation.  
3. Missing required headers → error.  
4. Restricted headers present → error.  
5. Deprecated headers/URLs add `Deprecation` / `Warning` response headers and still forward.  
6. Bearer presence is checked when enabled; configured skip paths bypass the check.  
7. Obsolete URLs fail validation.  
8. Unknown service versions fail validation.  
9. Body size is checked before proxying; oversize bodies are rejected against the global limit.
