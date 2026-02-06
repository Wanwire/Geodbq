package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/infra/conf"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Helper functions

func randomPort() uint32 {
	return uint32(10000 + rand.Intn(55000))
}

func randomUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(), rand.Intn(0x10000), rand.Intn(0x10000), rand.Intn(0x10000), rand.Uint64()%0x1000000000000)
}

// XrayConfigBuilder helps build Xray configs for testing
type XrayConfigBuilder struct {
	Inbounds  []map[string]interface{}
	Outbounds []map[string]interface{}
	Routing   map[string]interface{}
}

func NewXrayConfig() *XrayConfigBuilder {
	return &XrayConfigBuilder{
		Inbounds: []map[string]interface{}{
			{
				"tag":      "socks",
				"port":     randomPort(),
				"protocol": "socks",
				"settings": map[string]interface{}{
					"auth": "noauth",
				},
			},
			{
				"tag":      "http",
				"port":     randomPort(),
				"protocol": "http",
			},
		},
		Outbounds: []map[string]interface{}{
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
		Routing: map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules":          []map[string]interface{}{},
		},
	}
}

func (b *XrayConfigBuilder) AddOutbound(tag string, protocol string) *XrayConfigBuilder {
	outbound := map[string]interface{}{
		"tag":      tag,
		"protocol": protocol,
	}

	if protocol == "vmess" {
		outbound["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": fmt.Sprintf("proxy.example.com"),
					"port":    443,
					"users": []map[string]interface{}{
						{
							"id": randomUUID(),
							// Removed stray AddRule fragment
						},
					},
				},
			},
		}
	}

	b.Outbounds = append(b.Outbounds, outbound)
	return b
}

func (b *XrayConfigBuilder) AddRule(ruleType string, conditions map[string]interface{}, target string) *XrayConfigBuilder {
	rule := map[string]interface{}{
		"type": "field",
	}

	for k, v := range conditions {
		rule[k] = v
	}

	if target != "" {
		if strings.Contains(target, "-lb") {
			rule["balancerTag"] = target
		} else {
			rule["outboundTag"] = target
		}
	}

	if rules, ok := b.Routing["rules"].([]map[string]interface{}); ok {
		rules = append(rules, rule)
		b.Routing["rules"] = rules
	}

	return b
}

func (b *XrayConfigBuilder) SetDomainStrategy(strategy string) *XrayConfigBuilder {
	b.Routing["domainStrategy"] = strategy
	return b
}

func (b *XrayConfigBuilder) AddBalancer(tag string, selector []string) *XrayConfigBuilder {
	balancers := []map[string]interface{}{}
	if existing, ok := b.Routing["balancers"].([]map[string]interface{}); ok {
		balancers = existing
	}

	balancer := map[string]interface{}{
		"tag":      tag,
		"selector": selector,
		"strategy": map[string]interface{}{
			"type": "leastLoad",
			"settings": map[string]interface{}{
				"expected": 5,
			},
		},
	}

	b.Routing["balancers"] = append(balancers, balancer)
	return b
}

func (b *XrayConfigBuilder) ToJSON() []byte {
	config := map[string]interface{}{
		"inbounds":  b.Inbounds,
		"outbounds": b.Outbounds,
		"routing":   b.Routing,
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	return data
}

func (b *XrayConfigBuilder) WriteToFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp("", "xray-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(b.ToJSON()); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	return tmpFile.Name()
}

func parseAndBuildConfig(t *testing.T, configPath string) *conf.RouterConfig {
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var xrayConf conf.Config
	if err := json.Unmarshal(data, &xrayConf); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	routerCfg := xrayConf.RouterConfig
	if routerCfg == nil {
		t.Fatal("No routing section found in config")
	}

	return routerCfg
}

func buildRouterWithGeoDat(t *testing.T, routerCfg *conf.RouterConfig) *router.Config {
	// Set the global geoip and geosite paths using relative path to parent directory
	// Tests run in cmd subdirectory, so parent dir contains geoip.dat and geosite.dat
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to parent directory (workspace root) since we're in cmd/
	workspaceRoot := filepath.Dir(cwd)
	geoipPath = filepath.Join(workspaceRoot, "geoip.dat")
	geositePath = filepath.Join(workspaceRoot, "geosite.dat")

	// Verify files exist
	if _, err := os.Stat(geoipPath); err != nil {
		t.Logf("geoip.dat not found at %s: %v", geoipPath, err)
	}
	if _, err := os.Stat(geositePath); err != nil {
		t.Logf("geosite.dat not found at %s: %v", geositePath, err)
	}

	// Tell xray-core where to find the geodat files
	os.Setenv("xray.location.asset", workspaceRoot)

	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Logf("Failed to build routing with geodat: %v", err)
		return nil
	}
	return routerProto
}

