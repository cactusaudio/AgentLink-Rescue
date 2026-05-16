// Package diagnosisgraph projects a heavy DiagnosticReport into a
// COMPACT, redacted, structured graph (V0300 Layer 3).
//
// Design constraints:
//   - small enough for a Gemma 4-class small-memory model (no raw dumps,
//     booleans + short tokens + a few tool ids);
//   - precise enough for Qwen/OpenCode (failure class, confidence,
//     recommended next tool ids, forbidden next tools, reasons);
//   - pure + deterministic (test-stable from fixtures, no host access);
//   - recommended/forbidden tool ids are validated against the W2 typed
//     manifest (single source of truth — no drift).
package diagnosisgraph

import (
	"encoding/json"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/toolmanifest"
)

type FactBullet struct {
	Key    string `json:"key"`
	Value  string `json:"value"` // short, redacted
	Signal string `json:"signal,omitempty"`
}

type Check struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
}

type ToolRef struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type Graph struct {
	SchemaVersion        int          `json:"schemaVersion"`
	Note                 string       `json:"note"`
	PrimaryClass         string       `json:"primaryClass"`
	FailureClasses       []string     `json:"failureClasses"`
	Confidence           float64      `json:"confidence"`
	Symptoms             []string     `json:"symptoms"`
	Facts                []FactBullet `json:"facts"`
	Checks               []Check      `json:"checks"`
	RecommendedNextTools []ToolRef    `json:"recommendedNextTools"`
	ForbiddenNextTools   []string     `json:"forbiddenNextTools"`
	Reasons              []string     `json:"reasons"`
	Redacted             bool         `json:"redacted"`
}

const note = "Compact diagnosis graph. Decide ONLY from this evidence. " +
	"Execution tools require recipe_dry_run + explicit user approval " +
	"first. Never use raw system commands."

// classRoute maps a deterministic failure class to ordered recommended
// tool ids (must exist in the W2 manifest) + a short reason.
var classRoute = map[string][]ToolRef{
	classify.SystemProxyDirty: {{"agentlink.proxy_snapshot", "confirm dirty proxy scope"},
		{"agentlink.recipe_dry_run", "dry-run proxy-clean-stale-env"},
		{"agentlink.clean_stale_proxy_baseline", "execute after approval"}},
	classify.UserProxyDirty: {{"agentlink.proxy_snapshot", "confirm user proxy residue"},
		{"agentlink.recipe_dry_run", "dry-run proxy-clean-stale-env"},
		{"agentlink.clean_stale_proxy_baseline", "execute after approval"}},
	classify.DNSFail: {{"agentlink.dns_snapshot", "inspect resolvers/scoped DNS"},
		{"agentlink.verify_network", "confirm DNS-layer failure"},
		{"agentlink.recipe_dry_run", "dry-run dns-resolver-baseline"},
		{"agentlink.repair_dns_baseline", "execute after approval"}},
	classify.ClashTunActiveOrStale: {{"agentlink.tun_snapshot", "inspect TUN/route ownership"},
		{"agentlink.recipe_dry_run", "dry-run tun-residue-clean"},
		{"agentlink.approval_ticket_create", "user-approved clash-tun-fix ticket"}},
	classify.TunRouteOwnershipSuspected: {{"agentlink.tun_snapshot", "confirm route ownership"},
		{"agentlink.recipe_dry_run", "dry-run tun-residue-clean"},
		{"agentlink.remove_tun_residue", "execute after approval"}},
	classify.NetworkExtensionSuspected: {{"agentlink.network_extension_snapshot", "inspect NE residue"},
		{"agentlink.recommend_recipes", "rank safe options (higher risk -> ticket)"},
		{"agentlink.approval_ticket_create", "user-approved repair ticket"}},
	classify.NetworkExtensionSessionStale: {{"agentlink.network_extension_snapshot", "inspect stale NE session"},
		{"agentlink.recommend_recipes", "rank safe options"}},
	classify.KnownAgentResidue: {{"agentlink.app_residue_snapshot", "enumerate known-agent residue"},
		{"agentlink.recipe_dry_run", "dry-run app-residue-clean"},
		{"agentlink.clean_app_residue", "execute after approval"}},
	classify.NoActiveInterface: {{"agentlink.doctor", "full read-only diagnosis"},
		{"agentlink.verify_network", "confirm no usable interface"},
		{"agentlink.approval_ticket_create", "user-approved network bring-up ticket"}},
	classify.NoDefaultRoute: {{"agentlink.route_snapshot", "inspect route table"},
		{"agentlink.recommend_recipes", "rank restore options"},
		{"agentlink.restore_last_good", "restore last-good after approval"}},
	classify.GatewayUnreachable: {{"agentlink.route_snapshot", "inspect default route/gateway"},
		{"agentlink.verify_network", "confirm gateway unreachable"}},
	classify.MDMProfileSuspected: {{"agentlink.app_residue_snapshot", "inspect profiles (detect-only)"},
		{"agentlink.incident_report", "report; profile changes need user/admin"}},
	classify.OK: {{"agentlink.verify_network", "confirm healthy; no repair needed"}},
}

// fallbackRoute when class is unknown/empty: stay read-only.
var fallbackRoute = []ToolRef{
	{"agentlink.doctor", "broad read-only diagnosis"},
	{"agentlink.diagnosis_graph", "re-derive after more evidence"},
	{"agentlink.classify_incident", "obtain a deterministic class first"},
}

