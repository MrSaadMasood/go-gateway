# Invariants

Invariants are properties that must always hold for the system to be correct. Both explicit and implicit invariants are identified at design time.

Correctness is enforced by rules, not by hoping callers follow the happy path. The request state machine rejects illegal transitions: intentional misuse must not produce an invalid state. The design defines what is impossible, not only what is allowed. There are no ghost states — every transition has a defined destination.
