The week meant to teach about the start up behaviour of the application and also in terms of system contracts. What guarantees should exist for the application to start successfully. What happens if those guarnatees arent there. Are there are side effects that would be performed by the application. In short you should try to document and prove the behaviour of your system through tests.

Rehydration:
Rehydration in the context of this application means, any state that's needed for the correct functioning of the application should be loaded in the application on startup.
This for now includes loading the config
An extention of this also includes connecting to the database, for storing the request and service traffic data and logs.

Startup Sequence:
1- the gateway registers signals that indicate the process is going to be terminated or killed, like the sigint and sigterm signals
2- the config is loaded next. the failure of which resutls in panic
3- all the components of the system that are dependent on the config are created
4- connection to the perisitence layer is made. the failure of while results in panic
5- the system then registers the endpoints and starts the server finally.

Startup Invariants:

i- Rehydration Guarantees:
1- in case of system restart anything the system has in momory that is not persisted, would be lost e.g it includes the req traffic per server and per servcie route. The information wont be restored.
2- every system restart causing the config to be loaded and injected into the system. Any changes in the config would be reflected in the gateway.

ii-Configuration:
1- the configuration is loaded from the config.json file on root of the project.
2- Any errors in the configuration file would result in gateway to throw an error and shutdown immediately. The gateway would not serve any requests in case of misconfigured configuration file. The error would be logged to the std out
3- Without a config file the gateway would not start.

iii- Failure Guarantees:
1- In case of connection failure or connection loss at later stages; other than startup; with the persistence store, the gateway would continue on working and keep on serving requests, but the requests metadata wont be stored anywhere and the error messages would be logged to the std out. Reading the logs wont also work on such cases
2- If loading the confiugration fails, due to any reason, the gateway would not start and consequently no requests would be served.
