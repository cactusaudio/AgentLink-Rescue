# Volume 00 — Method: Encyclopedia as Build-Time Search

The point of the encyclopedia phase is to over-expand the possible failure universe before compressing it into a small rescue UI. It should contain the reasoning that the end user never sees.

## The core abstraction

A network failure is not a single event. It is a misalignment between:

- desired reachability,
- observed local state,
- active control plane,
- hidden policy layer,
- transport path,
- application expectation,
- and repair risk.

For AgentLink Rescue, every issue must eventually be represented as:

```text
symptom signature
  + local snapshot
  + discriminating tests
  -> probable failure node
  -> bounded repair action
  -> verifier
  -> rollback
```

## Failure diagnosis is graph traversal, not Q&A

Bad pattern:

```text
User: internet broken.
AI: try flushing DNS.
```

Good pattern:

```text
1. Is link up?
2. Does the interface have a non-link-local IP?
3. Is there a default route?
4. Can we reach gateway?
5. Can we reach raw IP outside LAN?
6. Can we resolve DNS?
7. Does curl work without proxy?
8. Does curl work with active proxy?
9. Does target app inherit the same proxy/DNS/TLS state?
10. If pro-audio, is media clock and multicast intact?
```

## Build-time explosion, runtime compression

The encyclopedia should expand aggressively:

- multiple symptoms for one cause,
- multiple causes for one symptom,
- variants by app,
- variants by privilege level,
- variants by VPN/TUN state,
- variants by audio-network topology.

The runtime must compress aggressively:

- show only the top 1–3 likely causes,
- run only safe checks automatically,
- separate "fix now" from "manual inspection",
- hide low-probability branches until needed.

## The wrong objective

"Write every possible network issue and never stop" is useful as a creative prompt but not as an engineering objective. It has no stop condition and produces duplicate prose.

## The right objective

Generate a failure genome until the following are true:

1. Every layer has cards.
2. Every card has discriminators.
3. Every repair has verifier.
4. Every mutating repair has rollback.
5. Every high-risk area is manual-first.
6. Every card can be indexed by symptom, layer, observation, repair, and risk.
