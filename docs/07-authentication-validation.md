Invariant:
Only the request with allowed ip, allowed headers, with optionally configured bearer token, non-obsolete url sent at one of the available versions of the servcie passes authentication and validation

Requirements:
1- The gateway would block ips at the global, service and route level
1- The gateway would block origins at the global and service level
2- The gateway limit the size of the request body
3- The gateway would be able to control which headers are required, restricted and which are allowed to pass
4- The gateway can also check the protected routes for the existence of bearer token
5- The gateway should also manage depricated, obsolete urls and headers for the service
6- the gatway should verify if the request is being transferred to the correct available version of the service. Request to incorrect versions will not be entertained.

Non Requirements:
1- The gateway would not check what's inside the bearer token. It will only check its existence for protected routes. The token decoding is not a responsiblity of the gateway and each service would hanle it according to its own business logic.
2- The gateway should not manage any auth requirements or procedures that are service specific and would need the knowledge of the service internal data models and domain knowledge.
3- The gateway would not block origins at route level

Any req that fails the authentication, would return an error response.
The request authentication happens as soon as the request enters the gateway and is mapped to a service

Operational Docs:
1- For globally allowed origins, it would be managed by the cors policy at the global level
2- For service allowed origins, it would be mangaged through the cors policy at the servcie level, while we are authenticating the service. The cors policy at the service level also checks for the allowed headers
3- if any required headers are not provided, it returns an error repsonse
4- if any restricted header is provided, it return an error repsonse
5- for any deprecation headers and url, it just adds the Deprecation and Warning header values but forward the response
6- it checks for the existence of bearer tokens if the option to check it is enabled, and it req path is in skipable paths for token check its skipped and the req is allowed to be processed
7- for obsolete urls the authentication fails
8- if the request service version is not present in the available service versions, the authentication fails
9- the req body size is calcuated before proxying the request. if exceeded than global req size it throws an error