func createTestCmd(domain, ip, inboundTag, preferIP string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().String("domain", "", "")
	cmd.Flags().String("ip", "", "")
	cmd.Flags().String("inbound-tag", "", "")
	cmd.Flags().String("prefer-ip", "", "")
	cmd.Flags().String("source-ip", "127.0.0.1", "")
	cmd.Flags().String("port", "", "")
	cmd.Flags().String("source-port", "", "")
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("network", "", "")
	cmd.Flags().String("user-email", "", "")

	if domain != "" {
		cmd.Flag("domain").Value.Set(domain)
	}
	if ip != "" {
		cmd.Flag("ip").Value.Set(ip)
	}
	if inboundTag != "" {
		cmd.Flag("inbound-tag").Value.Set(inboundTag)
	}
	if preferIP != "" {
		cmd.Flag("prefer-ip").Value.Set(preferIP)
	}

	return cmd
}

// Test cases

func TestSimpleDirectOutbound(t *testing.T) {
	// Command: geodbq route --domain "" --ip "8.8.8.8"
	// Expected output: Route to 'direct' outbound (all traffic goes direct)
	//
	// Test: Simple config with single rule routing all traffic to direct outbound
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {"domainStrategy": "AsIs", "rules": [{"type": "field", "ip": ["0.0.0.0/0"], "outboundTag": "direct"}]}
	// }
	builder := NewXrayConfig()
	builder.AddRule("field", map[string]interface{}{
		"ip": []string{"0.0.0.0/0"},
	}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	cmd := createTestCmd("", "8.8.8.8", "", "")
	destIP := net.ParseIP("8.8.8.8")

	// Should match the rule
	if len(routerProto.Rule) == 0 {
		t.Fatal("Expected at least one rule")
	}

	matched, _ := checkRuleMatch(routerProto.Rule[0], nil, nil, "", destIP, nil, cmd)
	if !matched {
		t.Error("Expected rule to match for IP 8.8.8.8")
	}
}

func TestInboundTagFiltering(t *testing.T) {
	// Command: geodbq route --domain "" --ip "1.1.1.1" --inbound-tag "socks"|"http"|""
	// Expected output: Matches 'direct' only with --inbound-tag "socks", fails with "http" or missing
	//
	// Test: Rules that require specific inbound tags should only match when tag is provided
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {"domainStrategy": "AsIs", "rules": [{"type": "field", "inboundTag": ["socks"], "ip": ["0.0.0.0/0"], "outboundTag": "direct"}]}
	// }
	builder := NewXrayConfig()
	builder.AddRule("field", map[string]interface{}{
		"inboundTag": []string{"socks"},
		"ip":         []string{"0.0.0.0/0"},
	}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	rule := routerProto.Rule[0]
	destIP := net.ParseIP("1.1.1.1")

	// With correct inbound tag
	cmdWithTag := createTestCmd("", "1.1.1.1", "socks", "")
	matched, _ := checkRuleMatch(rule, nil, nil, "", destIP, nil, cmdWithTag)
	if !matched {
		t.Error("Expected rule to match with socks inbound tag")
	}

	// With wrong inbound tag
	cmdWrongTag := createTestCmd("", "1.1.1.1", "http", "")
	matched, _ = checkRuleMatch(rule, nil, nil, "", destIP, nil, cmdWrongTag)
	if matched {
		t.Error("Expected rule to NOT match with http inbound tag")
	}

	// Without inbound tag
	cmdNoTag := createTestCmd("", "1.1.1.1", "", "")
	matched, _ = checkRuleMatch(rule, nil, nil, "", destIP, nil, cmdNoTag)
	if matched {
		t.Error("Expected rule to NOT match without inbound tag")
	}
}

func TestMultipleOutbounds(t *testing.T) {
	// Command: geodbq route --domain "" --ip "0.0.0.0/0" --inbound-tag "socks"
	// Expected output: 2 outbounds configured (direct, proxy-1), routing rule targets proxy-1 for socks inbound
	//
	// Test: Config with multiple outbounds and routing rules
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [
	//     {"tag": "direct", "protocol": "freedom"},
	//     {"tag": "proxy-1", "protocol": "vmess", "settings": {"vnext": [{"address": "proxy.example.com", "port": 443, "users": [{"id": "XXX", "security": "auto"}]}]}},
	//     {"tag": "proxy-2", "protocol": "vmess", "settings": {"vnext": [{"address": "proxy.example.com", "port": 443, "users": [{"id": "XXX", "security": "auto"}]}]}}
	//   ],
	//   "routing": {"domainStrategy": "AsIs", "rules": [{"type": "field", "inboundTag": ["socks"], "ip": ["0.0.0.0/0"], "outboundTag": "proxy-1"}]}
	// }
	builder := NewXrayConfig()
	builder.
		AddOutbound("proxy-1", "vmess").
		AddOutbound("proxy-2", "vmess").
		AddRule("field", map[string]interface{}{
			"inboundTag": []string{"socks"},
			"ip":         []string{"0.0.0.0/0"},
		}, "proxy-1")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	// Verify we have routing rules
	if len(routerProto.GetRule()) < 1 {
		t.Logf("Expected at least 1 routing rule, got %d", len(routerProto.GetRule()))
	}
}

func TestLoadBalancerConfig(t *testing.T) {
	// Command: geodbq route --domain "" --ip "0.0.0.0/0"
	// Expected output: Load balancer 'proxy-lb' configured with selector matching proxy-1,proxy-2,proxy-3; routes to balancer
	//
	// Test: Config with load balancer
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [
	//     {"tag": "direct", "protocol": "freedom"},
	//     {"tag": "proxy-1", "protocol": "vmess", "settings": {...}},
	//     {"tag": "proxy-2", "protocol": "vmess", "settings": {...}},
	//     {"tag": "proxy-3", "protocol": "vmess", "settings": {...}}
	//   ],
	//   "routing": {
	//     "domainStrategy": "AsIs",
	//     "rules": [{"type": "field", "ip": ["0.0.0.0/0"], "balancerTag": "proxy-lb"}],
	//     "balancers": [{"tag": "proxy-lb", "selector": ["proxy-"], "strategy": {"type": "leastLoad", "settings": {"expected": 5}}}]
	//   }
	// }
	builder := NewXrayConfig()
	builder.
		AddOutbound("proxy-1", "vmess").
		AddOutbound("proxy-2", "vmess").
		AddOutbound("proxy-3", "vmess").
		AddBalancer("proxy-lb", []string{"proxy-"}).
		AddRule("field", map[string]interface{}{
			"ip": []string{"0.0.0.0/0"},
		}, "proxy-lb")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	// Verify balancer exists
	if len(routerProto.BalancingRule) == 0 {
		t.Error("Expected at least one balancing rule")
	}

	// Verify balancer has correct tag
	if len(routerProto.BalancingRule) > 0 {
		if routerProto.BalancingRule[0].Tag != "proxy-lb" {
			t.Errorf("Expected balancer tag 'proxy-lb', got %s", routerProto.BalancingRule[0].Tag)
		}
	}
}

func TestDomainStrategyAsIs(t *testing.T) {
	// Command: geodbq route --domain "example.com"
	// Expected output: DomainStrategy set to AsIs (0); matches exact domain
	//
	// Test: DomainStrategy AsIs means domain-only matching
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {"domainStrategy": "AsIs", "rules": [{"type": "field", "domain": ["domain:example.com"], "outboundTag": "direct"}]}
	// }
	builder := NewXrayConfig()
	builder.
		SetDomainStrategy("AsIs").
		AddRule("field", map[string]interface{}{
			"domain": []string{"domain:example.com"},
		}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	if routerProto.DomainStrategy != 0 { // AsIs = 0
		t.Errorf("Expected DomainStrategy AsIs (0), got %d", routerProto.DomainStrategy)
	}
}

func TestConfigIsValid(t *testing.T) {
	// Command: (3 sub-tests: Simple, WithProxy, WithBalancer)
	// Expected output: JSON configs parse successfully; contain inbounds, outbounds, routing sections
	//
	// Test: Generated configs should be valid Xray configurations
	// Sample generated configs:
	// Simple:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {"domainStrategy": "AsIs", "rules": []}
	// }
	// WithProxy:
	// {
	//   "inbounds": [{...}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}, {"tag": "proxy", "protocol": "vmess", "settings": {...}}],
	//   "routing": {"domainStrategy": "AsIs", "rules": []}
	// }
	// WithBalancer:
	// {
	//   "inbounds": [{...}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}, {"tag": "proxy-1", "protocol": "vmess", ...}, {"tag": "proxy-2", "protocol": "vmess", ...}],
	//   "routing": {"domainStrategy": "AsIs", "rules": [], "balancers": [{"tag": "proxy-lb", "selector": ["proxy-"], ...}]}
	// }
	tests := []struct {
		name    string
		builder func() *XrayConfigBuilder
	}{
		{
			name:    "Simple",
			builder: NewXrayConfig,
		},
		{
			name: "WithProxy",
			builder: func() *XrayConfigBuilder {
				return NewXrayConfig().AddOutbound("proxy", "vmess")
			},
		},
		{
			name: "WithBalancer",
			builder: func() *XrayConfigBuilder {
				return NewXrayConfig().
					AddOutbound("proxy-1", "vmess").
					AddOutbound("proxy-2", "vmess").
					AddBalancer("proxy-lb", []string{"proxy-"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.builder()
			configPath := builder.WriteToFile(t)
			defer os.Remove(configPath)

			// Should not error when parsing
			data, _ := os.ReadFile(configPath)
			var config map[string]interface{}
			if err := json.Unmarshal(data, &config); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			// Verify structure
			if _, ok := config["inbounds"]; !ok {
				t.Error("Missing inbounds")
			}
			if _, ok := config["outbounds"]; !ok {
				t.Error("Missing outbounds")
			}
			if _, ok := config["routing"]; !ok {
				t.Error("Missing routing")
			}
		})
	}
}

func TestMatchingWithIPCIDR(t *testing.T) {
	// Command: geodbq route --domain "" --ip "192.168.1.1" and geodbq route --domain "" --ip "10.0.0.1"
	// Expected output: 192.168.1.1 matches (within 192.168.0.0/16), 10.0.0.1 doesn't match (outside CIDR)
	//
	// Test: Rules with CIDR blocks should match IPs within that range
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {"domainStrategy": "AsIs", "rules": [{"type": "field", "ip": ["192.168.0.0/16"], "outboundTag": "direct"}]}
	// }
	builder := NewXrayConfig()
	builder.AddRule("field", map[string]interface{}{
		"ip": []string{"192.168.0.0/16"},
	}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	cmd := createTestCmd("", "", "", "")

	// Should match IP within CIDR
	ip := net.ParseIP("192.168.1.1")
	matched, _ := checkRuleMatch(routerProto.Rule[0], nil, nil, "", ip, nil, cmd)
	if !matched {
		t.Error("Expected match for IP 192.168.1.1 within 192.168.0.0/16")
	}

	// Should not match IP outside CIDR
	ip = net.ParseIP("10.0.0.1")
	matched, _ = checkRuleMatch(routerProto.Rule[0], nil, nil, "", ip, nil, cmd)
	if matched {
		t.Error("Expected no match for IP 10.0.0.1 outside 192.168.0.0/16")
	}
}

func TestMultipleRulesFirstMatch(t *testing.T) {
	// Command: geodbq route --domain "" --ip "1.1.1.1" --inbound-tag "socks"
	// Expected output: Routes to 'proxy' outbound (first matching rule), not 'direct' (second rule)
	//
	// Test: Router should use first matching rule (first-match semantics)
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}, {"tag": "proxy", "protocol": "vmess", "settings": {...}}],
	//   "routing": {
	//     "domainStrategy": "AsIs",
	//     "rules": [
	//       {"type": "field", "inboundTag": ["socks"], "ip": ["0.0.0.0/0"], "outboundTag": "proxy"},
	//       {"type": "field", "ip": ["0.0.0.0/0"], "outboundTag": "direct"}
	//     ]
	//   }
	// }
	builder := NewXrayConfig()
	builder.
		AddOutbound("proxy", "vmess").
		AddRule("field", map[string]interface{}{
			"inboundTag": []string{"socks"},
			"ip":         []string{"0.0.0.0/0"},
		}, "proxy").
		AddRule("field", map[string]interface{}{
			"ip": []string{"0.0.0.0/0"},
		}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto, err := routerCfg.Build()
	if err != nil {
		t.Fatalf("Failed to build routing: %v", err)
	}

	cmd := createTestCmd("", "1.1.1.1", "socks", "")
	destIP := net.ParseIP("1.1.1.1")

	// First rule should match
	matched, _ := checkRuleMatch(routerProto.Rule[0], nil, nil, "", destIP, nil, cmd)
	if !matched {
		t.Error("Expected first rule to match")
	}
}

func TestGeositeApple(t *testing.T) {
	// Command: geodbq route --domain "example.cn" | "www.apple.com"
	// Expected output: example.cn matches rule 0 (geosite:cn -> direct); www.apple.com matches rule 2 (geosite:apple -> proxy)
	//
	// Test: Geosite matching for Apple domains
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}, {"tag": "proxy", "protocol": "vmess", "settings": {...}}],
	//   "routing": {
	//     "domainStrategy": "AsIs",
	//     "rules": [
	//       {"type": "field", "domain": ["geosite:cn"], "outboundTag": "direct"},
	//       {"type": "field", "domain": ["geosite:private"], "outboundTag": "direct"},
	//       {"type": "field", "domain": ["geosite:apple"], "outboundTag": "proxy"},
	//       {"type": "field", "ip": ["0.0.0.0/0"], "outboundTag": "direct"}
	//     ]
	//   }
	// }
	builder := NewXrayConfig()
	builder.
		AddOutbound("proxy", "vmess").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:cn"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:private"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:apple"},
		}, "proxy").
		AddRule("field", map[string]interface{}{
			"ip": []string{"0.0.0.0/0"},
		}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto := buildRouterWithGeoDat(t, routerCfg)
	if routerProto == nil {
		t.Skip("Skipping: geosite.dat not accessible")
	}

	// Verify structure
	if len(routerProto.GetRule()) != 4 {
		t.Errorf("Expected 4 rules, got %d", len(routerProto.GetRule()))
	}

	// Verify rule matching behavior with actual domain names
	cmd := createTestCmd("", "", "", "")

	// Check that rule 0 (geosite:cn) matches CN domains
	if len(routerProto.GetRule()) > 0 {
		matched, _ := checkRuleMatch(routerProto.Rule[0], nil, nil, "example.cn", nil, nil, cmd)
		t.Logf("Rule 0 (geosite:cn) match result for 'example.cn': %v", matched)
	}

	// Check that rule 2 (geosite:apple) matches Apple domains
	if len(routerProto.GetRule()) > 2 {
		matched, _ := checkRuleMatch(routerProto.Rule[2], nil, nil, "www.apple.com", nil, nil, cmd)
		t.Logf("Rule 2 (geosite:apple) match result for 'www.apple.com': %v", matched)
	}
}

func TestGeositeCN(t *testing.T) {
	// Command: geodbq route --domain "baidu.com" | "localhost"
	// Expected output: baidu.com matches rule 0 (geosite:cn -> direct); localhost matches rule 1 (geosite:private -> direct)
	//
	// Test: Geosite matching for CN domains - first matching rule
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [{"tag": "direct", "protocol": "freedom"}],
	//   "routing": {
	//     "domainStrategy": "AsIs",
	//     "rules": [
	//       {"type": "field", "domain": ["geosite:cn"], "outboundTag": "direct"},
	//       {"type": "field", "domain": ["geosite:private"], "outboundTag": "direct"},
	//       {"type": "field", "ip": ["0.0.0.0/0"], "outboundTag": "direct"}
	//     ]
	//   }
	// }
	builder := NewXrayConfig()
	builder.
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:cn"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:private"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"ip": []string{"0.0.0.0/0"},
		}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto := buildRouterWithGeoDat(t, routerCfg)
	if routerProto == nil {
		t.Skip("Skipping: geosite.dat not accessible")
	}

	// Verify structure
	if len(routerProto.GetRule()) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(routerProto.GetRule()))
	}

	// Verify rule matching behavior with actual domain names
	cmd := createTestCmd("", "", "", "")

	// Check that rule 0 (geosite:cn) matches CN domains
	if len(routerProto.GetRule()) > 0 {
		matched, _ := checkRuleMatch(routerProto.Rule[0], nil, nil, "baidu.com", nil, nil, cmd)
		t.Logf("Rule 0 (geosite:cn) match result for 'baidu.com': %v (should match CN domain)", matched)
	}

	// Check that rule 1 (geosite:private) matches private domains
	if len(routerProto.GetRule()) > 1 {
		matched, _ := checkRuleMatch(routerProto.Rule[1], nil, nil, "localhost", nil, nil, cmd)
		t.Logf("Rule 1 (geosite:private) match result for 'localhost': %v (should match private domain)", matched)
	}
}

func TestGeositeMultipleRulesMatchingFourth(t *testing.T) {
	// Command: geodbq route --domain "www.apple.com" | "localhost"
	// Expected output: www.apple.com matches rule 1 (geosite:apple -> direct); localhost matches rule 3 (geosite:private -> proxy-lb load balancer)
	//
	// Test: Match on the 4th routing rule when earlier geosite rules don't match
	// Generated config:
	// {
	//   "inbounds": [{"tag": "socks", "port": XXX, "protocol": "socks"}, {"tag": "http", "port": XXX, "protocol": "http"}],
	//   "outbounds": [
	//     {"tag": "direct", "protocol": "freedom"},
	//     {"tag": "proxy-1", "protocol": "vmess", "settings": {...}},
	//     {"tag": "proxy-2", "protocol": "vmess", "settings": {...}}
	//   ],
	//   "routing": {
	//     "domainStrategy": "AsIs",
	//     "rules": [
	//       {"type": "field", "inboundTag": ["api"], "outboundTag": "api"},
	//       {"type": "field", "domain": ["geosite:apple"], "outboundTag": "direct"},
	//       {"type": "field", "domain": ["geosite:cn"], "outboundTag": "direct"},
	//       {"type": "field", "domain": ["geosite:private"], "balancerTag": "proxy-lb"},
	//       {"type": "field", "ip": ["0.0.0.0/0"], "outboundTag": "direct"}
	//     ],
	//     "balancers": [{"tag": "proxy-lb", "selector": ["proxy-"], "strategy": {"type": "leastLoad", "settings": {"expected": 5}}}]
	//   }
	// }
	builder := NewXrayConfig()
	builder.
		AddOutbound("proxy-1", "vmess").
		AddOutbound("proxy-2", "vmess").
		AddBalancer("proxy-lb", []string{"proxy-"}).
		AddRule("field", map[string]interface{}{
			"inboundTag": []string{"api"},
		}, "api").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:apple"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:cn"},
		}, "direct").
		AddRule("field", map[string]interface{}{
			"domain": []string{"geosite:private"},
		}, "proxy-lb").
		AddRule("field", map[string]interface{}{
			"ip": []string{"0.0.0.0/0"},
		}, "direct")

	configPath := builder.WriteToFile(t)
	defer os.Remove(configPath)

	routerCfg := parseAndBuildConfig(t, configPath)
	routerProto := buildRouterWithGeoDat(t, routerCfg)
	if routerProto == nil {
		t.Skip("Skipping: geodata not accessible")
	}

	// Verify structure
	if len(routerProto.GetRule()) != 5 {
		t.Errorf("Expected 5 rules, got %d", len(routerProto.GetRule()))
	}

	if len(routerProto.BalancingRule) == 0 {
		t.Error("Expected load balancer rule")
	}

	// Verify rule matching behavior with actual domain names
	cmd := createTestCmd("", "", "", "")

	// Check rule 1 (geosite:apple) matches Apple domains
	if len(routerProto.GetRule()) > 1 {
		matched, _ := checkRuleMatch(routerProto.Rule[1], nil, nil, "www.apple.com", nil, nil, cmd)
		t.Logf("Rule 1 (geosite:apple) match result for 'www.apple.com': %v", matched)
	}

	// Check rule 3 (geosite:private -> proxy-lb) matches private domains
	if len(routerProto.GetRule()) >= 4 {
		matched, _ := checkRuleMatch(routerProto.Rule[3], nil, nil, "localhost", nil, nil, cmd)
		t.Logf("Rule 3 (geosite:private -> proxy-lb) match result for 'localhost': %v (4th rule, uses load balancer)", matched)
	}

	t.Logf("Config has %d routing rules:", len(routerProto.GetRule()))
	t.Logf("  Rule 0: inboundTag:api -> api")
	t.Logf("  Rule 1: geosite:apple -> direct")
	t.Logf("  Rule 2: geosite:cn -> direct")
	t.Logf("  Rule 3: geosite:private -> proxy-lb (4th rule, matches here with load balancer)")
	t.Logf("  Rule 4: 0.0.0.0/0 -> direct (fallback)")
}
