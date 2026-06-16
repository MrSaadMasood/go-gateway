Requirements:
1- The gateway would block ips and origins at the global, service and route level.
2- The gateway limit the size of the request body
3- The gateway would be able to control which headers are required, restricted and which are allowed to pass
4- The gateway can also check the protected routes for the existence of bearer token
5- The gateway should also manage depricated, obsolete urls and headers for the service
6- the gatway should verify if the request is being transferred to the correct available version of the service. Request to incorrect versions will not be entertained.

Non Requirements:
1- The gateway would not check what's inside the bearer token. It will only check its existence for protected routes. The token decoding is not a responsiblity of the gateway and each service would hanle it according to its own business logic.
2- The gateway should not manage any auth requirements or procedures that are service specific and would need the knowledge of the service internal data models and domain knowledge.

Any req that fails the authentication, would return an error response.
The request authentication happens as soon as the request enters the gateway and is mapped to a service
