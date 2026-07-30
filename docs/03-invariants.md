Invariants are the truths about the system that must hold true for the system to work.
We should identify the invairiants of the sytem, both eplicit and implicit when designing the systems.
We design systems where correctness is enforced by rules, not by happy path sequence.
while designing state machine, we need to make sure if someone were to intentionally misuse the system, would they ever cause the system to be in an invalid state.
The design should also define what is impossible rather than what is allowed.
There is be no ghost state (transitions that lead to no where)
