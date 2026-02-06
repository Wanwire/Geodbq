package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wanwire/geodbq/internal/geo"
	"github.com/xtls/xray-core/app/router"
)

// contains checks if a string slice contains an item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// filterIPs filters DNS results according to preferIP setting
func filterIPs(ips []net.IP, prefer string) []net.IP {
	if prefer == "" {
		return ips
	}
	p := strings.ToLower(prefer)
	var out []net.IP
	for _, ip := range ips {
		switch p {
		case "ipv4", "v4":
			if ip.To4() != nil {
				out = append(out, ip)
			}
		case "ipv6", "v6":
			if ip.To4() == nil {
				out = append(out, ip)
			}
		default:
			// unknown value: treat as any
			out = append(out, ip)
		}
	}
	return out
}

// checkRuleMatch evaluates if a routing rule matches the given criteria
func checkRuleMatch(
	rule *router.RoutingRule,
	rawRule json.RawMessage,
	geoSiteList *router.GeoSiteList,
	domain string,
	destIP net.IP,
	srcIP net.IP,
	cmd *cobra.Command,
) (bool, []string) {

	var details []string

	// Protocol
	if len(rule.Protocol) > 0 {
		protoStr := cmd.Flag("protocol").Value.String()
		if protoStr == "" {
			return false, nil
		}
		protoLower := strings.ToLower(protoStr)
		protoMatched := false
		for _, p := range rule.Protocol {
			if strings.ToLower(p) == protoLower {
				protoMatched = true
				details = append(details, fmt.Sprintf("Protocol matched: %s", p))
				break
			}
		}
		if !protoMatched {
			return false, nil
		} else {
			ruleDetails := describeProtocolMatch(rawRule, protoStr)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// Networks (tcp/udp — assume "tcp" if not specified)
	if len(rule.Networks) > 0 {
		netStr := cmd.Flag("network").Value.String()
		if netStr == "" {
			netStr = "tcp" // default assumption
		}
		netLower := strings.ToLower(netStr)
		netMatched := false
		for _, n := range rule.Networks {
			if strings.ToLower(n.String()) == netLower {
				netMatched = true
				details = append(details, fmt.Sprintf("Network matched: %s", n.String()))
				break
			}
		}
		if !netMatched {
			return false, nil
		} else {
			ruleDetails := describeNetworkMatch(rawRule)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// Domain
	if domain != "" && len(rule.Domain) > 0 {
		var matchedEntry *router.Domain
		domainMatched := false
		for _, d := range rule.Domain {
			if geo.MatchDomain(d, domain) {
				matchedEntry = d
				domainMatched = true
				details = append(details, fmt.Sprintf("Domain matched [%s] %s", geo.DomainTypeName(d.Type), d.Value))
				break
			}
		}

		if !domainMatched {
			return false, nil
		} else if matchedEntry != nil {
			ruleDetails := describeDomainMatch(rawRule, matchedEntry, geoSiteList, domain)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// GeoIP (dest)
	if destIP != nil && len(rule.Geoip) > 0 {
		ipMatched := false
		for _, g := range rule.Geoip {
			for _, c := range g.Cidr {
				if cidrContains(c, destIP) {
					ipMatched = true
					details = append(details, fmt.Sprintf("Dest IP matched geoip:%s (%s/%d)", g.CountryCode, net.IP(c.Ip).String(), c.Prefix))
					break
				}
			}
			if ipMatched {
				break
			}
		}
		if !ipMatched {
			return false, nil
		} else {
			ruleDetails := describeGeoIPMatch(rawRule, "geoip")
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// SourceGeoIP
	if srcIP != nil && len(rule.SourceGeoip) > 0 {
		srcMatched := false
		for _, geoEntry := range rule.SourceGeoip {
			for _, c := range geoEntry.Cidr {
				if cidrContains(c, srcIP) {
					srcMatched = true
					details = append(details, fmt.Sprintf("Source IP matched geoip:%s (%s/%d)", geoEntry.CountryCode, net.IP(c.Ip).String(), c.Prefix))
					break
				}
			}
			if srcMatched {
				break
			}
		}
		if !srcMatched {
			return false, nil
		} else {
			ruleDetails := describeGeoIPMatch(rawRule, "sourceGeoip")
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// Port
	if rule.PortList != nil && len(rule.PortList.Range) > 0 {
		portStr := cmd.Flag("port").Value.String()
		if portStr == "" {
			return false, nil
		}
		port, err := strconv.ParseUint(portStr, 10, 32)
		if err != nil {
			return false, nil
		}
		portMatched := false
		for _, r := range rule.PortList.Range {
			if uint32(port) >= r.From && uint32(port) <= r.To {
				portMatched = true
				details = append(details, fmt.Sprintf("Port matched: %d (range %d-%d)", port, r.From, r.To))
				break
			}
		}
		if !portMatched {
			return false, nil
		} else {
			ruleDetails := describePortMatch(rawRule, false)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// SourcePortList
	if rule.SourcePortList != nil && len(rule.SourcePortList.Range) > 0 {
		sourcePortStr := cmd.Flag("source-port").Value.String()
		if sourcePortStr == "" {
			return false, nil
		}
		port, err := strconv.ParseUint(sourcePortStr, 10, 32)
		if err != nil {
			return false, nil
		}
		portMatched := false
		for _, r := range rule.SourcePortList.Range {
			if uint32(port) >= r.From && uint32(port) <= r.To {
				portMatched = true
				details = append(details, fmt.Sprintf("Source port matched: %d (range %d-%d)", port, r.From, r.To))
				break
			}
		}
		if !portMatched {
			return false, nil
		} else {
			ruleDetails := describePortMatch(rawRule, true)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// UserEmail
	if len(rule.UserEmail) > 0 {
		userEmail := cmd.Flag("user-email").Value.String()
		if userEmail == "" {
			return false, nil
		}
		userMatched := false
		for _, u := range rule.UserEmail {
			if u == userEmail {
				userMatched = true
				details = append(details, fmt.Sprintf("User email matched: %s", u))
				break
			}
		}
		if !userMatched {
			return false, nil
		} else {
			ruleDetails := describeUserEmailMatch(rawRule, userEmail)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	// Inbound tag
	if len(rule.InboundTag) > 0 {
		inboundTag := cmd.Flag("inbound-tag").Value.String()
		if inboundTag == "" {
			return false, nil
		}
		inboundTagMatched := false
		for _, t := range rule.InboundTag {
			if t == inboundTag {
				inboundTagMatched = true
				details = append(details, fmt.Sprintf("Inbound tag matched: %s", t))
				break
			}
		}
		if !inboundTagMatched {
			return false, nil
		} else {
			ruleDetails := describeInboundTagMatch(rawRule, inboundTag)
			if len(ruleDetails) > 0 {
				details = append(details, ruleDetails...)
			}
		}
	}

	matched := len(details) > 0
	return matched, details
}

// cidrContains checks if an IP is contained within a CIDR block
func cidrContains(cidr *router.CIDR, ip net.IP) bool {
	prefix := int(cidr.Prefix)
	ipBytes := cidr.Ip
	if len(ipBytes) == 4 && prefix > 32 {
		prefix = 32
	} else if len(ipBytes) == 16 && prefix > 128 {
		prefix = 128
	}
	_, n, err := net.ParseCIDR(fmt.Sprintf("%s/%d", net.IP(ipBytes), prefix))
	if err != nil {
		return false
	}
	return n.Contains(ip)
}

// describeDomainMatch returns description lines for a domain match,
// including both the resolved (expanded) match and the original config condition
func describeDomainMatch(
	rawRule json.RawMessage,
	matchedEntry *router.Domain,
	geositeList *router.GeoSiteList,
	queryDomain string,
) []string {
	var lines []string

	if matchedEntry == nil {
		return nil // no match → caller should reject the rule
	}

	// Try to find original geosite:/domain:/regexp:/full: condition
	if len(rawRule) == 0 {
		return lines
	}

	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err != nil {
		return lines // can't parse → only show resolved
	}

	origDomains, ok := ruleMap["domain"].([]interface{})
	if !ok {
		return lines
	}

	for _, item := range origDomains {
		cond, ok := item.(string)
		if !ok {
			continue
		}

		if strings.HasPrefix(cond, "geosite:") {
			category := strings.TrimPrefix(cond, "geosite:")
			entry := geo.FindGeoSiteEntry(geositeList, category)
			if entry != nil {
				for _, expanded := range entry.Domain {
					if geo.MatchDomain(expanded, queryDomain) {
						lines = append(lines, fmt.Sprintf("Matched original condition: %s", cond))
						return lines // usually return after first good match
					}
				}
			}
		} else {
			// direct domain/full/regexp from config
			if directMatch(cond, queryDomain) {
				lines = append(lines, fmt.Sprintf("Matched original condition: %s", cond))
				return lines
			}
		}
	}

	return lines
}

// directMatch checks direct domain conditions from config
func directMatch(cond string, query string) bool {
	q := strings.ToLower(query)
	if strings.HasPrefix(cond, "domain:") {
		v := strings.ToLower(strings.TrimPrefix(cond, "domain:"))
		return q == v || strings.HasSuffix(q, "."+v)
	}
	if strings.HasPrefix(cond, "full:") {
		return strings.ToLower(strings.TrimPrefix(cond, "full:")) == q
	}
	if strings.HasPrefix(cond, "regexp:") {
		pat := strings.TrimPrefix(cond, "regexp:")
		r, err := regexp.Compile(pat)
		return err == nil && r.MatchString(q)
	}
	return false
}

// describeProtocolMatch returns description lines for protocol matching
func describeProtocolMatch(rawRule json.RawMessage, queryProtocol string) []string {
	var lines []string

	// Try to show original config value
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		if protocols, ok := ruleMap["protocol"].([]interface{}); ok {
			for _, item := range protocols {
				if p, ok := item.(string); ok && strings.EqualFold(p, queryProtocol) {
					lines = append(lines, fmt.Sprintf("From config protocol: %s", p))
					break
				}
			}
		}
	}

	return lines
}

// describeNetworkMatch returns description lines for network matching
func describeNetworkMatch(rawRule json.RawMessage) []string {
	var lines []string

	// original config
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		if netList, ok := ruleMap["network"].(string); ok {
			lines = append(lines, fmt.Sprintf("From config network: %s", netList))
		}
	}

	return lines
}

// describeGeoIPMatch returns description lines for geoip matching
func describeGeoIPMatch(rawRule json.RawMessage, fieldName string) []string {
	var lines []string

	// Try original config geoip / ip field
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		key := "geoip"
		if fieldName == "sourceGeoip" {
			key = "sourceGeoip"
		}
		if geoItems, ok := ruleMap[key].([]interface{}); ok {
			for _, item := range geoItems {
				if m, ok := item.(map[string]interface{}); ok {
					if cc, ok := m["countryCode"].(string); ok {
						lines = append(lines, fmt.Sprintf("From config %s: %s", key, cc))
						break
					}
				}
			}
		} else if ipList, ok := ruleMap["ip"].([]interface{}); ok && fieldName == "geoip" {
			// sometimes people write "ip": ["geoip:cn", "1.2.3.0/24"]
			for _, item := range ipList {
				if s, ok := item.(string); ok && strings.HasPrefix(s, "geoip:") {
					lines = append(lines, fmt.Sprintf("From config ip: %s", s))
				}
			}
		}
	}

	return lines
}

// describePortMatch returns description lines for port matching
func describePortMatch(rawRule json.RawMessage, isSource bool) []string {
	var lines []string

	// original config
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		key := "port"
		if isSource {
			key = "sourcePort"
		}
		if p, ok := ruleMap[key]; ok {
			lines = append(lines, fmt.Sprintf("From config %s: %v", key, p))
		}
	}

	return lines
}

// describeUserEmailMatch returns description lines for user email matching
func describeUserEmailMatch(rawRule json.RawMessage, queryEmail string) []string {
	var lines []string

	// original
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		if emails, ok := ruleMap["userEmail"].([]interface{}); ok {
			for _, item := range emails {
				if s, ok := item.(string); ok && s == queryEmail {
					lines = append(lines, fmt.Sprintf("From config userEmail: %s", s))
					break
				}
			}
		}
	}

	return lines
}

// describeInboundTagMatch returns description lines for inbound tag matching
func describeInboundTagMatch(rawRule json.RawMessage, providedTag string) []string {
	var lines []string

	// original config
	var ruleMap map[string]interface{}
	if err := json.Unmarshal(rawRule, &ruleMap); err == nil {
		if tags, ok := ruleMap["inboundTag"].([]interface{}); ok {
			for _, item := range tags {
				if s, ok := item.(string); ok && s == providedTag {
					lines = append(lines, fmt.Sprintf("From config inboundTag: %s", s))
					break
				}
			}
		}
	}

	return lines
}
