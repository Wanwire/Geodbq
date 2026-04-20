# geodbq

A lightweight command-line tool to inspect and query `geoip.dat` and `geosite.dat` files used by **Xray-core**.

It allows you to:

- Look up country codes for IP addresses
- Find matching geosite categories for domains
- List all available country codes and categories
- Summarize file contents (entry count, rule count, top categories)
- List the actual CIDRs or domain rules for any specific country/category

## Features

- Case-insensitive domain matching
- Proper prefix clamping for invalid CIDR entries
- Real Xray-style domain matching (`plain`, `regex`, `domain`, `full`)
- Clean, sorted output
- Optional limit on number of rules shown (`--max-show`)

## Requirements

- Go 1.26 or newer

## Installation

```bash
# Clone the repository
git clone https://github.com/wanwire/geodbq.git
cd geodbq

# Download dependencies
go mod tidy

# Build the binary
go build -o geodbq
```

Or install directly from source:
```bash
go install github.com/wanwire/geodbq@latest
```

## Usage
```text
geodbq [flags] [command]

Commands:
  domain          Query matching geosite categories for a domain
  extract         Generate Go/Swift/Kotlin struct code from Xray conf.Config
  help            Help about any command
  ip              Query country code(s) for an IP
  list-categories List all available country codes / categories
  list-rules      List rules (CIDRs or domains) for a specific country code / category
  simulate-route  Simulate Xray routing decision for a domain or IP using config.json
  summarize       Show summary statistics for geoip.dat and/or geosite.dat

Flags:
      --geoip string     Path to geoip.dat (default: ./geoip.dat)
      --geosite string   Path to geosite.dat (default: ./geosite.dat)
  -h, --help             help for geodbq

list-rules flags:
      --max-show int    Maximum number of items to show (-1 = all)

simulate-route flags:
      --config string       Path to Xray config.json (default: config.json)
      --domain string      Simulate traffic destined for this domain
      --ip string         Simulate traffic destined to this IP address
      --source-ip string   Simulate traffic originating from this IP (default: 127.0.0.1)
      --prefer-ip string   Prefer IPv4 or IPv6 for DNS resolution
      --inbound-tag string Simulate traffic from this inbound tag
      --port string       Destination port
      --protocol string   Protocol (e.g. bittorrent, http)
      --network string     Network (tcp, udp)

extract flags:
      --lang string      Output language: golang, go, swift, s, kotlin, k (default: golang)
```

## Examples
1. Query an IP

```bash
geodbq ip 8.8.8.8
```
or with custom file

```bash
geodbq --geoip ./geoip.dat ip 1.1.1.1
```

2. Query a domain

```bash
geodbq domain google.com
geodbq domain youtube.com
```

3. List all categories / country codes

```bash
geodbq list-categories
```

Typical output:

```text
Available geoip country codes (about 250):
  AD, AE, AF, AG, AI, AL, AM, AO, AQ, AR, AS, AT, AU, AW, AX, AZ, BA, BB, BD, BE, ...

Available geosite categories (500–2000+ depending on file):
  apple, category-ads-all, cn, discord, facebook, gfw, google, instagram, microsoft, netflix, ...
```

4. Summarize file contents

```bash
geodbq summarize
```

Example output:

```text
=== geoip.dat summary ===
Total entries: 285
Total CIDR blocks: 12473
Unique countries: 285

Top 15 countries by entry count:
    CN :  4123
    US :  2189
    RU :   987
  ...

=== geosite.dat summary ===
Total categories: 1247
Total domain rules: 45892
Unique categories: 1247

Top 15 categories by rule count:
  geolocation-!cn                :  8921
  cn                             :  7456
  gfw                            :  3124
  google                         :  1842
  ...
```

5. List rules for a category
Show all rules:

```bash
geodbq list-rules google
geodbq list-rules cn
```

Limit output (first 50 items):

```bash
geodbq list-rules google --max-show 50
geodbq list-rules us --max-show 20
```

Example output for list-rules google:

```text
Category: google
Total domain rules: 1842
   1. [domain] google.com
   2. [domain] googlevideo.com
   3. [domain] ggpht.com
   4. [domain] youtube.com
   ...
  ... (+ 1792 more)
```

6. Simulate Xray-core
Simulate how Xray-core would route a given domain or IP based on your config.json file. Shows the matching rule, specific geoip/geosite condition that matched, and the resulting outbound or balancer candidates.

```bash
# Simulate traffic destined for a domain
geodbq simulate-route --config config.json --domain google.com

# Simulate traffic destined to an IP address
geodbq simulate-route --config config.json --ip 8.8.8.8

# Simulate traffic from a specific source IP
geodbq simulate-route --config config.json --domain youtube.com --source-ip 192.168.1.1

# Simulate traffic from a specific inbound
geodbq simulate-route --config config.json --domain netflix.com --inbound-tag socks
```

Example output:

```text
Matched rule #2:

Domain match: geosite:google via [domain] google.com
Balancer tag: my-balancer (strategy: random)
Candidate outbounds: proxy1, proxy2, proxy3
Fallback tag: direct
```

## Notes

- Country codes in geoip.dat are usually uppercase (e.g. US, CN).
- Geosite category names are case-sensitive in Xray but the tool uses case-insensitive matching for convenience.
- Very large categories (cn, geolocation-!cn, gfw) may contain thousands of entries — use --max-show or redirect to file:
```bash
geodbq list-rules cn > cn-rules.txt
```
- The tool clamps invalid CIDR prefixes (e.g. IPv4 /127 → /32) to avoid parsing errors, matching common real-world .dat file quirks.

### Notes on Simulation

- Loads geoip.dat and geosite.dat to evaluate geo rules.
- Supports domain (including geosite), IP/geoip, and source geoip matching.
- Other conditions (ports, protocols, inbound tags) are assumed true and not simulated.
- Domain strategy (AsIs, IpIfNonMatch, IpOnDemand) is supported: the tool will perform DNS lookups for IP-based strategies. Use `--prefer-ip ipv4|ipv6` to prefer IPv4 or IPv6 during DNS resolution. You can still provide `--ip` to force an IP-based simulation.
- Balancer selection shows candidates but does not simulate actual load balancing (e.g., no ping check for leastping).
- If no match, defaults to "no matching rule" (in real Xray, falls to first outbound).

## Getting geoip.dat and geosite.dat

Download the required data files:

- **geoip.dat**: https://github.com/v2ray/geoip/releases/download/202604142202/geoip.dat
- **geosite.dat**: https://github.com/v2ray/domain-list-community/releases/download/20260420014318/dlc.dat

Rename dlc.dat to geosite.dat, or use the `--geosite` flag to specify the path.

Place the files in the same directory as geodbq, or use `--geoip` and `--geosite` flags.

## Testing

```bash
# Run all tests
go test ./... -v

# Run CLI tests only
go test ./cmd/ -v
```

## License
MIT

Built with love for the Xray community.
