// internal/geo/geo.go
package geo

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

// ─────────────────────────────────────────────────────────────────────────────
// Loading
// ─────────────────────────────────────────────────────────────────────────────

func LoadGeoIP(path string) (*router.GeoIPList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read geoip.dat: %w", err)
	}
	list := &router.GeoIPList{}
	if err := proto.Unmarshal(data, list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geoip: %w", err)
	}
	return list, nil
}

func LoadGeoSite(path string) (*router.GeoSiteList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read geosite.dat: %w", err)
	}
	list := &router.GeoSiteList{}
	if err := proto.Unmarshal(data, list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geosite: %w", err)
	}
	return list, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Query functions
// ─────────────────────────────────────────────────────────────────────────────

func QueryGeoIP(list *router.GeoIPList, ipStr string) []string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}

	codes := make(map[string]struct{})
	for _, entry := range list.Entry {
		for _, cidr := range entry.Cidr {
			prefix := int(cidr.Prefix)
			ipBytes := cidr.Ip
			if len(ipBytes) == 4 && prefix > 32 {
				prefix = 32
			} else if len(ipBytes) == 16 && prefix > 128 {
				prefix = 128
			}
			_, network, err := net.ParseCIDR(fmt.Sprintf("%s/%d", net.IP(ipBytes).String(), prefix))
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				codes[strings.ToUpper(entry.CountryCode)] = struct{}{}
			}
		}
	}
	var result []string
	for c := range codes {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

func QueryGeoSite(list *router.GeoSiteList, domainStr string) []string {
	query := strings.ToLower(domainStr)
	codes := make(map[string]struct{})

	for _, entry := range list.Entry {
		matched := false
		for _, d := range entry.Domain {
			if matchDomain(d, query) {
				matched = true
				break
			}
		}
		if matched {
			codes[entry.CountryCode] = struct{}{}
		}
	}

	var result []string
	for c := range codes {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

func matchDomain(d *router.Domain, query string) bool {
	v := strings.ToLower(d.Value)
	switch d.Type {
	case router.Domain_Plain:
		return strings.Contains(query, v)
	case router.Domain_Regex:
		r, err := regexp.Compile(v)
		if err != nil {
			return false
		}
		return r.MatchString(query)
	case router.Domain_Domain:
		return query == v || strings.HasSuffix(query, "."+v)
	case router.Domain_Full:
		return query == v
	default:
		return false
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// List categories
// ─────────────────────────────────────────────────────────────────────────────

func ListGeoIPCategories(list *router.GeoIPList) []string {
	m := make(map[string]struct{})
	for _, e := range list.Entry {
		if e.CountryCode != "" {
			m[strings.ToUpper(e.CountryCode)] = struct{}{}
		}
	}
	var codes []string
	for c := range m {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

func ListGeoSiteCategories(list *router.GeoSiteList) []string {
	m := make(map[string]struct{})
	for _, e := range list.Entry {
		if e.CountryCode != "" {
			m[e.CountryCode] = struct{}{}
		}
	}
	var codes []string
	for c := range m {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// ─────────────────────────────────────────────────────────────────────────────
// Summarize
// ─────────────────────────────────────────────────────────────────────────────

func SummarizeGeoIP(list *router.GeoIPList) {
	countries := make(map[string]int)
	totalCIDRs := 0

	for _, e := range list.Entry {
		cc := strings.ToUpper(e.CountryCode)
		countries[cc]++
		totalCIDRs += len(e.Cidr)
	}

	fmt.Printf("Total entries: %d\n", len(list.Entry))
	fmt.Printf("Total CIDR blocks: %d\n", totalCIDRs)
	fmt.Printf("Unique countries: %d\n", len(countries))

	fmt.Println("\nTop 15 countries by entry count:")
	var pairs []struct {
		Code  string
		Count int
	}
	for code, cnt := range countries {
		pairs = append(pairs, struct {
			Code  string
			Count int
		}{code, cnt})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Count > pairs[j].Count })
	for i, p := range pairs {
		if i >= 15 {
			break
		}
		fmt.Printf("  %4s : %5d\n", p.Code, p.Count)
	}
}

func SummarizeGeoSite(list *router.GeoSiteList) {
	categories := make(map[string]int)
	totalRules := 0

	for _, e := range list.Entry {
		cc := e.CountryCode
		categories[cc]++
		totalRules += len(e.Domain)
	}

	fmt.Printf("Total categories: %d\n", len(list.Entry))
	fmt.Printf("Total domain rules: %d\n", totalRules)
	fmt.Printf("Unique categories: %d\n", len(categories))

	fmt.Println("\nTop 15 categories by rule count:")
	var pairs []struct {
		Code  string
		Count int
	}
	for code, cnt := range categories {
		pairs = append(pairs, struct {
			Code  string
			Count int
		}{code, cnt})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Count > pairs[j].Count })
	for i, p := range pairs {
		if i >= 15 {
			break
		}
		fmt.Printf("  %-30s : %5d\n", p.Code, p.Count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// List rules for specific category
// ─────────────────────────────────────────────────────────────────────────────

func ListGeoIPRules(list *router.GeoIPList, code string, maxShow int) bool {
	code = strings.ToUpper(code)
	for _, entry := range list.Entry {
		if strings.ToUpper(entry.CountryCode) == code {
			fmt.Printf("Country: %s\n", entry.CountryCode)
			fmt.Printf("Total CIDRs: %d\n", len(entry.Cidr))

			show := len(entry.Cidr)
			if maxShow >= 0 && maxShow < show {
				show = maxShow
			}

			for i, cidr := range entry.Cidr[:show] {
				ip := net.IP(cidr.Ip).String()
				fmt.Printf("  %4d. %s/%d\n", i+1, ip, cidr.Prefix)
			}
			if show < len(entry.Cidr) {
				fmt.Printf("  ... (+ %d more)\n", len(entry.Cidr)-show)
			}
			return true
		}
	}
	return false
}

func ListGeoSiteRules(list *router.GeoSiteList, category string, maxShow int) {
	for _, entry := range list.Entry {
		if strings.EqualFold(entry.CountryCode, category) {
			fmt.Printf("Category: %s\n", entry.CountryCode)
			fmt.Printf("Total domain rules: %d\n", len(entry.Domain))

			show := len(entry.Domain)
			if maxShow >= 0 && maxShow < show {
				show = maxShow
			}

			typeNames := map[router.Domain_Type]string{
				router.Domain_Plain:  "plain",
				router.Domain_Regex:  "regex",
				router.Domain_Domain: "domain",
				router.Domain_Full:   "full",
			}

			for i, d := range entry.Domain[:show] {
				tname := typeNames[d.Type]
				if tname == "" {
					tname = fmt.Sprintf("type-%d", d.Type)
				}
				fmt.Printf("  %4d. [%s] %s\n", i+1, tname, d.Value)
			}
			if show < len(entry.Domain) {
				fmt.Printf("  ... (+ %d more)\n", len(entry.Domain)-show)
			}
			return
		}
	}
	fmt.Printf("Category '%s' not found in geosite.dat\n", category)
}

// ─────────────────────────────────────────────────────────────────────────────
// Simulation
// ─────────────────────────────────────────────────────────────────────────────
func DomainTypeName(t router.Domain_Type) string {
	switch t {
	case router.Domain_Plain:
		return "plain"
	case router.Domain_Regex:
		return "regex"
	case router.Domain_Domain:
		return "domain"
	case router.Domain_Full:
		return "full"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func MatchDomain(d *router.Domain, query string) bool {
	v := strings.ToLower(d.Value)
	q := strings.ToLower(query)
	switch d.Type {
	case router.Domain_Plain:
		return strings.Contains(q, v)
	case router.Domain_Regex:
		r, err := regexp.Compile(v)
		if err != nil {
			return false
		}
		return r.MatchString(q)
	case router.Domain_Domain:
		return q == v || strings.HasSuffix(q, "."+v)
	case router.Domain_Full:
		return q == v
	default:
		return false
	}
}

func FindGeoSiteEntry(list *router.GeoSiteList, code string) *router.GeoSite {
	for _, e := range list.Entry {
		if strings.EqualFold(e.CountryCode, code) {
			return e
		}
	}
	return nil
}
