// cmd/root.go
package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wanwire/geodbq/internal/geo"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/infra/conf"
)

var (
	geoipPath   string
	geositePath string
	maxShow     int
	configPath  string
	sourceIPStr string
	preferIP    string
)

var rootCmd = &cobra.Command{
	Use:   "geodbq",
	Short: "Query geoip.dat / geosite.dat from Xray-core",
	Long: `geodbq is a tool to inspect and query geoip.dat and geosite.dat files used by Xray-core.
Supports default lookup in current directory if paths not specified.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if geoipPath == "" {
			geoipPath = filepath.Join(".", "geoip.dat")
		}
		if geositePath == "" {
			geositePath = filepath.Join(".", "geosite.dat")
		}
	},
}

var queryIPCmd = &cobra.Command{
	Use:   "ip ADDRESS",
	Short: "Query country code(s) for an IP",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		list, err := geo.LoadGeoIP(geoipPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading geoip.dat: %v\n", err)
			os.Exit(1)
		}
		codes := geo.QueryGeoIP(list, args[0])
		if len(codes) == 0 {
			fmt.Printf("No match for %s\n", args[0])
		} else {
			sort.Strings(codes)
			fmt.Printf("Matching country codes for %s: %s\n", args[0], strings.Join(codes, ", "))
		}
	},
}

var queryDomainCmd = &cobra.Command{
	Use:   "domain DOMAIN",
	Short: "Query matching geosite categories for a domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		list, err := geo.LoadGeoSite(geositePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading geosite.dat: %v\n", err)
			os.Exit(1)
		}
		codes := geo.QueryGeoSite(list, args[0])
		if len(codes) == 0 {
			fmt.Printf("No matching geosite categories for '%s'\n", args[0])
		} else {
			sort.Strings(codes)
			fmt.Printf("Matching geosite categories for %s: %s\n", args[0], strings.Join(codes, ", "))
		}
	},
}

var listCategoriesCmd = &cobra.Command{
	Use:   "list-categories",
	Short: "List all available country codes / categories",
	Run: func(cmd *cobra.Command, args []string) {
		anyLoaded := false
		if geoipPath != "" {
			list, err := geo.LoadGeoIP(geoipPath)
			if err == nil {
				codes := geo.ListGeoIPCategories(list)
				sort.Strings(codes)
				fmt.Printf("Available geoip country codes (%d):\n  %s\n", len(codes), strings.Join(codes, ", "))
				anyLoaded = true
			} else {
				fmt.Fprintf(os.Stderr, "Skipping geoip: %v\n", err)
			}
		}
		if geositePath != "" {
			list, err := geo.LoadGeoSite(geositePath)
			if err == nil {
				codes := geo.ListGeoSiteCategories(list)
				sort.Strings(codes)
				fmt.Printf("Available geosite categories (%d):\n  %s\n", len(codes), strings.Join(codes, ", "))
				anyLoaded = true
			} else {
				fmt.Fprintf(os.Stderr, "Skipping geosite: %v\n", err)
			}
		}
		if !anyLoaded {
			fmt.Println("No files loaded. Provide --geoip or --geosite paths if needed.")
		}
	},
}

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Show summary statistics for geoip.dat and/or geosite.dat",
	Run: func(cmd *cobra.Command, args []string) {
		if geoipPath != "" {
			list, err := geo.LoadGeoIP(geoipPath)
			if err == nil {
				fmt.Println("=== geoip.dat summary ===")
				geo.SummarizeGeoIP(list)
			} else {
				fmt.Fprintf(os.Stderr, "geoip summary skipped: %v\n", err)
			}
		}
		if geositePath != "" {
			list, err := geo.LoadGeoSite(geositePath)
			if err == nil {
				fmt.Println("=== geosite.dat summary ===")
				geo.SummarizeGeoSite(list)
			} else {
				fmt.Fprintf(os.Stderr, "geosite summary skipped: %v\n", err)
			}
		}
	},
}

var listRulesCmd = &cobra.Command{
	Use:   "list-rules CATEGORY",
	Short: "List rules (CIDRs or domains) for a specific country code / category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		category := strings.ToLower(args[0])
		if geoipPath != "" {
			list, err := geo.LoadGeoIP(geoipPath)
			if err == nil {
				found := geo.ListGeoIPRules(list, category, maxShow)
				if found {
					return // if found in geoip, no need to check geosite
				}
			}
		}
		if geositePath != "" {
			list, err := geo.LoadGeoSite(geositePath)
			if err == nil {
				geo.ListGeoSiteRules(list, category, maxShow)
			} else {
				fmt.Fprintf(os.Stderr, "geosite list-rules error: %v\n", err)
			}
		}
	},
}

var simulateRouteCmd = &cobra.Command{
	Use:   "simulate-route",
	Short: "Simulate Xray routing decision for a domain or IP using config.json",
	Run: func(cmd *cobra.Command, args []string) {
		configPath := cmd.Flag("config").Value.String()
		domain := cmd.Flag("domain").Value.String()
		ipStr := cmd.Flag("ip").Value.String()
		sourceIPStr := cmd.Flag("source-ip").Value.String()

		if configPath == "" || (domain == "" && ipStr == "") {
			fmt.Fprintln(os.Stderr, "Error: --config required + at least one of --domain or --ip")
			cmd.Help()
			os.Exit(1)
		}

		// Load geosite.dat early (needed for reverse lookup)
		var geositeList *router.GeoSiteList
		if geositePath != "" {
			var err error
			geositeList, err = geo.LoadGeoSite(geositePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load geosite.dat for better simulation: %v\n", err)
			}
		}

		// Read config.json
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		var xrayConf conf.Config
		if err := json.Unmarshal(data, &xrayConf); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config.json: %v\n", err)
			os.Exit(1)
		}

		routerCfg := xrayConf.RouterConfig
		if routerCfg == nil {
			fmt.Println("No routing section found in config.json")
			return
		}

		var rawRules []json.RawMessage
		if routerCfg != nil {
			rawRules = routerCfg.RuleList
		}

		// Build real router configuration (expands geosite:, etc.)
		routerProto, err := routerCfg.Build()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build routing config: %v\n", err)
			os.Exit(1)
		}

		if len(routerProto.Rule) == 0 {
			fmt.Println("No routing rules after building config")
			return
		}

		destIP := net.ParseIP(ipStr)
		srcIP := net.ParseIP(sourceIPStr)

		// Domain strategy handling: resolve domain when needed for IP-based strategies
		var resolvedIPs []net.IP
		ds := routerProto.DomainStrategy
		if domain != "" {
			if ds == router.Config_IpOnDemand {
				ips, err := net.LookupIP(domain)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: DNS lookup failed for %s: %v\n", domain, err)
				} else {
					resolvedIPs = filterIPs(ips, preferIP)
					if len(resolvedIPs) > 0 {
						fmt.Printf("Domain strategy: IpOnDemand\n")
						fmt.Printf("Resolved %s to: %v\n", domain, resolvedIPs)
					}
				}
			}
		}

		matched := false
		for i, rule := range routerProto.Rule {
			var rawRule json.RawMessage
			if i < len(rawRules) {
				rawRule = rawRules[i]
			}

			// Skip rules that require specific inboundTag if we didn't provide one or it doesn't match
			if len(rule.InboundTag) > 0 {
				providedTag := cmd.Flag("inbound-tag").Value.String()
				if providedTag == "" || !contains(rule.InboundTag, providedTag) {
					continue // this rule requires specific inbound → skip if not matching
				}
			}

			// First try: normal match using domain (if present) and provided destIP
			isMatch, details := checkRuleMatch(rule, rawRule, geositeList, domain, destIP, srcIP, cmd)
			if !isMatch && domain != "" && ds == router.Config_IpOnDemand && len(resolvedIPs) > 0 {
				// For IPOnDemand: if not matched by domain-only, try matching using resolved IPs
				for _, rip := range resolvedIPs {
					isMatch, details = checkRuleMatch(rule, rawRule, geositeList, "", rip, srcIP, cmd)
					if isMatch {
						fmt.Printf("\n→ Falling back to IP-based match (IpOnDemand): %s\n", rip.String())
						break
					}
				}
			}

			if isMatch {
				matched = true
				fmt.Printf("\nMatched rule #%d:\n", i+1)
				for _, detail := range details {
					fmt.Printf("  • %s\n", detail)
				}

				// Handle the target (outbound or balancer)
				switch target := rule.TargetTag.(type) {
				case *router.RoutingRule_Tag:
					fmt.Printf("→ Outbound tag: %s\n", target.Tag)

				case *router.RoutingRule_BalancingTag:
					balancerTag := target.BalancingTag
					fmt.Printf("→ Balancer tag: %s\n", balancerTag)

					// Find and show balancer details from original config
					foundBalancer := false
					for _, bal := range routerCfg.Balancers {
						if bal.Tag == balancerTag {
							foundBalancer = true
							fmt.Printf("  Strategy: %s\n", bal.Strategy.Type)
							if len(bal.Selectors) > 0 {
								fmt.Printf("  Candidates: %s\n", strings.Join(bal.Selectors, ", "))
							}
							if bal.FallbackTag != "" {
								fmt.Printf("  Fallback: %s\n", bal.FallbackTag)
							}
							break
						}
					}
					if !foundBalancer {
						fmt.Printf("  (balancer definition not found in config)\n")
					}

				default:
					fmt.Println("→ No target tag or balancer specified in this rule")
				}

				break // first-match semantics
			}
		}

		// If DomainStrategy is IpIfNonMatch and we didn't match earlier,
		// resolve the domain and try matching again using resolved IPs only.
		if !matched && domain != "" && ds == router.Config_IpIfNonMatch {
			ips, err := net.LookupIP(domain)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: DNS lookup failed for %s: %v\n", domain, err)
			} else {
				filteredIPs := filterIPs(ips, preferIP)
				if len(filteredIPs) > 0 {
					fmt.Printf("Domain strategy: IpIfNonMatch\n")
					fmt.Printf("No domain match found, resolving %s to: %v\n", domain, filteredIPs)
				}
				for _, rip := range filteredIPs {
					for i, rule := range routerProto.Rule {
						var rawRule json.RawMessage
						if i < len(rawRules) {
							rawRule = rawRules[i]
						}

						// Skip rules that require specific inboundTag if we didn't provide one or it doesn't match
						if len(rule.InboundTag) > 0 {
							providedTag := cmd.Flag("inbound-tag").Value.String()
							if providedTag == "" || !contains(rule.InboundTag, providedTag) {
								continue // this rule requires specific inbound → skip if not matching
							}
						}

						isMatch, details := checkRuleMatch(rule, rawRule, geositeList, "", rip, srcIP, cmd)
						if isMatch {
							matched = true
							fmt.Printf("\n→ Matched rule #%d using resolved IP %s (IpIfNonMatch):\n", i+1, rip.String())
							for _, detail := range details {
								fmt.Printf("  • %s\n", detail)
							}

							// Handle the target (outbound or balancer)
							switch target := rule.TargetTag.(type) {
							case *router.RoutingRule_Tag:
								fmt.Printf("→ Outbound tag: %s\n", target.Tag)

							case *router.RoutingRule_BalancingTag:
								balancerTag := target.BalancingTag
								fmt.Printf("→ Balancer tag: %s\n", balancerTag)

								// Find and show balancer details from original config
								foundBalancer := false
								for _, bal := range routerCfg.Balancers {
									if bal.Tag == balancerTag {
										foundBalancer = true
										fmt.Printf("  Strategy: %s\n", bal.Strategy.Type)
										if len(bal.Selectors) > 0 {
											fmt.Printf("  Candidates: %s\n", strings.Join(bal.Selectors, ", "))
										}
										if bal.FallbackTag != "" {
											fmt.Printf("  Fallback: %s\n", bal.FallbackTag)
										}
										break
									}
								}
								if !foundBalancer {
									fmt.Printf("  (balancer definition not found in config)\n")
								}

							default:
								fmt.Println("→ No target tag or balancer specified in this rule")
							}

							break // first-match semantics
						}
					}
					if matched {
						break
					}
				}
			}
		}

		if !matched {
			fmt.Println("\nNo rule matched this traffic.")
			outbounds := xrayConf.OutboundConfigs
			if len(outbounds) > 0 {
				fmt.Printf("→ Falls back to default outbound %s\n", outbounds[0].Tag)
			} else {
				fmt.Println("→ Falls back to 'direct'")
			}
		}
	},
}

// Helper
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
		if p == "ipv4" || p == "v4" {
			if ip.To4() != nil {
				out = append(out, ip)
			}
		} else if p == "ipv6" || p == "v6" {
			if ip.To4() == nil {
				out = append(out, ip)
			}
		} else {
			// unknown value: treat as any
			out = append(out, ip)
		}
	}
	return out
}

func init() {
	rootCmd.PersistentFlags().StringVar(&geoipPath, "geoip", "", "Path to geoip.dat (default: ./geoip.dat)")
	rootCmd.PersistentFlags().StringVar(&geositePath, "geosite", "", "Path to geosite.dat (default: ./geosite.dat)")

	listRulesCmd.Flags().IntVar(&maxShow, "max-show", -1, "maximum number of items to show (-1 = all)")

	simulateRouteCmd.Flags().StringVar(&configPath, "config", "config.json", "Path to Xray config.json (default: config.json)")
	simulateRouteCmd.Flags().String("domain", "", "Domain to simulate routing for")
	simulateRouteCmd.Flags().String("ip", "", "IP address to simulate routing for")
	simulateRouteCmd.Flags().StringVar(&sourceIPStr, "source-ip", "127.0.0.1", "Source IP to simulate (for source_geoip rules)")
	simulateRouteCmd.Flags().StringVar(&preferIP, "prefer-ip", "", "Prefer IP family for DNS lookups: ipv4, ipv6 (default: any)")
	simulateRouteCmd.Flags().String("inbound-tag", "", "Simulate traffic from this inbound tag (e.g. socks, http)")
	simulateRouteCmd.Flags().String("port", "", "Destination port to simulate (e.g. 80, 443)")
	simulateRouteCmd.Flags().String("source-port", "", "Source port (e.g. 12345)")
	simulateRouteCmd.Flags().String("protocol", "", "Protocol (e.g. bittorrent, http, tls)")
	simulateRouteCmd.Flags().String("network", "", "Network (tcp, udp)")
	simulateRouteCmd.Flags().String("user-email", "", "User email for matching")

	rootCmd.AddCommand(queryIPCmd)
	rootCmd.AddCommand(queryDomainCmd)
	rootCmd.AddCommand(listCategoriesCmd)
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(listRulesCmd)
	rootCmd.AddCommand(simulateRouteCmd)
}

// Helper functions
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
		// Add --source-port flag if needed
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
