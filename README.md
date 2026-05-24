# ispcheck

**Find out if your ISP is hijacking your DNS and showing you sponsored pages.**

When you visit a website that doesn't exist, your browser should show a "Site not found" error. Some ISPs intercept these failed lookups and silently redirect you to a sponsored page instead. This is called DNS hijacking — and most people never realise it's happening.

`ispcheck` detects this in one command and tells you exactly which ISP is doing it.

```
✗  Bharti Airtel Limited (2401:4900:50:9::7a3) is hijacking your DNS

  Got a response for a domain that cannot exist:

    http://vteaavshc22753qcv7w411ek.com [200]
    Body: "<html>...Verifying…"
```

---

## Download

Go to the [Releases](https://github.com/Latencot/ispcheck/releases) page and download the file for your system:

| System | File |
|---|---|
| Linux (64-bit) | `ispcheck-linux-amd64` |
| Linux (ARM, e.g. Raspberry Pi) | `ispcheck-linux-arm64` |
| macOS (Intel) | `ispcheck-darwin-amd64` |
| macOS (Apple Silicon / M1/M2/M3) | `ispcheck-darwin-arm64` |
| Windows | `ispcheck-windows-amd64.exe` |

### Linux / macOS — after downloading

Open a terminal, go to where the file was downloaded, and run:

```bash
chmod +x ispcheck-linux-amd64   # make it executable (use your filename)
./ispcheck-linux-amd64           # run it
```

### Windows — after downloading

Double-click the `.exe` file, or open Command Prompt, navigate to the download folder, and run:

```
ispcheck-windows-amd64.exe
```

---

## Usage

```bash
# Run with a random test domain (recommended)
./ispcheck

# Test with a specific domain
./ispcheck somefakedomain.com
```

No flags, no config, no account needed.

---

## What the output means

**Hijacking detected:**
```
✗  Bharti Airtel Limited (2401:4900:50:9::7a3) is hijacking your DNS
```
Your ISP is intercepting failed DNS lookups and redirecting you to their own page. The tool shows you which ISP and which DNS server is responsible.

**Clean:**
```
✓  ISP is NOT monetizing error pages
```
Your connection is clean. Failed lookups fail properly without any redirection.

---

## How to fix it

**1. Enable DNS-over-HTTPS in your browser** *(easiest, fixes it just for that browser)*

- **Firefox:** Settings → Privacy & Security → DNS over HTTPS → Select *Max Protection* → Choose Cloudflare
- **Chrome:** Settings → Privacy and Security → Security → Use secure DNS → With Cloudflare (1.1.1.1)

**2. Use a VPN** *(fixes it for all traffic)*

Any VPN routes your traffic outside the ISP's network entirely.

**3. Change DNS on your router** *(fixes it for all devices on that network)*

Set your router's DNS to `1.1.1.1` (Cloudflare) or `8.8.8.8` (Google).
> ⚠️ This may not work on Airtel — they intercept DNS queries on port 53 regardless of which server you configure.

**4. Linux only — DNS-over-TLS system-wide**

Edit `/etc/systemd/resolved.conf`:
```ini
[Resolve]
DNS=1.1.1.1#cloudflare-dns.com
DNSOverTLS=yes
```
Then restart: `sudo systemctl restart systemd-resolved`

This fixes it for all apps on your system, not just the browser.

---

## How it works

`ispcheck` generates a random 24-character domain name (e.g. `vteaavshc22753qcv7w411ek.com`) that cannot possibly be registered or exist. It then makes an HTTP request to that domain.

- **Clean ISP:** DNS returns "not found" → connection fails → no response
- **Hijacking ISP:** DNS returns the ISP's own server IP → connection succeeds → ISP serves a redirect or sponsored page

If a response is received, the tool looks up who owns the DNS server your system is using and names them.

---

## Build from source

Requires [Go 1.24+](https://golang.org/dl/).

```bash
git clone https://github.com/Latencot/ispcheck.git
cd ispcheck
go build -o ispcheck .
./ispcheck
```

---

## Known ISPs doing this

- Bharti Airtel (India)
- Reliance Jio (India)
- BSNL (India)
- Comcast (USA) — discontinued
- Verizon (USA) — discontinued
- British Telecom (UK) — discontinued

---

Built by [Latencot](https://latencot.com)
