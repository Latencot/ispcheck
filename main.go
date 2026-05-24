package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ANSI helpers
var useColor = isTerminal()

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func ansi(code string) string {
	if useColor {
		return code
	}
	return ""
}

func b(s string) string    { return ansi("\033[1m") + s + ansi("\033[0m") }
func dim(s string) string  { return ansi("\033[2m") + s + ansi("\033[0m") }
func pass(s string) string { return ansi("\033[32m") + s + ansi("\033[0m") }
func fail(s string) string { return ansi("\033[31m") + s + ansi("\033[0m") }
func warn(s string) string { return ansi("\033[33m") + s + ansi("\033[0m") }
func hi(s string) string   { return ansi("\033[36m") + s + ansi("\033[0m") }

func randDomain() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 24 random chars — astronomically unlikely to be a registered domain,
	// but still goes through the ISP's upstream DNS (unlike .invalid which
	// systemd-resolved handles locally and never forwards upstream)
	b := make([]byte, 24)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b) + ".com"
}

type hop struct {
	url    string
	status int
}

type probeResult struct {
	monetized bool
	chain     []hop
	body      string // snippet from final response if not a redirect
	err       error
}

func probe(domain string) probeResult {
	chain := traceChain("http://" + domain)

	// No hops at all means DNS failed before any connection — ISP is clean
	if len(chain) == 0 {
		return probeResult{err: fmt.Errorf("could not connect")}
	}
	// A single hop with status 0 means connection error on the domain itself — also clean
	if len(chain) == 1 && chain[0].status == 0 {
		return probeResult{err: fmt.Errorf("connection failed: %s", chain[0].url)}
	}

	// Read body snippet from the final hop
	var snippet string
	if last := chain[len(chain)-1]; last.status != 0 {
		req, _ := http.NewRequest("GET", last.url, nil)
		req.Header.Set("User-Agent", firefoxUA)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		client := &http.Client{Timeout: 5 * time.Second}
		if resp, err := client.Do(req); err == nil {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 240))
			resp.Body.Close()
			snippet = strings.Join(strings.Fields(string(raw)), " ")
			if len(snippet) > 120 {
				snippet = snippet[:120] + "…"
			}
		}
	}

	return probeResult{monetized: true, chain: chain, body: snippet}
}

const firefoxUA = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

// upstreamDNS returns the first public (non-loopback, non-private) DNS server
// the system is actually querying. Checks systemd-resolved's real upstream first,
// then falls back to /etc/resolv.conf.
func upstreamDNS() string {
	for _, path := range []string{"/run/systemd/resolve/resolv.conf", "/etc/resolv.conf"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "nameserver") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			ip := net.ParseIP(parts[1])
			if ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsPrivate() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

type ipInfo struct {
	Org string `json:"org"`
	ISP string `json:"isp"`
}

// whoIsHijacking looks up the org that owns the given IP via ip-api.com.
func whoIsHijacking(ip string) string {
	if ip == "" {
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=org,isp")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var info ipInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return ""
	}
	if info.ISP != "" {
		return info.ISP
	}
	return info.Org
}

// traceChain follows redirects one at a time, recording every URL and status code.
func traceChain(start string) []hop {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	var chain []hop
	current := start
	for i := 0; i < 15; i++ {
		req, _ := http.NewRequest("GET", current, nil)
		req.Header.Set("User-Agent", firefoxUA)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err := client.Do(req)
		if err != nil {
			// DNS error on the very first hop = clean; otherwise record the dead end
			if i > 0 {
				chain = append(chain, hop{url: current, status: 0})
			}
			break
		}
		resp.Body.Close()
		chain = append(chain, hop{url: current, status: resp.StatusCode})
		loc := resp.Header.Get("Location")
		if loc == "" || resp.StatusCode < 300 || resp.StatusCode >= 400 {
			break
		}
		current = loc
	}
	return chain
}

func main() {
	fmt.Println()
	fmt.Println(b("  ispcheck — ISP Error Page Monetization Detector"))
	fmt.Println(dim("  Checks if your ISP serves ads or redirects when a connection fails"))
	fmt.Println()

	nxDomain := randDomain()
	if len(os.Args) > 1 && os.Args[1] != "--help" && os.Args[1] != "-h" {
		nxDomain = os.Args[1]
		fmt.Printf("  Test domain: %s\n\n", hi(nxDomain))
	} else {
		fmt.Printf("  Test domain: %s\n", hi(nxDomain))
		fmt.Println(dim("  (24 random chars — cannot realistically be a registered domain)\n"))
	}

	dnsIP := upstreamDNS()
	isp := whoIsHijacking(dnsIP)

	r := probe(nxDomain)

	if r.err != nil {
		fmt.Printf("  %s\n\n", pass("✓  ISP is NOT monetizing error pages"))
		fmt.Printf("  Connection failed as expected: %s\n", dim(r.err.Error()))
		fmt.Println()
		os.Exit(0)
	}

	hijacker := "your ISP"
	if isp != "" {
		hijacker = isp
		if dnsIP != "" {
			hijacker += " (" + dnsIP + ")"
		}
	} else if dnsIP != "" {
		hijacker = dnsIP
	}

	fmt.Printf("  %s\n\n", fail("✗  "+hijacker+" is hijacking your DNS"))
	fmt.Printf("  Got a response for a domain that cannot exist:\n\n")
	for i, h := range r.chain {
		status := ""
		if h.status > 0 {
			status = fmt.Sprintf(" [%d]", h.status)
		}
		if i == 0 {
			fmt.Printf("    %s%s\n", hi(h.url), dim(status))
		} else {
			fmt.Printf("    %s↳ %s%s\n", dim("  "), hi(h.url), dim(status))
		}
	}
	if r.body != "" {
		fmt.Printf("\n    Body : %s\n", dim(`"`+r.body+`"`))
	}
	fmt.Println()
	fmt.Println(b("  What this means:"))
	fmt.Println("  Your ISP intercepts failed DNS lookups and serves their own page")
	fmt.Println("  instead of letting the browser show a standard 'site not found' error.")
	fmt.Println("  This is typically used to show ads or a sponsored search page.")
	fmt.Println()
	fmt.Println(b("  Fix:"))
	fmt.Println("  Enable DNS-over-HTTPS in your browser:")
	fmt.Println("    Firefox : Settings → Privacy & Security → DNS over HTTPS → Max Protection")
	fmt.Println("    Chrome  : Settings → Security → Use secure DNS → Cloudflare (1.1.1.1)")
	fmt.Println()
	os.Exit(1)
}
