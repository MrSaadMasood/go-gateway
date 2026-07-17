One should think about systems in terms of their behaviour. Implementation is secondary and should reflect those behaviours.

Think in terms of:
1- How should the system behave generally and under certain special conditions.
2- Why a specific behaviour exists
3- What things must hold true for the system to exhibit the specific defined behaviour.
4- What assumptions are we making regarding that behaviour of the system
5- What different sub behaviours are exhibitited.
6- What are the criteria to exhibit those sub behaviours.

What is routing in context of the gateway?
Routing is redirecting a request to a target that is different from its original target based on the various configuration options and requirements.

Why routing exists?
Routing is done to support backward compatible api versioning in a smooth manner and to suppport the old and new clients.
To handle obsolete and deprecated routes, without much overhead.
To have dynamic route matching.

Terminology:
Route: Route refers to the path of the request. It does not include request url domain.

Invariants:
1- All valid requests would be routed based on the configuration.

Requirements:
1- The request should be evaluated based on various things including the path, headers, query parameters etc. For complete details you can read the configuration documentation.
2- Everything that is not involved in determining where the request should be routed based on the configuration options is ignored. Including headers, query, path etc
3- Once a request passes the routing criteria, is evaluated, its routing to the target route defined in the config for that route criteria.
4- When no rules apply, we simply forward the request to the default target uri after ommiting the service uri from the path. it would be left to the service to handle it.

Non Requirements:
1- Router should not be reponsible for proxying the request. It would only evalute the request based on the config and various parameters and provide find uri the request should be proxied to.
