After SRS, you should start designing the components.
While designing the components, you should keep in mind the Single Responsibility Principle and the Dependency Inversion Priciple
Once the responsibilities of each components are decided, you should create api contracts.
Contracts are ususally created through interfaces and the data models
The contracts should not include any implementation details ( or somethings that might change the behaviour of the system - that too is implementation detail) in them either explicitly or implicitly
Try to keep the coupling at the minimum
No component should do the work of the other