func rd(redact bool, s string) string {
	if !redact {
		return s
	}
	return safety.RedactSensitive(s)
}

// Build is the pure projection. If redact is true (default in product
// use) all free-text is run through safety.RedactSensitive.
func Build(r diagnose.DiagnosticReport, redact bool) Graph {
	classes := r.Classifications
	if len(classes) == 0 {
		classes = classify.Classify(r)
	}
	primary := classify.OK
	for _, c := range classes {
		if c != classify.OK {
			primary = c
			break
		}
	}
	if len(classes) == 0 {
		primary = ""
	}

	// compact facts (booleans + short tokens only; NO raw dumps)
	net := r.Network
	activeIface := 0
	utun := 0
	for _, i := range net.Interfaces {
		if i.Status == "active" && len(i.IPv4) > 0 && !i.LinkLocalOnly {
			activeIface++
		}
		if i.IsUTun {
			utun++
		}
	}
	resTotal := len(r.Residues.LaunchAgents) + len(r.Residues.LaunchDaemons) +
		len(r.Residues.SystemExtensions) + len(r.Residues.Profiles)
	dnsOK := false
	for _, p := range net.DNSSummary.Resolution {
		if p.OK {
			dnsOK = true
			break
		}
	}
	facts := []FactBullet{
		{"activeInterfaces", itoa(activeIface), boolSig(activeIface > 0)},
		{"defaultRoute", boolStr(net.DefaultRoute.Present), boolSig(net.DefaultRoute.Present)},
		{"dnsResolves", boolStr(dnsOK), boolSig(dnsOK)},
		{"proxyDirty", boolStr(net.ProxySummary.Dirty), boolSig(!net.ProxySummary.Dirty)},
		{"utunInterfaces", itoa(utun), boolSig(utun == 0)},
		{"residueItems", itoa(resTotal), boolSig(resTotal == 0)},
		{"recommendedRepairLevel", rd(redact, r.RecommendedRepairLevel), ""},
	}
	checks := []Check{
		{"hasActiveInterface", activeIface > 0},
		{"hasDefaultRoute", net.DefaultRoute.Present},
		{"dnsResolves", dnsOK},
		{"proxyClean", !net.ProxySummary.Dirty},
		{"noTunResidue", utun == 0},
		{"noKnownResidue", resTotal == 0},
	}
	symptoms := []string{}
	if activeIface == 0 {
		symptoms = append(symptoms, "no active interface with routable IP")
	}
	if !net.DefaultRoute.Present {
		symptoms = append(symptoms, "no default route")
	}
	if !dnsOK && len(net.DNSSummary.Resolution) > 0 {
		symptoms = append(symptoms, "DNS resolution failing")
	}
	if net.ProxySummary.Dirty {
		symptoms = append(symptoms, "proxy configuration dirty")
	}
	if utun > 0 {
		symptoms = append(symptoms, "TUN/utun interface present")
	}
	if resTotal > 0 {
		symptoms = append(symptoms, "known network-agent residue detected")
	}
	for _, w := range r.Warnings {
		symptoms = append(symptoms, rd(redact, w))
	}

	rec := classRoute[primary]
	if len(rec) == 0 {
		rec = fallbackRoute
	}
	// confidence: corroboration heuristic (deterministic, bounded).
	conf := 0.4
	if primary != "" && primary != classify.OK {
		conf = 0.55 + 0.1*float64(min(len(symptoms), 4))
	}
	if primary == classify.OK {
		conf = 0.9
	}
	if conf > 0.95 {
		conf = 0.95
	}

	reasons := []string{}
	if primary == "" {
		reasons = append(reasons, "no deterministic failure class; stay read-only and gather more evidence")
	} else if primary == classify.OK {
		reasons = append(reasons, "network healthy by deterministic checks; recommend no repair")
	} else {
		reasons = append(reasons, "primary class "+primary+" routed to bounded AgentLink recipe path")
		reasons = append(reasons, "execution requires dry-run + user approval; rollback captured by envelope")
	}

	// forbidden = host-mutating execution tool ids from the W2 manifest
	// (forbidden as a FIRST action / before dry-run+approval) + raw
	// surfaces (never). Single source of truth = the manifest.
	forb := []string{}
	for _, t := range toolmanifest.Build("").Tools {
		if t.MutationClass == toolmanifest.MutationHostTxn {
			forb = append(forb, t.ID+" (only after recipe_dry_run + user approval)")
		}
	}
	forb = append(forb, "RAW: "+joinStr(toolmanifest.ForbiddenRawSurfaces)+" (never model-facing)")

	return Graph{
		SchemaVersion: 1, Note: note,
		PrimaryClass: primary, FailureClasses: classes, Confidence: conf,
		Symptoms: symptoms, Facts: facts, Checks: checks,
		RecommendedNextTools: rec, ForbiddenNextTools: forb,
		Reasons: reasons, Redacted: redact,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func boolSig(good bool) string {
	if good {
		return "ok"
	}
	return "problem"
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	d := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		d = append([]byte{'-'}, d...)
	}
	return string(d)
}
func joinStr(a []string) string {
	o := ""
	for i, s := range a {
		if i > 0 {
			o += ", "
		}
		o += s
	}
	return o
}

// JSON renders the compact graph.
func (g Graph) JSON() ([]byte, error) { return json.MarshalIndent(g, "", "  ") }
