#!/usr/bin/env python3
import json, sys, yaml
from collections import Counter
from pathlib import Path
root = Path(__file__).resolve().parents[1]
doc = yaml.safe_load((root/'cards/failure_cards.yaml').read_text())
cards = doc['cards']
ids = [c['id'] for c in cards]
assert len(ids) == len(set(ids)), 'duplicate ids'
required = yaml.safe_load((root/'schema/failure_card.schema.yaml').read_text())['required_fields']
missing=[]
for c in cards:
    for f in required:
        if f not in c: missing.append((c['id'], f))
    assert c.get('verify'), f"{c['id']} has no verifier"
    assert c.get('rollback'), f"{c['id']} has no rollback"
    if c['layer'] in ['L10_multicast_discovery','L11_ptp_clock','L12_aoip_media']:
        assert c['risk'] == 'never_auto', f"{c['id']} AoIP card must be never_auto"
assert not missing, missing[:10]
print(json.dumps({'cards':len(cards),'layers':Counter(c['layer'] for c in cards),'risks':Counter(c['risk'] for c in cards)}, indent=2, default=dict))
