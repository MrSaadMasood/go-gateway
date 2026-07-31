# Proxy model

System integrity is a primary design constraint. The gateway is structured so invalid or inconsistent states are difficult to reach, and each guard that prevents breakage is intentional and reviewable.

Order is treated carefully: when sequence is behavior (not an implementation detail), it is made explicit. Multi-step request handling is driven by states — preferably a state machine — so execution structure is enforceable and hard to misuse.
