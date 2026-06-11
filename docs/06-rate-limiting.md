When designing low level system like in this case rate limiter. We should follow the same approach that we use to design the systems at high level.
Meaning we should start from defining the requirements, non requirements, any guarantees the system would need to provide.
Identify the key components it would have, define the boundaries of such system. Idenfity what each component would do, what are the invariants of the sub system.
how this system would manage the consistency, concurrency, resources, conflicts, time, lifecycle, cleanup, operational guarantees etc.

System guarantees exist in the architect docs, api contracts, interfaces, operational documentation.
Code only enforces those guarantees.

the system would rate limit the requests based on the ip address of the user.
The rate limit is applied at 3 different levels:
1- the global level when the request enters the system,
2- the service level when the request enters the system,
3- the url path level for that service

All requests that would be rate limited would fail with status code of 429.

The rate limit strategy used are token buckets
The token refil rate and tokens in a bucket can be configured through the config
Token buckets exist at above mentioned levels.
A token is consumed at every level moving from global to route level. Any consumed token wont be reverted back, in case the of failure at lower levels
