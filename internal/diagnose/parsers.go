package diagnose

import (
	"bufio"
	"net"
	"regexp"
	"strings"
)

func ParseNetworkServices(text string) []NetworkService {
	var out []NetworkService
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		svc := NetworkService{Name: line}
		if strings.HasPrefix(line, "*") {
			svc.Disabled = true
			svc.Name = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		}
		if svc.Name != "" {
			out = append(out, svc)
		}
	}
	return out
}

func ParseHardwarePorts(text string) []HardwarePort {
	var ports []HardwarePort
	var cur HardwarePort
	flush := func() {
		if cur.Port != "" || cur.Device != "" {
			ports = append(ports, cur)
		}
		cur = HardwarePort{}
	}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Hardware Port:") {
			flush()
			cur.Port = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
			continue
		}
		if strings.HasPrefix(line, "Device:") {
			cur.Device = strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			continue
		}
		if strings.HasPrefix(line, "Ethernet Address:") {
			cur.EthernetAddress = strings.TrimSpace(strings.TrimPrefix(line, "Ethernet Address:"))
		}
	}
	flush()
	return ports
}

func ParseScutilProxy(text string) ProxySummary {
	p := ProxySummary{Raw: text}
	scanner := bufio.NewScanner(strings.NewReader(text))
	inExceptions := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "HTTPEnable"):
			p.HTTPEnabled = parseEnabled(line)
		case strings.HasPrefix(line, "HTTPSEnable"):
			p.HTTPSEnabled = parseEnabled(line)
		case strings.HasPrefix(line, "SOCKSEnable"):
			p.SOCKSEnabled = parseEnabled(line)
		case strings.HasPrefix(line, "ProxyAutoConfigEnable"):
			p.PACEnabled = parseEnabled(line)
		case strings.HasPrefix(line, "ExceptionsList"):
			inExceptions = true
		case inExceptions && strings.Contains(line, ":"):
			parts := strings.SplitN(line, ":", 2)
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"")
			val = strings.TrimSuffix(val, ",")
			if val != "" && val != "{" && val != "}" {
				p.Exceptions = append(p.Exceptions, val)
			}
		}
		if inExceptions && strings.Contains(line, "}") {
			inExceptions = false
		}
	}
	p.Dirty = p.HTTPEnabled || p.HTTPSEnabled || p.SOCKSEnabled || p.PACEnabled
	return p
}

func parseEnabled(line string) bool {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return false
	}
	val := strings.TrimSpace(parts[len(parts)-1])
	return val == "1" || strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")
}

func ParseRouteDefault(routeGet string, netstat string) DefaultRoute {
	dr := DefaultRoute{Raw: strings.TrimSpace(routeGet)}
	for _, raw := range strings.Split(routeGet, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "gateway:") {
			dr.Gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			dr.Present = dr.Gateway != ""
		}
		if strings.HasPrefix(line, "interface:") {
			dr.Interface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	if dr.Present {
		return dr
	}
	for _, raw := range strings.Split(netstat, "\n") {
		line := strings.Join(strings.Fields(raw), " ")
		if strings.HasPrefix(line, "default ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				dr.Present = true
				dr.Gateway = fields[1]
			}
			if len(fields) >= 4 {
				dr.Interface = fields[len(fields)-1]
			}
			return dr
		}
	}
	return dr
}

func ParseIfconfig(text string) []NetworkInterface {
	var out []NetworkInterface
	var cur *NetworkInterface
	header := regexp.MustCompile(`^([A-Za-z0-9_.-]+):\s`)
	flush := func() {
		if cur != nil {
			cur.LinkLocalOnly = interfaceLinkLocalOnly(*cur)
			out = append(out, *cur)
		}
		cur = nil
	}
	for _, raw := range strings.Split(text, "\n") {
		if match := header.FindStringSubmatch(raw); match != nil {
			flush()
			name := match[1]
			cur = &NetworkInterface{Name: name, IsUTun: strings.HasPrefix(name, "utun"), IsBridge: strings.HasPrefix(name, "bridge")}
			continue
		}
		if cur == nil {
			continue
		}
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "status:") {
			cur.Status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
		}
		if strings.HasPrefix(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				cur.IPv4 = append(cur.IPv4, fields[1])
			}
		}
		if strings.HasPrefix(line, "inet6 ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				cur.IPv6 = append(cur.IPv6, strings.TrimSuffix(fields[1], "%"+cur.Name))
			}
		}
	}
	flush()
	return out
}

func interfaceLinkLocalOnly(iface NetworkInterface) bool {
	var hasNonLoopIP bool
	var hasLinkLocal bool
	for _, ip := range iface.IPv4 {
		if strings.HasPrefix(ip, "127.") {
			continue
		}
		hasNonLoopIP = true
		if strings.HasPrefix(ip, "169.254.") {
			hasLinkLocal = true
		} else {
			return false
		}
	}
	if hasNonLoopIP {
		return hasLinkLocal
	}
	for _, ip := range iface.IPv6 {
		if strings.HasPrefix(strings.ToLower(ip), "fe80:") {
			hasLinkLocal = true
		} else if parsed := net.ParseIP(ip); parsed != nil && !parsed.IsLoopback() {
			return false
		}
	}
	return hasLinkLocal
}

func ParseDNSSummary(scutilDNS string, resolutions map[string]ProbeResult) DNSSummary {
	out := DNSSummary{Resolution: resolutions, Raw: scutilDNS}
	seenNS := map[string]bool{}
	seenSearch := map[string]bool{}
	for _, raw := range strings.Split(scutilDNS, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "nameserver[") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && !seenNS[val] {
					out.Nameservers = append(out.Nameservers, val)
					seenNS[val] = true
				}
			}
		}
		if strings.HasPrefix(line, "search domain[") || strings.HasPrefix(line, "domain") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && !seenSearch[val] {
					out.Search = append(out.Search, val)
					seenSearch[val] = true
				}
			}
		}
	}
	return out
}

func IsLinkLocalIPv4(ip string) bool {
	return strings.HasPrefix(ip, "169.254.")
}
