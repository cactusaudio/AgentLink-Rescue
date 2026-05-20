# AgentLink Rescue v0.5 Runtime Spec seeded by v0.2A/v0.2B

## Runtime pipeline

```text
collector -> redactor -> feature extractor -> symptom route -> card retrieval -> discriminator plan -> verifier -> recipe proposal -> human-confirmed execution -> verifier -> rollback/report
```

## Planner rule

The local model does **not** invent fixes. It selects cards, explains discriminators, and ranks the next safest observation or recipe.

## Scoring

```text
score = symptom_match
      + observation_match
      + discriminator_match * 2
      + domain_route_prior
      - risk_penalty
      - lower_layer_uncertainty_penalty
```

## Mutating actions

- `read_only`: can run automatically.
- `human_confirmed`: can show copyable command only after snapshot.
- `manual_only`: no execution; checklist/report only.
- `never_auto`: blocked by runtime.

## Closure states

- `fixed_verified`
- `fixed_pending_retest`
- `manual_only_escalation`
- `external_provider_likely`
- `policy_blocked`
- `insufficient_observation`
