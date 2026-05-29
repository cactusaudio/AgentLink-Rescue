// Package genomekernel bridges the distilled network genome (the 15-layer
// failure-card ontology) to the execution kernel's classify taxonomy, making
// the genome the single source of truth for the failure space without
// duplicating 300 cards into classify.
//
// The bridge is one explicit, documented map (ClassLayer) anchored on the
// genome's OWN layer ontology — the shared identifier the architecture review
// (P0.1) found missing. A consistency test asserts every kernel class is
// covered by genome cards, so the two taxonomies cannot silently drift.
package genomekernel

import (
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/genome"
)

// ClassLayer maps each deterministic classify class to the genome layer that
// owns its knowledge. OK is intentionally absent (it is the healthy state, not
// a failure with cards).
var ClassLayer = map[string]string{
	classify.NoActiveInterface:                 "L01_interface_service",
	classify.LinkLocalOnly:                     "L02_ip_dhcp",
	classify.NoDHCPLease:                       "L02_ip_dhcp",
	classify.NoDefaultRoute:                    "L03_routing",
	classify.GatewayUnreachable:                "L03_routing",
	classify.RawIPUnreachable:                  "L03_routing",
	classify.DNSFail:                           "L04_dns",
	classify.HTTPSFail:                         "L09_tls_identity",
	classify.SystemProxyDirty:                  "L05_proxy",
	classify.UserProxyDirty:                    "L05_proxy",
	classify.GitProxyDirty:                     "L05_proxy",
	classify.NpmProxyDirty:                     "L05_proxy",
	classify.BrewProxyDirty:                    "L05_proxy",
	classify.KnownAgentResidue:                 "L06_vpn_ne_tun",
	classify.NetworkExtensionSuspected:         "L06_vpn_ne_tun",
	classify.ClashTunActiveOrStale:             "L06_vpn_ne_tun",
	classify.NetworkExtensionSessionStale:      "L06_vpn_ne_tun",
	classify.TunRouteOwnershipSuspected:        "L06_vpn_ne_tun",
	classify.AirDropDiscoveryDegraded:          "L10_multicast_discovery",
	classify.NetworkLocationSuspected:          "L14_user_policy",
	classify.SysconfigSuspected:                "L14_user_policy",
	classify.MDMProfileSuspected:               "L14_user_policy",
	classify.GeneralInternetOKAgentEndpointBlocked: "L13_external_provider",
	classify.ProtectedTopologyConstraint:       "L12_aoip_media",
	classify.ProtectedAudioVLANRouteTrap:       "L12_aoip_media",
}

// LayerForClass returns the genome layer that owns a class, and whether the
// class has a mapping (OK and unmapped classes return false).
func LayerForClass(class string) (string, bool) {
	layer, ok := ClassLayer[class]
	return layer, ok
}

// CoverageForClass returns the genome cards whose layer owns the given class —
// the knowledge the kernel can surface for that failure. Returns nil for OK or
// an unmapped class.
func CoverageForClass(corpus genome.Corpus, class string) []genome.Card {
	layer, ok := ClassLayer[class]
	if !ok {
		return nil
	}
	var out []genome.Card
	for _, c := range corpus.Cards {
		if c.Layer == layer {
			out = append(out, c)
		}
	}
	return out
}

// Advisory is the compact, read-only genome context the decision loop can
// attach to a report. It never influences the decision — it is operator-facing
// knowledge sourced from the distilled cards.
type Advisory struct {
	Class      string   `json:"class"`
	Layer      string   `json:"layer"`
	CardIDs    []string `json:"cardIds"`
	SafeChecks []string `json:"safeChecks,omitempty"`
}

// AdvisoryForClass projects up to maxCards genome cards for a class into a
// compact advisory (card IDs + a few read-only safe checks). Returns ok=false
// when the class has no genome coverage.
func AdvisoryForClass(corpus genome.Corpus, class string, maxCards int) (Advisory, bool) {
	layer, ok := ClassLayer[class]
	if !ok {
		return Advisory{}, false
	}
	cards := CoverageForClass(corpus, class)
	if len(cards) == 0 {
		return Advisory{}, false
	}
	adv := Advisory{Class: class, Layer: layer}
	seenCheck := map[string]bool{}
	for i, c := range cards {
		if maxCards > 0 && i >= maxCards {
			break
		}
		adv.CardIDs = append(adv.CardIDs, c.ID)
		for _, chk := range c.SafeChecks {
			if chk == "" || seenCheck[chk] {
				continue
			}
			seenCheck[chk] = true
			adv.SafeChecks = append(adv.SafeChecks, chk)
		}
	}
	return adv, true
}
