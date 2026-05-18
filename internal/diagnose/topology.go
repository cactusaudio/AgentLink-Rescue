package diagnose

import (
	"sort"
	"strings"
)

const (
	TopologyClassProtectedAudioVLAN   = "PROTECTED_AUDIO_VLAN"
	TopologyClassProtectedVideoVLAN   = "PROTECTED_VIDEO_VLAN"
	TopologyClassProtectedCameraLAN   = "PROTECTED_CAMERA_CONTROL_LAN"
	TopologyClassProtectedShowControl = "PROTECTED_SHOW_CONTROL_LAN"
	TopologyClassProtectedStorage     = "PROTECTED_STORAGE_NETWORK"
	TopologyClassProtectedDirectLink  = "PROTECTED_DIRECT_LINK"
	TopologyClassProtectedLabControl  = "PROTECTED_LAB_CONTROL_LAN"
	TopologyClassProtectedPolicy      = "PROTECTED_POLICY_NETWORK"
	TopologyClassProtectedLinkLocal   = "PROTECTED_LINK_LOCAL_SERVICE"
	TopologyClassVirtualBridge        = "VIRTUAL_BRIDGE_RED_HERRING"
	TopologyClassDefaultRouteStale    = "DEFAULT_ROUTE_OWNERSHIP_STALE"
	TopologyClassTunFakeIPPresent     = "TUN_FAKE_IP_PRESENT"
	TopologyClassProtectedRepair      = "PROTECTED_TOPOLOGY_REPAIR_BOUNDARY"
)

var protectedForbiddenActions = []string{
	"dhcp_renew",
	"set_dhcp",
	"set_dns",
	"route_flush",
	"ifconfig_down",
	"proxy_reset",
	"switch_vlan_mutation",
	"tun_cleanup_before_route_ownership_proof",
	"clean_baseline",
}

var protectedAllowedActions = []string{
	"snapshot",
	"scoped_probe",
	"incident_report",
	"support_bundle",
}

// AnalyzeTopology derives deterministic, model-facing topology constraints
// from structured report fields. It never needs Raw to honor an explicit
// topology manifest, and it only uses Raw when evidence is bound to the same
// interface being protected.
func AnalyzeTopology(r *DiagnosticReport) {
	if r == nil {
		return
	}
	if r.Topology.SchemaVersion == 0 {
		r.Topology.SchemaVersion = 1
	}
	mergeTopologyInterfaces(r)
	detectTopologyConstraints(r)
	detectRouteCandidates(r)
	buildRepairCorridor(r)
}

func HasProtectedTopologyConstraint(r DiagnosticReport) bool {
	AnalyzeTopology(&r)
	return len(r.Topology.ProtectedConstraints) > 0
}

func HasProtectedRouteOwnershipConflict(r DiagnosticReport) bool {
	AnalyzeTopology(&r)
	def := r.Network.DefaultRoute.Interface
	if !r.Network.DefaultRoute.Present || def == "" {
		return false
	}
	if r.Reachability.Gateway.Target != "" && r.Reachability.Gateway.OK {
		return false
	}
	if !interfaceProtected(r.Topology, def) {
		return false
	}
	return hasAlternateRoutableInterface(r, def)
}

func ProtectedAudioRouteTrap(r DiagnosticReport) bool {
	AnalyzeTopology(&r)
	def := r.Network.DefaultRoute.Interface
	if !HasProtectedRouteOwnershipConflict(r) {
		return false
	}
	for _, c := range r.Topology.ProtectedConstraints {
		if c.Interface == def && c.Class == TopologyClassProtectedAudioVLAN {
			return true
		}
	}
	return false
}

func RepairCorridorAllowsMutation(r DiagnosticReport) bool {
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) == 0 {
		return true
	}
	return r.Topology.RepairCorridor.MutationAllowed
}

