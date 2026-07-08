Requirements:

1- The request metadata including the req path, params, body, headers, response body, response status. In case of failures it should also record the failure reason / error and the step at which failure occured should be persisted.
2- The metadata should survice persistence across server restarts.
3- Any unexpected errors that might occur that can result in process crashing or req failures should be persisted. if not possible to persist should be logged to the stdout for visiblity.
4- Any gateway internal state that is not directly involved with req response lifecycle and debugging would not be persisted.
5- We should be able to read the already persisted logs.
6- In case of log write failure, the gateway should log the error on the system console.
7- In case of log read failure, it should send a proper error response
8- the audit log should be done at the following places: When the request enter the system, When the response for the request is sent

Non Requirements:
1- The gateway would not provide any fancy ui or dashboard to read and analyze the logs.
2- The gateway should not provide any kind of metric system for analyzing the logs
3- The gateway would not provide any guarantees when the system is forcefully spotted or brought down bypassing any gracefull shutdowns.

Operational Docs:
1- The audit logger does not block the request cycle for any kind of logging.
2- The logs are collected for a period of time and flushed to the persistence store
3- The collection of log data is split across the entire request lifecycle. This distributed data collection is done for maximum flexibility.

Invariant:
1- The persistence store should be up and running for auditor to work properly
