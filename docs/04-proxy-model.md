You should always reason about system integrity.
You should design system, so its nearly impossible to break them or leave them in an inconsistent / invalid state.
You should argue about what in the system prevents it from breaking.
You should recognize when order is an implementation detail and when order is a behaviour

if the system has to do some processing with multiple steps and states, the states should determine the flow and execution steps, preferrably through a state machine.
Make execution structure explicit, enforceable, and hard to misuse
