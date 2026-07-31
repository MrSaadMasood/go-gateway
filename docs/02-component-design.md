# Component design

Components are split by responsibility, with dependencies pointed at abstractions (interfaces), not concrete implementations.

Each component owns a single concern. Contracts are defined as interfaces and data models before implementation. Contracts exclude implementation details — including anything that would change observable behavior if swapped. Coupling stays minimal: no component performs another component's work.