func mergeTopologyInterfaces(r *DiagnosticReport) {
	byName := map[string]int{}
	for i, ti := range r.Topology.Interfaces {
		if ti.Name != "" {
			byName[ti.Name] = i
		}
	}
	portByDevice := map[string]string{}
	for _, hp := range r.Network.HardwarePorts {
		if hp.Device != "" {
			portByDevice[hp.Device] = hp.Port
		}
	}
	for _, ni := range r.Network.Interfaces {
		if ni.Name == "" {
			continue
		}
		ti := TopologyInterface{
			Name:         ni.Name,
			Status:       ni.Status,
			HardwarePort: portByDevice[ni.Name],
			Addresses:    append(append([]string{}, ni.IPv4...), ni.IPv6...),
		}
		if looksPhysicalInternetCandidate(ni) {
			ti.InternetCandidate = true
		}
		if idx, ok := byName[ni.Name]; ok {
			merged := r.Topology.Interfaces[idx]
			if merged.Status == "" {
				merged.Status = ti.Status
			}
			if merged.HardwarePort == "" {
				merged.HardwarePort = ti.HardwarePort
			}
			if len(merged.Addresses) == 0 {
				merged.Addresses = ti.Addresses
			}
			if !merged.InternetCandidate {
				merged.InternetCandidate = ti.InternetCandidate
			}
			r.Topology.Interfaces[idx] = merged
		} else {
			byName[ni.Name] = len(r.Topology.Interfaces)
			r.Topology.Interfaces = append(r.Topology.Interfaces, ti)
		}
	}
	r.Topology.Routes.DefaultRoute = TopologyRoute{
		Interface:        r.Network.DefaultRoute.Interface,
		Gateway:          r.Network.DefaultRoute.Gateway,
		Reachable:        r.Reachability.Gateway.OK,
		GatewayReachable: r.Reachability.Gateway.OK,
	}
	for _, ni := range r.Network.Interfaces {
		if ni.Name == "" || ni.Name == r.Network.DefaultRoute.Interface || !looksPhysicalInternetCandidate(ni) {
			continue
		}
		r.Topology.Routes.AlternateRoutes = appendUniqueRoute(r.Topology.Routes.AlternateRoutes, TopologyRoute{
			Interface:      ni.Name,
			RawIPReachable: scopedProbeOK(r.Reachability.RawIPs, ni.Name),
			DNSReachable:   scopedProbeOK(r.Reachability.DNSNames, ni.Name),
			HTTPSReachable: scopedProbeOK(r.Reachability.HTTPSTargets, ni.Name),
			ProbeMode:      "scoped_interface_or_active_physical_candidate",
		})
	}
}

