// cmd/root.go
package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

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
	outputLang  string
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

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Output the structure of conf.Config using reflection",
	Run: func(cmd *cobra.Command, args []string) {
		t := reflect.TypeOf(conf.Config{})
		if outputLang == "swift" || outputLang == "s" {
			code := generateSwiftStructCode(t)
			fmt.Print(code)
		} else if outputLang == "kotlin" || outputLang == "k" {
			code := generateKotlinStructCode(t)
			fmt.Print(code)
		} else {
			code := generateFullStructCode(t)
			fmt.Print(code)
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
		cfg := cmd.Flag("config").Value.String()
		domain := cmd.Flag("domain").Value.String()
		ipStr := cmd.Flag("ip").Value.String()
		srcIP := cmd.Flag("source-ip").Value.String()

		if cfg == "" || (domain == "" && ipStr == "") {
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
		data, err := os.ReadFile(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		var xrayConf conf.Config
		if err := json.Unmarshal(data, &xrayConf); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config.json: %v\n", err)
			os.Exit(1)
		}

		// Validate config using Xray-core's Build method (same validation as Xray startup)
		if _, err := xrayConf.Build(); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
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

		// Validate each routing rule has at least one effective field
		for i, rule := range routerProto.Rule {
			_, err := rule.BuildCondition()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid configuration: routing rule #%d has no effective fields\n", i+1)
				os.Exit(1)
			}
		}

		if len(routerProto.Rule) == 0 {
			fmt.Println("No routing rules after building config")
			return
		}

		destIP := net.ParseIP(ipStr)
		sourceIP := net.ParseIP(srcIP)

		// Domain strategy handling: resolve domain when needed for IP-based strategies
		var resolvedIPs []net.IP
		ds := routerProto.DomainStrategy
		if ds == router.Config_IpOnDemand && domain != "" {
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
			isMatch, details := checkRuleMatch(rule, rawRule, geositeList, domain, destIP, sourceIP, cmd)
			if !isMatch && domain != "" && ds == router.Config_IpOnDemand && len(resolvedIPs) > 0 {
				// For IPOnDemand: if not matched by domain-only, try matching using resolved IPs
				for _, rip := range resolvedIPs {
					isMatch, details = checkRuleMatch(rule, rawRule, geositeList, "", rip, sourceIP, cmd)
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

							isMatch, details := checkRuleMatch(rule, rawRule, geositeList, "", rip, sourceIP, cmd)
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

	extractCmd.Flags().StringVar(&outputLang, "lang", "golang", "Output language: golang, go, swift, s, kotlin, k")

	rootCmd.AddCommand(queryIPCmd)
	rootCmd.AddCommand(queryDomainCmd)
	rootCmd.AddCommand(listCategoriesCmd)
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(listRulesCmd)
	rootCmd.AddCommand(simulateRouteCmd)
	rootCmd.AddCommand(extractCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func typeToString(t reflect.Type) string {
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return "json.RawMessage"
	}

	if t.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && t.Name() == "Duration" {
		return "duration.Duration"
	}

	switch t.Kind() {
	case reflect.Ptr:
		inner := t.Elem()
		if inner.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && inner.Name() == "Duration" {
			return "*duration.Duration"
		}
		return "*" + typeToString(inner)
	case reflect.Slice:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "json.RawMessage"
		}
		elem := t.Elem()
		if elem.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && elem.Name() == "Duration" {
			return "[]duration.Duration"
		}
		return "[]" + typeToString(elem)
	case reflect.Map:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "map[" + typeToString(t.Key()) + "]json.RawMessage"
		}
		return "map[" + typeToString(t.Key()) + "]" + typeToString(t.Elem())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float64"
	case reflect.Bool:
		return "bool"
	case reflect.String:
		return "string"
	case reflect.Interface:
		return "interface{}"
	default:
		return t.Name()
	}
}

func isOpaqueType(t reflect.Type) bool {
	var structType reflect.Type
	if t.Kind() == reflect.Ptr {
		structType = t.Elem()
	} else {
		structType = t
	}

	if structType.Kind() != reflect.Struct {
		return false
	}

	publicFieldCount := 0
	hasJsonTagOnPublic := false

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath == "" {
			publicFieldCount++
			if field.Tag.Get("json") != "" && field.Tag.Get("json") != "-" {
				hasJsonTagOnPublic = true
			}
		}
	}

	if publicFieldCount == 0 {
		return true
	}
	return !hasJsonTagOnPublic
}

func generateFullStructCode(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	typeQueue := []reflect.Type{t}
	typeOrder := []reflect.Type{}
	visited := make(map[string]bool)
	usedDuration := false

	for len(typeQueue) > 0 {
		current := typeQueue[0]
		typeQueue = typeQueue[1:]

		typeName := current.Name()
		if typeName == "" {
			continue
		}
		if visited[typeName] {
			continue
		}
		visited[typeName] = true
		typeOrder = append(typeOrder, current)

		for i := 0; i < current.NumField(); i++ {
			field := current.Field(i)
			fieldType := field.Type

			var fieldTypeName string
			if fieldType.Kind() == reflect.Ptr {
				fieldTypeName = fieldType.Elem().Name()
			} else {
				fieldTypeName = fieldType.Name()
			}

			if fieldTypeName == "Duration" {
				usedDuration = true
			}

			if fieldType.Kind() == reflect.Ptr {
				if fieldType.Elem().Kind() == reflect.Struct {
					elemType := fieldType.Elem()
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType) {
						typeQueue = append(typeQueue, elemType)
					}
				}
				continue
			}

			if fieldTypeName != "" && !visited[fieldTypeName] {
				if fieldType.Kind() == reflect.Struct && !isOpaqueType(fieldType) {
					typeQueue = append(typeQueue, fieldType)
				}
			}

			if fieldType.Kind() == reflect.Slice {
				elemType := fieldType.Elem()
				if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
					elemTypeName := elemType.Elem().Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType.Elem()) {
						typeQueue = append(typeQueue, elemType.Elem())
					}
				} else if elemType.Kind() == reflect.Struct && !isOpaqueType(elemType) {
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] {
						typeQueue = append(typeQueue, elemType)
					}
				}
			}

			if fieldType.Kind() == reflect.Map {
				valueType := fieldType.Elem()
				if valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
					valueTypeName := valueType.Elem().Name()
					if valueTypeName != "" && !visited[valueTypeName] && !isOpaqueType(valueType.Elem()) {
						typeQueue = append(typeQueue, valueType.Elem())
					}
				} else if valueType.Kind() == reflect.Struct && !isOpaqueType(valueType) {
					valueTypeName := valueType.Name()
					if valueTypeName != "" && !visited[valueTypeName] {
						typeQueue = append(typeQueue, valueType)
					}
				}
			}
		}
	}

	var lines []string

	lines = append(lines, "package main")
	lines = append(lines, "")
	lines = append(lines, "import (")
	lines = append(lines, "  \"encoding/json\"")
	if usedDuration {
		lines = append(lines, "  \"github.com/xtls/xray-core/infra/conf/cfgcommon/duration\"")
	}
	lines = append(lines, ")")
	lines = append(lines, "")

	sort.Slice(typeOrder, func(i, j int) bool { return typeOrder[i].Name() < typeOrder[j].Name() })

	for _, structType := range typeOrder {
		lines = append(lines, generateSingleStruct(structType))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func generateSingleStruct(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	var lines []string
	lines = append(lines, "type "+t.Name()+" struct {")

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		fieldType := field.Type

		typeStr := typeToString(fieldType)

		if jsonTag == "" || jsonTag == "-" {
			if isOpaqueType(fieldType) {
				lines = append(lines, fmt.Sprintf("  %s json.RawMessage", field.Name))
			} else {
				lines = append(lines, fmt.Sprintf("  %s %s", field.Name, typeStr))
			}
			continue
		}

		if isOpaqueType(fieldType) {
			lines = append(lines, fmt.Sprintf("  %s json.RawMessage `json:\"%s\"`", field.Name, jsonTag))
			continue
		}

		lines = append(lines, fmt.Sprintf("  %s %s `json:\"%s\"`", field.Name, typeStr, jsonTag))
	}

	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	result := strings.Builder{}
	upperNext := false
	for _, r := range s {
		if r == '_' || r == '-' {
			upperNext = true
			continue
		}
		if upperNext {
			result.WriteRune(unicode.ToUpper(r))
			upperNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func needsSwiftCodingKeys(t reflect.Type) bool {
	var structType reflect.Type
	if t.Kind() == reflect.Ptr {
		structType = t.Elem()
	} else {
		structType = t
	}

	if structType.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		tagValue := strings.Split(jsonTag, ",")[0]
		camelName := toCamelCase(tagValue)
		if camelName != tagValue {
			return true
		}
	}
	return false
}

func swiftTypeToString(t reflect.Type) string {
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return "AnyCodable"
	}

	if t.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && t.Name() == "Duration" {
		return "Int64"
	}

	switch t.Kind() {
	case reflect.Ptr:
		inner := t.Elem()
		if inner.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && inner.Name() == "Duration" {
			return "Int64?"
		}
		return swiftTypeToString(inner) + "?"
	case reflect.Slice:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "[AnyCodable]"
		}
		elem := t.Elem()
		if elem.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && elem.Name() == "Duration" {
			return "[Int64]"
		}
		return "[" + swiftTypeToString(elem) + "]"
	case reflect.Map:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "[String: AnyCodable]"
		}
		return "[" + swiftTypeToString(t.Key()) + ": " + swiftTypeToString(t.Elem()) + "]"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "Int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "UInt64"
	case reflect.Float32, reflect.Float64:
		return "Double"
	case reflect.Bool:
		return "Bool"
	case reflect.String:
		return "String"
	case reflect.Interface:
		return "Any?"
	default:
		return t.Name()
	}
}

