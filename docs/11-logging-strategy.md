What is logging?
Logging in terms of gateway is recoding particular events for various purposes like debugging, insights, observability etc.

How logging is handled in gateway?
Logging is primiarily handled through 2 methods in gateway:
1- Directly logging to the stdout for most critical scenerios e.g server errors, server close message and in case of third party connection loss midway, while the server is running and processing request, the normal request logs are also logged to the stdout
2- Normal request logs are stored directly to the connected database

What is logged in gateway?
1- Gateway creates one log/request, storing the request status, where it failed, how far did it reach in the gateway, failure reason, request and response metadata.
2- Server message that effect the working of gateway including gateway start, server close and unexpected server errors

Requirements:
1- The logging should log what happened to each request, either it was successfull, failed including the failure reason and where exactly the failure happened in the request response cycle within the gateway
2- The logging should log any unexpected errors that can happen while the gateway and server is running.
3- Request logs should be persisted.
4- The complete request path should be logged for proper tracking and observation.

Non Requirements:
1- The logging would not try to force persistence if the third party persistence layer becomes unavaialbe midway. It would only log directly to the stdout.

Operational Docs:
1- The normal request logs are flushed directly into the connected database at regular intervals.
2- Structured logging is used for logging to stdout.
3- If the database connection fails or is lost or some errors occurs while storing the logs in the database, we dump them on the stdout for visibility and debuggability
4- All gateway start and server errors are logged directly to the stdout
5- If the server is gracefully shutdown, any request logs waiting to be persisted, would be flushed to the stdout.