func detectTopologyConstraints(r *DiagnosticReport) {
	seen := map[string]bool{}
	var out []TopologyConstraint
	add := func(c TopologyConstraint) {
		if c.ID == "" {
			c.ID = strings.ToLower(c.Class + "-" + c.Interface)
		}
		if c.Confidence == 0 {
			c.Confidence = 0.7
		}
		if seen[c.ID] {
			return
		}
		seen[c.ID] = true
		out = append(out, c)
	}
	for _, c := range r.Topology.ProtectedConstraints {
		add(c)
	}
	for i := range r.Topology.Interfaces {
		ti := &r.Topology.Interfaces[i]
		class, evidence := protectedClassFromInterface(*ti)
		rawClass, rawEvidence := protectedClassFromBoundRaw(r.Raw, ti.Name)
		if class == "" && rawClass != "" {
			class = rawClass
			evidence = rawEvidence
			ti.RoleEvidence = appendUniqueStrings(ti.RoleEvidence, rawEvidence...)
		}
		if class == "" {
			continue
		}
		ti.Protected = true
		ti.Roles = appendUniqueStrings(ti.Roles, roleForClass(class))
		if ti.ProtectionPolicy.ForbiddenActions == nil {
			ti.ProtectionPolicy.ForbiddenActions = append([]string{}, protectedForbiddenActions...)
		}
		if ti.ProtectionPolicy.AllowedActions == nil {
			ti.ProtectionPolicy.AllowedActions = append([]string{}, protectedAllowedActions...)
		}
		add(TopologyConstraint{
			ID:         "protected-" + strings.ToLower(strings.ReplaceAll(class, "_", "-")) + "-" + ti.Name,
			Class:      class,
			Interface:  ti.Name,
			Confidence: maxFloat(ti.RoleConfidence, 0.82),
			Evidence:   appendUniqueStrings(evidence, ti.RoleEvidence...),
		})
	}
	if len(r.Residues.Profiles) > 0 {
		add(TopologyConstraint{
			ID:         "protected-policy-mdm",
			Class:      TopologyClassProtectedPolicy,
			Confidence: 0.9,
			Evidence:   []string{"MDM/profile network policy present; local mutation requires admin policy review"},
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Interface == out[j].Interface {
			return out[i].Class < out[j].Class
		}
		return out[i].Interface < out[j].Interface
	})
	r.Topology.ProtectedConstraints = out
}

func detectRouteCandidates(r *DiagnosticReport) {
	def := r.Network.DefaultRoute.Interface
	if def != "" && interfaceProtected(r.Topology, def) && !r.Reachability.Gateway.OK && hasAlternateRoutableInterface(*r, def) {
		r.Topology.RootCauseCandidates = appendUniqueCandidate(r.Topology.RootCauseCandidates, TopologyCandidate{
			Class:      TopologyClassDefaultRouteStale,
			Interface:  def,
			Confidence: 0.82,
			Evidence:   []string{"default route owner is protected/local-only and alternate physical path exists"},
		})
	}
	for _, ni := range r.Network.Interfaces {
		if ni.IsUTun || strings.HasPrefix(ni.Name, "utun") {
			for _, ip := range ni.IPv4 {
				if strings.HasPrefix(ip, "198.18.") || strings.HasPrefix(ip, "198.19.") {
					r.Topology.RedHerrings = appendUniqueRedHerring(r.Topology.RedHerrings, TopologyRedHerring{
						Class:  TopologyClassTunFakeIPPresent,
						Reason: "TUN/fake-IP is present but cannot override protected topology constraints without route ownership proof",
					})
				}
			}
		}
		if ni.IsBridge {
			r.Topology.RedHerrings = appendUniqueRedHerring(r.Topology.RedHerrings, TopologyRedHerring{
				Class:  TopologyClassVirtualBridge,
				Reason: "bridge interface must not be used as physical internet proof",
			})
		}
	}
}

func buildRepairCorridor(r *DiagnosticReport) {
	if len(r.Topology.ProtectedConstraints) == 0 {
		r.Topology.RepairCorridor = RepairCorridor{MutationAllowed: true}
		return
	}
	r.Topology.RepairCorridor = RepairCorridor{
		MutationAllowed: false,
		AllowedNext: []string{
			"agentlink.route_snapshot",
			"agentlink.network_snapshot",
			"agentlink.incident_report",
			"agentlink.support_bundle",
		},
		GlobalForbidden: []string{
			"dhcp_renew_protected_interface",
			"set_dhcp_protected_service",
			"set_dns_protected_service",
			"route_flush",
			"ifconfig_down_protected_interface",
			"proxy_reset_protected_service",
			"tun_cleanup_before_route_ownership_proof",
			"clean_baseline",
			"switch_vlan_mutation",
		},
		Reason: "protected topology constraint present; gather evidence and create a ticket instead of mutating host network state",
	}
}

func protectedClassFromInterface(ti TopologyInterface) (string, []string) {
	text := strings.ToLower(strings.Join(append(append([]string{}, ti.Roles...), ti.RoleEvidence...), " ") + " " + ti.HardwarePort + " " + ti.NetworkService)
	if text == "" && !ti.Protected {
		return "", nil
	}
	if ti.Protected && text == "" {
		return TopologyClassProtectedLabControl, []string{"interface marked protected"}
	}
	class, evidence := protectedClassFromText(text)
	if class == "" && ti.Protected {
		return TopologyClassProtectedLabControl, []string{"interface marked protected"}
	}
	return class, evidence
}

func protectedClassFromBoundRaw(raw map[string]string, iface string) (string, []string) {
	if iface == "" {
		return "", nil
	}
	lowerIface := strings.ToLower(iface)
	for k, v := range raw {
		key := strings.ToLower(k)
		val := strings.ToLower(v)
		if !rawEntryBoundToInterface(key, val, lowerIface) {
			continue
		}
		if class, evidence := protectedClassFromText(key + " " + val); class != "" {
			return class, append([]string{"bound-to-interface:" + iface}, evidence...)
		}
	}
	return "", nil
}

func rawEntryBoundToInterface(key, val, iface string) bool {
	if iface == "" {
		return false
	}
	if strings.Contains(val, "not bound to "+iface) || strings.Contains(val, "unrelated "+iface) || strings.Contains(val, "not on "+iface) {
		return false
	}
	if key == iface || strings.HasSuffix(key, "."+iface) || strings.Contains(key, iface+".") {
		return true
	}
	for _, marker := range []string{" on " + iface, " interface " + iface, " iface " + iface, " bound to " + iface, " " + iface + " "} {
		if strings.Contains(" "+val+" ", marker) {
			return true
		}
	}
	return false
}

func protectedClassFromText(text string) (string, []string) {
	signals := []struct {
		class string
		terms []string
	}{
		{TopologyClassProtectedAudioVLAN, []string{"dante", "dvs", "mtrx", "_netaudio", "ravenna", "aes67", "avb", "milan", "ptp", "st2110", "audio vlan"}},
		{TopologyClassProtectedVideoVLAN, []string{"ndi", "video vlan", "smpte", "st 2110", "st2110"}},
		{TopologyClassProtectedCameraLAN, []string{"ptz", "atem", "camera control", "blackmagic"}},
		{TopologyClassProtectedShowControl, []string{"art-net", "artnet", "sacn", "sacn", "lighting", "show control"}},
		{TopologyClassProtectedStorage, []string{"iscsi", "nas", "smb direct", "backup interface", "storage network"}},
		{TopologyClassProtectedDirectLink, []string{"thunderbolt bridge", "direct mac", "direct link", "vm/container", "virtual_bridge"}},
		{TopologyClassProtectedLabControl, []string{"lab instrument", "plc", "scada", "measurement lan", "static control"}},
		{TopologyClassProtectedPolicy, []string{"mdm", "corporate profile", "split vpn", "policy route", "scoped dns", "managed proxy"}},
		{TopologyClassProtectedLinkLocal, []string{"bonjour", "link-local service", "airprint", "printer lab"}},
	}
	for _, s := range signals {
		for _, term := range s.terms {
			if strings.Contains(text, term) {
				return s.class, []string{term}
			}
		}
	}
	return "", nil
}

func roleForClass(class string) string {
	switch class {
	case TopologyClassProtectedAudioVLAN:
		return "protected_media"
	case TopologyClassProtectedVideoVLAN:
		return "protected_video"
	case TopologyClassProtectedCameraLAN, TopologyClassProtectedShowControl, TopologyClassProtectedLabControl:
		return "local_control"
	case TopologyClassProtectedStorage:
		return "protected_storage"
	case TopologyClassProtectedPolicy:
		return "policy_owned"
	default:
		return "protected_local"
	}
}

func interfaceProtected(topology TopologyInfo, name string) bool {
	for _, ti := range topology.Interfaces {
		if ti.Name == name && ti.Protected {
			return true
		}
	}
	for _, c := range topology.ProtectedConstraints {
		if c.Interface == name && c.Interface != "" {
			return true
		}
	}
	return false
}

func hasAlternateRoutableInterface(r DiagnosticReport, defaultIface string) bool {
	for _, ni := range r.Network.Interfaces {
		if ni.Name == defaultIface || noisyTopologyInterface(ni.Name) {
			continue
		}
		if strings.EqualFold(ni.Status, "active") && hasUsableIPv4(ni.IPv4) {
			return true
		}
	}
	for _, route := range r.Topology.Routes.AlternateRoutes {
		if route.Interface != "" && route.Interface != defaultIface {
			return true
		}
	}
	return false
}

func looksPhysicalInternetCandidate(ni NetworkInterface) bool {
	if noisyTopologyInterface(ni.Name) || ni.IsBridge || ni.IsUTun {
		return false
	}
	return strings.EqualFold(ni.Status, "active") && hasUsableIPv4(ni.IPv4)
}

func noisyTopologyInterface(name string) bool {
	lower := strings.ToLower(name)
	if lower == "lo0" {
		return true
	}
	for _, prefix := range []string{"utun", "awdl", "llw", "stf", "gif"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func hasUsableIPv4(values []string) bool {
	for _, ip := range values {
		if ip != "" && !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
			return true
		}
	}
	return false
}

func scopedProbeOK(probes map[string]ProbeResult, iface string) bool {
	if iface == "" {
		return false
	}
	for key, probe := range probes {
		text := strings.ToLower(key + " " + probe.Target + " " + probe.Status + " " + probe.RedactedInfo)
		if strings.Contains(text, "%"+strings.ToLower(iface)) || strings.Contains(text, " "+strings.ToLower(iface)) || strings.Contains(text, "over "+strings.ToLower(iface)) {
			if probe.OK {
				return true
			}
		}
	}
	return false
}

func appendUniqueStrings(base []string, values ...string) []string {
	seen := map[string]bool{}
	out := append([]string{}, base...)
	for _, v := range base {
		seen[v] = true
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func appendUniqueRoute(base []TopologyRoute, v TopologyRoute) []TopologyRoute {
	for _, existing := range base {
		if existing.Interface == v.Interface {
			return base
		}
	}
	return append(base, v)
}

func appendUniqueCandidate(base []TopologyCandidate, v TopologyCandidate) []TopologyCandidate {
	for _, existing := range base {
		if existing.Class == v.Class && existing.Interface == v.Interface {
			return base
		}
	}
	return append(base, v)
}

func appendUniqueRedHerring(base []TopologyRedHerring, v TopologyRedHerring) []TopologyRedHerring {
	for _, existing := range base {
		if existing.Class == v.Class {
			return base
		}
	}
	return append(base, v)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
