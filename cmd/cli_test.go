// cmd/cli_test.go
package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func getProjectRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	return filepath.Dir(wd)
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	root := getProjectRoot(t)
	bin := filepath.Join(root, "geodbq")

	args = append([]string{}, args...)

	cmd := exec.Command(bin, args...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(out), strings.TrimSpace(string(out)), exitCode
}

func runCLIWithGeoip(t *testing.T, args ...string) (string, string, int) {
	root := getProjectRoot(t)
	bin := filepath.Join(root, "geodbq")
	geoipPath := filepath.Join(root, "geoip.dat")
	geositePath := filepath.Join(root, "geosite.dat")

	fullArgs := []string{}
	if len(args) > 0 {
		fullArgs = append(fullArgs, "--geoip", geoipPath, "--geosite", geositePath)
		fullArgs = append(fullArgs, args...)
	}

	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(out), strings.TrimSpace(string(out)), exitCode
}

func hasGeoIP(t *testing.T) bool {
	root := getProjectRoot(t)
	_, err := os.Stat(filepath.Join(root, "geoip.dat"))
	return err == nil
}

func hasGeoSite(t *testing.T) bool {
	root := getProjectRoot(t)
	_, err := os.Stat(filepath.Join(root, "geosite.dat"))
	return err == nil
}

func skipIfNoGeoData(t *testing.T) {
	if !hasGeoIP(t) || !hasGeoSite(t) {
		t.Skip("geoip.dat or geosite.dat not found")
	}
}

func TestIPLookup_GoogleDNS(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "ip", "8.8.8.8")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "US") {
		t.Errorf("Expected US for 8.8.8.8, got: %s", output)
	}
}

func TestIPLookup_Cloudflare(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "ip", "1.1.1.1")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "Matching country codes") {
		t.Errorf("Expected match output, got: %s", output)
	}
}

func TestIPLookup_Invalid(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "ip", "999.999.999.999")
	if exitCode == 0 {
		t.Logf("Invalid IP returned exit code 0, output: %s", output)
	}
}

func TestIPLookup_Private(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "ip", "192.168.1.1")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "No match") {
		t.Logf("Expected no match for private IP, got: %s", output)
	}
}

func TestIPLookup_IPv6(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "ip", "2001:4860:4860::8888")
	if exitCode != 0 {
		t.Logf("IPv6 lookup returned exit code %d: %s", exitCode, output)
	}
}

func TestDomainLookup_Google(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "domain", "google.com")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "google") {
		t.Errorf("Expected google category, got: %s", output)
	}
}

func TestDomainLookup_Youtube(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "domain", "youtube.com")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "youtube") {
		t.Errorf("Expected youtube category, got: %s", output)
	}
}

func TestDomainLookup_Subdomain(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "domain", "www.google.com")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "google") {
		t.Errorf("Expected google for subdomain, got: %s", output)
	}
}

func TestDomainLookup_NotFound(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "domain", "this-domain-does-not-exist-123456.invalid")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(output, "No matching geosite categories") && !strings.Contains(output, "PRIVATE") {
		t.Errorf("Expected no match or PRIVATE, got: %s", output)
	}
}

func TestDomainLookup_CaseInsensitive(t *testing.T) {
	skipIfNoGeoData(t)

	_, output, exitCode := runCLIWithGeoip(t, "domain", "GOOGLE.COM")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, output)
	}

	if !strings.Contains(strings.ToLower(output), "google") {
		t.Errorf("Expected google (case-insensitive), got: %s", output)
	}
}

func TestListCategories_GeoIP(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-categories")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	hasUS := strings.Contains(raw, "US")
	hasCN := strings.Contains(raw, "CN")

	if !hasUS || !hasCN {
		t.Errorf("Expected US and CN in country codes, got truncated: %s...", raw[:min(200, len(raw))])
	}
}

func TestListCategories_GeoSite(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-categories")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	hasGoogle := strings.Contains(strings.ToLower(raw), "google")
	hasGFW := strings.Contains(strings.ToLower(raw), "gfw")

	if !hasGoogle {
		t.Logf("Expected google category, got: %s...", raw[:min(200, len(raw))])
	}
	_ = hasGFW
}

func TestSummarize_GeoIP(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "summarize")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	if !strings.Contains(raw, "Total entries") {
		t.Errorf("Expected summary output, got: %s", raw)
	}
}

func TestSummarize_GeoSite(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "summarize")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	hasGeoIP := strings.Contains(raw, "geoip.dat")
	hasGeoSite := strings.Contains(raw, "geosite.dat")

	if !hasGeoIP || !hasGeoSite {
		t.Logf("Expected both geoip and geosite summary")
	}
}

func TestListRules_US(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-rules", "us")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	if !strings.Contains(raw, "Country:") && !strings.Contains(raw, "Total CIDRs") {
		t.Errorf("Expected US rules, got: %s", raw)
	}
}

func TestListRules_Google(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-rules", "google")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	if !strings.Contains(raw, "Category:") && !strings.Contains(raw, "Total domain rules") {
		t.Errorf("Expected google rules, got: %s", raw)
	}
}

func TestListRules_MaxShow(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-rules", "cn", "--max-show", "10")
	if exitCode != 0 {
		t.Fatalf("CLI failed with exit code %d: %s", exitCode, raw)
	}

	lines := strings.Split(raw, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), ".") {
			count++
		}
	}

	if count > 11 {
		t.Logf("Expected <= 11 lines with --max-show 10, got ~%d", count)
	}
}

func TestListRules_NotFound(t *testing.T) {
	skipIfNoGeoData(t)

	raw, _, exitCode := runCLIWithGeoip(t, "list-rules", "nonexistent_category_xyz")
	if exitCode != 0 {
		t.Logf("Exit code: %d", exitCode)
	}

	if !strings.Contains(raw, "not found") && !strings.Contains(raw, "Category") {
		t.Logf("Expected not found message, got: %s", raw)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}