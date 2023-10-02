package firewall

import (
	"context"
	"strings"

	"github.com/khulnasoft-lab/portmaster/compat"
	"github.com/khulnasoft-lab/portmaster/nameserver/nsutil"
	"github.com/khulnasoft-lab/portmaster/network"
	"github.com/khulnasoft-lab/portmaster/network/packet"
	"github.com/khulnasoft-lab/portmaster/profile/endpoints"
)

var resolverFilterLists = []string{"17-DNS"}

// PreventBypassing checks if the connection should be denied or permitted
// based on some bypass protection checks.
func PreventBypassing(ctx context.Context, conn *network.Connection) (endpoints.EPResult, string, nsutil.Responder) {
	// Exclude incoming connections.
	if conn.Inbound {
		return endpoints.NoMatch, "", nil
	}

	// Exclude ICMP.
	switch packet.IPProtocol(conn.Entity.Protocol) { //nolint:exhaustive // Checking for specific values only.
	case packet.ICMP, packet.ICMPv6:
		return endpoints.NoMatch, "", nil
	}

	// Block firefox canary domain to disable DoH.
	// This MUST also affect the System Resolver, because the return value must
	// be correct for this to work.
	if strings.ToLower(conn.Entity.Domain) == "use-application-dns.net." {
		return endpoints.Denied,
			"blocked canary domain to prevent enabling of DNS-over-HTTPs",
			nsutil.NxDomain()
	}

	// Exclude DNS requests coming from the System Resolver.
	// This MUST also affect entities in the secure dns filter list, else the
	// System Resolver is wrongly accused of bypassing.
	if conn.Type == network.DNSRequest && conn.Process().IsSystemResolver() {
		return endpoints.NoMatch, "", nil
	}

	// Block bypass attempts using an (encrypted) DNS server.
	switch {
	case conn.Entity.Port == 53:
		return endpoints.Denied,
			"blocked DNS query, manual dns setup required",
			nsutil.BlockIP()
	case conn.Entity.Port == 853:
		// Block connections to port 853 - DNS over TLS.
		fallthrough
	case conn.Entity.MatchLists(resolverFilterLists):
		// Block connection entities in the secure dns filter list.
		compat.ReportSecureDNSBypassIssue(conn.Process())
		return endpoints.Denied,
			"blocked rogue connection to DNS resolver",
			nsutil.BlockIP()
	}

	return endpoints.NoMatch, "", nil
}
