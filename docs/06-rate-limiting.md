the system would rate limit the requests based on the ip address of the user.
The rate limit is applied at 3 different levels:
1- the global level when the request enters the system,
2- the service level when the request enters the system,
3- the url path level for that service

Only the requests that are processes and not rate limited would be considered
All requests that would be rate limited would fail with status code of 429.

The rate limit strategy used are token buckets
The token refil rate and tokens in a bucket can be configured through the config
