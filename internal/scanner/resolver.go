package scanner

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// ResolveStatus represents the outcome of a DNS resolution attempt.
type ResolveStatus int

const (
	Success ResolveStatus = iota // Found an A record.
	NotFound                     // Definitive negative result (e.g., NXDOMAIN, NOERROR with no A record).
	Failed                       // All retry attempts failed due to network errors.
)

const (
	maxAttempts = 6 // 1 initial + 5 retries
	retryDelay  = 500 * time.Millisecond
)

// Resolver is responsible for DNS resolutions.
type Resolver struct {
	dnsClient    *dns.Client
	tier1Servers []string
	tier2Servers []string
	debugNetwork bool
}

// NewResolver creates a new Resolver with default tiered DNS servers.
func NewResolver(debugNetwork bool) *Resolver {
	return &Resolver{
		dnsClient: &dns.Client{
			Net:          "udp",
			Timeout:      5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		tier1Servers: []string{
			"8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53",
			"114.114.114.114:53", "114.114.115.115:53", "223.5.5.5:53",
		},
		tier2Servers: []string{
			"119.29.29.29:53", "119.28.28.28:53", "223.6.6.6:53",
			"9.9.9.9:53", "149.112.112.112:53",
		},
		debugNetwork: debugNetwork,
	}
}

// SetDNSServers sets custom DNS servers for the resolver.
func (r *Resolver) SetDNSServers(servers []string) {
	var validServers []string
	for _, s := range servers {
		if !strings.Contains(s, ":") {
			s = s + ":53"
		}
		validServers = append(validServers, s)
	}
	if len(validServers) > 0 {
		r.tier1Servers = validServers
		r.tier2Servers = []string{}
	}
}

func selectServer(servers []string, usedServers map[string]bool) (string, bool) {
	if len(servers) == 0 {
		return "", false
	}
	available := make([]string, 0, len(servers))
	for _, s := range servers {
		if !usedServers[s] {
			available = append(available, s)
		}
	}
	if len(available) == 0 {
		return "", false
	}
	return available[rand.Intn(len(available))], true
}

// Resolve performs a DNS A record lookup and returns ip, status, attempts, and error.
func (r *Resolver) Resolve(domain string) (string, ResolveStatus, int, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	usedServers := make(map[string]bool)
	var lastErr error
	isNetworkError := false

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var server string
		var found bool

		if attempt <= 3 {
			server, found = selectServer(r.tier1Servers, usedServers)
		} else {
			server, found = selectServer(r.tier2Servers, usedServers)
		}

		if !found {
			continue
		}
		usedServers[server] = true

		reply, _, err := r.dnsClient.Exchange(msg, server)
		lastErr = err
		isNetworkError = false

		if err == nil {
			if reply.Rcode != dns.RcodeSuccess {
				return "", NotFound, attempt, fmt.Errorf("invalid rcode: %s", dns.RcodeToString[reply.Rcode])
			}
			for _, ans := range reply.Answer {
				if a, ok := ans.(*dns.A); ok {
					return a.A.String(), Success, attempt, nil
				}
			}
			return "", NotFound, attempt, nil
		}

		if _, ok := err.(net.Error); ok {
			isNetworkError = true
			if r.debugNetwork {
				log.Printf("Network error for %s using %s (attempt %d/%d): %v. Retrying...", domain, server, attempt, maxAttempts, err)
			}
			time.Sleep(retryDelay)
			continue
		}

		return "", Failed, attempt, err
	}

	if isNetworkError {
		return "", Failed, maxAttempts, fmt.Errorf("all %d attempts failed for %s; last network error: %w", maxAttempts, domain, lastErr)
	}
	
	return "", NotFound, maxAttempts, fmt.Errorf("all attempts failed for %s without a definitive result; last error: %w", maxAttempts, domain, lastErr)
}