func generateSwiftStructCode(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	typeQueue := []reflect.Type{t}
	typeOrder := []reflect.Type{}
	visited := make(map[string]bool)

	for len(typeQueue) > 0 {
		current := typeQueue[0]
		typeQueue = typeQueue[1:]

		typeName := current.Name()
		if typeName == "" {
			continue
		}
		if visited[typeName] {
			continue
		}
		visited[typeName] = true
		typeOrder = append(typeOrder, current)

		for i := 0; i < current.NumField(); i++ {
			field := current.Field(i)
			fieldType := field.Type

			var fieldTypeName string
			if fieldType.Kind() == reflect.Ptr {
				fieldTypeName = fieldType.Elem().Name()
			} else {
				fieldTypeName = fieldType.Name()
			}

			if fieldType.Kind() == reflect.Ptr {
				if fieldType.Elem().Kind() == reflect.Struct {
					elemType := fieldType.Elem()
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType) {
						typeQueue = append(typeQueue, elemType)
					}
				}
				continue
			}

			if fieldTypeName != "" && !visited[fieldTypeName] {
				if fieldType.Kind() == reflect.Struct && !isOpaqueType(fieldType) {
					typeQueue = append(typeQueue, fieldType)
				}
			}

			if fieldType.Kind() == reflect.Slice {
				elemType := fieldType.Elem()
				if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
					elemTypeName := elemType.Elem().Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType.Elem()) {
						typeQueue = append(typeQueue, elemType.Elem())
					}
				} else if elemType.Kind() == reflect.Struct && !isOpaqueType(elemType) {
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] {
						typeQueue = append(typeQueue, elemType)
					}
				}
			}

			if fieldType.Kind() == reflect.Map {
				valueType := fieldType.Elem()
				if valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
					valueTypeName := valueType.Elem().Name()
					if valueTypeName != "" && !visited[valueTypeName] && !isOpaqueType(valueType.Elem()) {
						typeQueue = append(typeQueue, valueType.Elem())
					}
				} else if valueType.Kind() == reflect.Struct && !isOpaqueType(valueType) {
					valueTypeName := valueType.Name()
					if valueTypeName != "" && !visited[valueTypeName] {
						typeQueue = append(typeQueue, valueType)
					}
				}
			}
		}
	}

	var lines []string

	lines = append(lines, "struct AnyCodable {")
	lines = append(lines, "    let value: Any?")
	lines = append(lines, "")
	lines = append(lines, "    init(_ value: Any?) {")
	lines = append(lines, "        self.value = value")
	lines = append(lines, "    }")
	lines = append(lines, "}")
	lines = append(lines, "extension AnyCodable: Codable, @unchecked Sendable {")
	lines = append(lines, "    init(from decoder: Decoder) throws {")
	lines = append(lines, "        let container = try decoder.singleValueContainer()")
	lines = append(lines, "")
	lines = append(lines, "        if container.decodeNil() { value = nil }")
	lines = append(lines, "        else if let v = try? container.decode(Bool.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(Int.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(UInt.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(Int64.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(UInt64.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(Double.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode(String.self) { value = v }")
	lines = append(lines, "        else if let v = try? container.decode([String: AnyCodable].self) {")
	lines = append(lines, "            value = v.mapValues { $0.value }")
	lines = append(lines, "        }")
	lines = append(lines, "        else if let v = try? container.decode([AnyCodable].self) {")
	lines = append(lines, "            value = v.map { $0.value }")
	lines = append(lines, "        }")
	lines = append(lines, "        else {")
	lines = append(lines, "            throw DecodingError.dataCorruptedError(in: container, debugDescription: \"Unsupported JSON value\")")
	lines = append(lines, "        }")
	lines = append(lines, "    }")
	lines = append(lines, "")
	lines = append(lines, "    func encode(to encoder: Encoder) throws {")
	lines = append(lines, "        var container = encoder.singleValueContainer()")
	lines = append(lines, "")
	lines = append(lines, "        switch value {")
	lines = append(lines, "        case nil:")
	lines = append(lines, "            try container.encodeNil()")
	lines = append(lines, "        case let v as Bool:")
	lines = append(lines, "            try container.encode(v)")
	lines = append(lines, "        case let v as Int:")
	lines = append(lines, "            try container.encode(v)")
	lines = append(lines, "        case let v as Double:")
	lines = append(lines, "            try container.encode(v)")
	lines = append(lines, "        case let v as String:")
	lines = append(lines, "            try container.encode(v)")
	lines = append(lines, "        case let v as [String: Any]:")
	lines = append(lines, "            try container.encode(v.mapValues { AnyCodable($0) })")
	lines = append(lines, "        case let v as [Any]:")
	lines = append(lines, "            try container.encode(v.map { AnyCodable($0) })")
	lines = append(lines, "        default:")
	lines = append(lines, "            throw EncodingError.invalidValue(value as Any, .init(codingPath: container.codingPath, debugDescription: \"Unsupported JSON value\"))")
	lines = append(lines, "        }")
	lines = append(lines, "    }")
	lines = append(lines, "}")
	lines = append(lines, "")

	sort.Slice(typeOrder, func(i, j int) bool { return typeOrder[i].Name() < typeOrder[j].Name() })

	for _, structType := range typeOrder {
		lines = append(lines, generateSingleSwiftStruct(structType))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func generateSingleSwiftStruct(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	var lines []string
	lines = append(lines, "struct "+t.Name()+" {")

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		fieldType := field.Type

		typeStr := swiftTypeToString(fieldType)

		if jsonTag == "" || jsonTag == "-" {
			if isOpaqueType(fieldType) {
				lines = append(lines, fmt.Sprintf("    let %s: AnyCodable?", field.Name))
			} else {
				lines = append(lines, fmt.Sprintf("    let %s: %s?", field.Name, typeStr))
			}
			continue
		}

		tagValue := strings.Split(jsonTag, ",")[0]
		fieldName := tagValue

		if isOpaqueType(fieldType) {
			lines = append(lines, fmt.Sprintf("    let %s: AnyCodable?", fieldName))
			continue
		}

		optionalSuffix := ""
		if !strings.HasSuffix(typeStr, "?") {
			optionalSuffix = "?"
		}
		lines = append(lines, fmt.Sprintf("    let %s: %s%s", fieldName, typeStr, optionalSuffix))
	}

	lines = append(lines, "}")
	lines = append(lines, "")

	if needsSwiftCodingKeys(t) {
		lines = append(lines, "extension "+t.Name()+": Codable, Sendable {")
		lines = append(lines, "    private enum CodingKeys: String, CodingKey {")

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}

			tagValue := strings.Split(jsonTag, ",")[0]
			fieldName := tagValue

			lines = append(lines, fmt.Sprintf("        case %s = \"%s\"", fieldName, tagValue))
		}

		lines = append(lines, "    }")
		lines = append(lines, "}")
	} else {
		lines = append(lines, "extension "+t.Name()+": Codable, Sendable {}")
	}

	return strings.Join(lines, "\n")
}

