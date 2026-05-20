# v0.2B Topology Notes

v0.2B turns the card corpus into a diagnosis graph.

## Topology rules

1. Diagnose lower layers before higher layers.
2. Treat proxy/VPN/TUN as control planes that can mask DNS, route, and app failures.
3. Treat browser-vs-CLI divergence as a high-signal symptom for proxy/env/TLS/tool config.
4. Treat AoIP as three separate planes: discovery, clock, media.
5. Treat external-provider outage only after local baseline passes.
6. Treat MDM/policy as a terminal report state, not a hack-around state.

## Runtime compression

At runtime, retrieve a small number of cards per stage; never dump the encyclopedia to the model. The local model should see:

- user symptom;
- redacted snapshot features;
- top candidate cards;
- topology constraints;
- allowed next actions.

The runtime, not the model, enforces safety gates.