func kotlinTypeToString(t reflect.Type) string {
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return "AnyCodable"
	}

	if t.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && t.Name() == "Duration" {
		return "Long"
	}

	switch t.Kind() {
	case reflect.Ptr:
		inner := t.Elem()
		if inner.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && inner.Name() == "Duration" {
			return "Long?"
		}
		return kotlinTypeToString(inner) + "?"
	case reflect.Slice:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "List<AnyCodable>"
		}
		elem := t.Elem()
		if elem.PkgPath() == "github.com/xtls/xray-core/infra/conf/cfgcommon/duration" && elem.Name() == "Duration" {
			return "List<Long>"
		}
		return "List<" + kotlinTypeToString(elem) + ">"
	case reflect.Map:
		if t.Elem().PkgPath() == "encoding/json" && t.Elem().Name() == "RawMessage" {
			return "Map<String, AnyCodable>"
		}
		return "Map<" + kotlinTypeToString(t.Key()) + ", " + kotlinTypeToString(t.Elem()) + ">"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "Long"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "Long"
	case reflect.Float32, reflect.Float64:
		return "Double"
	case reflect.Bool:
		return "Boolean"
	case reflect.String:
		return "String"
	case reflect.Interface:
		return "Any?"
	default:
		return t.Name()
	}
}

func generateKotlinStructCode(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	typeQueue := []reflect.Type{t}
	typeOrder := []reflect.Type{}
	visited := make(map[string]bool)

	for len(typeQueue) > 0 {
		current := typeQueue[0]
		typeQueue = typeQueue[1:]

		typeName := current.Name()
		if typeName == "" {
			continue
		}
		if visited[typeName] {
			continue
		}
		visited[typeName] = true
		typeOrder = append(typeOrder, current)

		for i := 0; i < current.NumField(); i++ {
			field := current.Field(i)
			fieldType := field.Type

			var fieldTypeName string
			if fieldType.Kind() == reflect.Ptr {
				fieldTypeName = fieldType.Elem().Name()
			} else {
				fieldTypeName = fieldType.Name()
			}

			if fieldType.Kind() == reflect.Ptr {
				if fieldType.Elem().Kind() == reflect.Struct {
					elemType := fieldType.Elem()
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType) {
						typeQueue = append(typeQueue, elemType)
					}
				}
				continue
			}

			if fieldTypeName != "" && !visited[fieldTypeName] {
				if fieldType.Kind() == reflect.Struct && !isOpaqueType(fieldType) {
					typeQueue = append(typeQueue, fieldType)
				}
			}

			if fieldType.Kind() == reflect.Slice {
				elemType := fieldType.Elem()
				if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
					elemTypeName := elemType.Elem().Name()
					if elemTypeName != "" && !visited[elemTypeName] && !isOpaqueType(elemType.Elem()) {
						typeQueue = append(typeQueue, elemType.Elem())
					}
				} else if elemType.Kind() == reflect.Struct && !isOpaqueType(elemType) {
					elemTypeName := elemType.Name()
					if elemTypeName != "" && !visited[elemTypeName] {
						typeQueue = append(typeQueue, elemType)
					}
				}
			}

			if fieldType.Kind() == reflect.Map {
				valueType := fieldType.Elem()
				if valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
					valueTypeName := valueType.Elem().Name()
					if valueTypeName != "" && !visited[valueTypeName] && !isOpaqueType(valueType.Elem()) {
						typeQueue = append(typeQueue, valueType.Elem())
					}
				} else if valueType.Kind() == reflect.Struct && !isOpaqueType(valueType) {
					valueTypeName := valueType.Name()
					if valueTypeName != "" && !visited[valueTypeName] {
						typeQueue = append(typeQueue, valueType)
					}
				}
			}
		}
	}

	var lines []string

	lines = append(lines, "@Serializable(with = AnyCodable.Companion.AnyCodableSerializer::class)")
	lines = append(lines, "class AnyCodable(val value: Any?) {")
	lines = append(lines, "    companion object {")
	lines = append(lines, "        object AnyCodableSerializer : KSerializer<AnyCodable> {")
	lines = append(lines, "")
	lines = append(lines, "            @OptIn(InternalSerializationApi::class)")
	lines = append(lines, "            override val descriptor: SerialDescriptor =")
	lines = append(lines, "                buildSerialDescriptor(\"AnyCodable\", SerialKind.CONTEXTUAL)")
	lines = append(lines, "")
	lines = append(lines, "            override fun deserialize(decoder: Decoder): AnyCodable {")
	lines = append(lines, "                val input = decoder as? JsonDecoder")
	lines = append(lines, "                    ?: throw SerializationException(\"AnyCodable works only with JSON\")")
	lines = append(lines, "")
	lines = append(lines, "                val element = input.decodeJsonElement()")
	lines = append(lines, "                return AnyCodable(decodeElement(element))")
	lines = append(lines, "            }")
	lines = append(lines, "")
	lines = append(lines, "            private fun decodeElement(element: JsonElement): Any? {")
	lines = append(lines, "                return when (element) {")
	lines = append(lines, "")
	lines = append(lines, "                    JsonNull -> null")
	lines = append(lines, "")
	lines = append(lines, "                    is JsonPrimitive -> {")
	lines = append(lines, "                        when {")
	lines = append(lines, "                            element.isString -> element.content")
	lines = append(lines, "                            element.booleanOrNull != null -> element.boolean")
	lines = append(lines, "                            element.longOrNull != null -> element.long")
	lines = append(lines, "                            element.doubleOrNull != null -> element.double")
	lines = append(lines, "                            else -> element.content")
	lines = append(lines, "                        }")
	lines = append(lines, "                    }")
	lines = append(lines, "")
	lines = append(lines, "                    is JsonArray -> element.map { decodeElement(it) }")
	lines = append(lines, "")
	lines = append(lines, "                    is JsonObject -> element.mapValues { decodeElement(it.value) }")
	lines = append(lines, "                }")
	lines = append(lines, "            }")
	lines = append(lines, "")
	lines = append(lines, "            override fun serialize(encoder: Encoder, value: AnyCodable) {")
	lines = append(lines, "                val output = encoder as? JsonEncoder")
	lines = append(lines, "                    ?: throw SerializationException(\"AnyCodable works only with JSON\")")
	lines = append(lines, "")
	lines = append(lines, "                output.encodeJsonElement(encodeElement(value.value))")
	lines = append(lines, "            }")
	lines = append(lines, "")
	lines = append(lines, "            private fun encodeElement(value: Any?): JsonElement {")
	lines = append(lines, "                return when (value) {")
	lines = append(lines, "")
	lines = append(lines, "                    null -> JsonNull")
	lines = append(lines, "")
	lines = append(lines, "                    is Short -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is UShort -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is Long -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is ULong -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is Boolean -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is Int -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is Double -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is String -> JsonPrimitive(value)")
	lines = append(lines, "")
	lines = append(lines, "                    is Map<*, *> -> JsonObject(")
	lines = append(lines, "                        value.entries.associate {")
	lines = append(lines, "                            val key = it.key as? String")
	lines = append(lines, "                                ?: throw SerializationException(\"Map keys must be String\")")
	lines = append(lines, "                            key to encodeElement(it.value)")
	lines = append(lines, "                        }")
	lines = append(lines, "                    )")
	lines = append(lines, "")
	lines = append(lines, "                    is List<*> -> JsonArray(")
	lines = append(lines, "                        value.map { encodeElement(it) }")
	lines = append(lines, "                    )")
	lines = append(lines, "")
	lines = append(lines, "                    else -> throw SerializationException(\"Unsupported JSON value: ${value::class}\")")
	lines = append(lines, "                }")
	lines = append(lines, "            }")
	lines = append(lines, "        }")
	lines = append(lines, "    }")
	lines = append(lines, "}")
	lines = append(lines, "")

	sort.Slice(typeOrder, func(i, j int) bool { return typeOrder[i].Name() < typeOrder[j].Name() })

	for _, structType := range typeOrder {
		lines = append(lines, generateSingleKotlinStruct(structType))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func generateSingleKotlinStruct(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	var lines []string
	lines = append(lines, "@Serializable")
	lines = append(lines, "data class "+t.Name()+"(")

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		fieldType := field.Type

		typeStr := kotlinTypeToString(fieldType)

		if jsonTag == "" || jsonTag == "-" {
			if isOpaqueType(fieldType) {
				lines = append(lines, fmt.Sprintf("    val %s: AnyCodable?", field.Name))
			} else {
				lines = append(lines, fmt.Sprintf("    val %s: %s?", field.Name, typeStr))
			}
			continue
		}

		tagValue := strings.Split(jsonTag, ",")[0]
		fieldName := tagValue

		if isOpaqueType(fieldType) {
			lines = append(lines, fmt.Sprintf("    val %s: AnyCodable?", fieldName))
			continue
		}

		optionalSuffix := ""
		if !strings.HasSuffix(typeStr, "?") {
			optionalSuffix = "?"
		}
		lines = append(lines, fmt.Sprintf("    val %s: %s%s", fieldName, typeStr, optionalSuffix))
	}

	lines = append(lines, ")")

	return strings.Join(lines, "\n")
}
