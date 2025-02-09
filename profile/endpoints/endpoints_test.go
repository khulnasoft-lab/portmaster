package endpoints

import (
	"context"
	"net"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/khulnasoft-lab/portmaster/core/pmtesting"
	"github.com/khulnasoft-lab/portmaster/intel"
)

func TestMain(m *testing.M) {
	pmtesting.TestMain(m, intel.Module)
}

func testEndpointMatch(t *testing.T, ep Endpoint, entity *intel.Entity, expectedResult EPResult) {
	t.Helper()

	result, _ := ep.Matches(context.TODO(), entity)
	if result != expectedResult {
		t.Errorf(
			"line %d: unexpected result for endpoint %s and entity %+v: result=%s, expected=%s",
			getLineNumberOfCaller(1),
			ep,
			entity,
			result,
			expectedResult,
		)
	}
}

func testFormat(t *testing.T, endpoint string, shouldSucceed bool) {
	t.Helper()

	_, err := parseEndpoint(endpoint)
	if shouldSucceed {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}
}

func TestEndpointFormat(t *testing.T) {
	t.Parallel()

	testFormat(t, "+ .", false)
	testFormat(t, "+ .at", true)
	testFormat(t, "+ .at.", true)
	testFormat(t, "+ 1.at", true)
	testFormat(t, "+ 1.at.", true)
	testFormat(t, "+ 1.f.ix.de.", true)
	testFormat(t, "+ *contains*", true)
	testFormat(t, "+ *has.suffix", true)
	testFormat(t, "+ *.has.suffix", true)
	testFormat(t, "+ *has.prefix*", true)
	testFormat(t, "+ *has.prefix.*", true)
	testFormat(t, "+ .sub.and.prefix.*", false)
	testFormat(t, "+ *.sub..and.prefix.*", false)
}

func TestEndpointMatching(t *testing.T) { //nolint:maintidx // TODO
	t.Parallel()

	// ANY

	ep, err := parseEndpoint("+ *")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)

	// DOMAIN

	// wildcard domains
	ep, err = parseEndpoint("+ *example.com")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc.example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc-example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc.example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc-example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)

	ep, err = parseEndpoint("+ *.example.com")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc.example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc-example.com.",
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc.example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc-example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)

	ep, err = parseEndpoint("+ .example.com")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc.example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc-example.com.",
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc.example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc-example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)

	ep, err = parseEndpoint("+ example.*")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc.example.com.",
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc.example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)

	ep, err = parseEndpoint("+ *.exampl*")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "abc.example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "abc.example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)

	ep, err = parseEndpoint("+ *.com.")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.org.",
	}).Init(0), NoMatch)

	// protocol

	ep, err = parseEndpoint("+ example.com UDP")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 17,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), NoMatch)

	// ports

	ep, err = parseEndpoint("+ example.com 17/442-444")
	if err != nil {
		t.Fatal(err)
	}

	entity := (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 17,
		Port:     441,
	}).Init(0)
	testEndpointMatch(t, ep, entity, NoMatch)

	entity.Port = 442
	entity.Init(0)
	testEndpointMatch(t, ep, entity, Permitted)

	entity.Port = 443
	entity.Init(0)
	testEndpointMatch(t, ep, entity, Permitted)

	entity.Port = 444
	entity.Init(0)
	testEndpointMatch(t, ep, entity, Permitted)

	entity.Port = 445
	entity.Init(0)
	testEndpointMatch(t, ep, entity, NoMatch)

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), NoMatch)

	// IP

	ep, err = parseEndpoint("+ 10.2.3.4")
	if err != nil {
		t.Fatal(err)
	}

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 17,
		Port:     443,
	}).Init(0), Permitted)

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "",
		IP:       net.ParseIP("10.2.3.3"),
		Protocol: 6,
		Port:     443,
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		Domain:   "example.com.",
		IP:       net.ParseIP("10.2.3.5"),
		Protocol: 17,
		Port:     443,
	}).Init(0), NoMatch)

	testEndpointMatch(t, ep, (&intel.Entity{
		Domain: "example.com.",
	}).Init(0), NoMatch)

	// IP Range

	ep, err = parseEndpoint("+ 10.2.3.0/24")
	if err != nil {
		t.Fatal(err)
	}
	testEndpointMatch(t, ep, (&intel.Entity{
		IP: net.ParseIP("10.2.2.4"),
	}).Init(0), NoMatch)
	testEndpointMatch(t, ep, (&intel.Entity{
		IP: net.ParseIP("10.2.3.4"),
	}).Init(0), Permitted)
	testEndpointMatch(t, ep, (&intel.Entity{
		IP: net.ParseIP("10.2.4.4"),
	}).Init(0), NoMatch)

	// Skip test that need the geoip database in CI.
	if !testing.Short() {

		// ASN

		ep, err = parseEndpoint("+ AS15169")
		if err != nil {
			t.Fatal(err)
		}

		entity = (&intel.Entity{IP: net.IPv4(8, 8, 8, 8)}).Init(0)
		testEndpointMatch(t, ep, entity, Permitted)

		entity = (&intel.Entity{IP: net.IPv4(1, 1, 1, 1)}).Init(0)
		testEndpointMatch(t, ep, entity, NoMatch)

		// Country

		ep, err = parseEndpoint("+ AT")
		if err != nil {
			t.Fatal(err)
		}

		entity = (&intel.Entity{IP: net.IPv4(194, 232, 104, 1)}).Init(0) // orf.at
		testEndpointMatch(t, ep, entity, Permitted)

		entity = (&intel.Entity{IP: net.IPv4(151, 101, 1, 164)}).Init(0) // nytimes.com
		testEndpointMatch(t, ep, entity, NoMatch)

	}

	// Scope

	ep, err = parseEndpoint("+ Localhost,LAN")
	if err != nil {
		t.Fatal(err)
	}

	entity = (&intel.Entity{IP: net.IPv4(192, 168, 0, 1)}).Init(0)
	testEndpointMatch(t, ep, entity, Permitted)

	entity = (&intel.Entity{IP: net.IPv4(151, 101, 1, 164)}).Init(0) // nytimes.com
	testEndpointMatch(t, ep, entity, NoMatch)

	// Port with protocol wildcard

	ep, err = parseEndpoint("+ * */443")
	if err != nil {
		t.Fatal(err)
	}
	entity = &intel.Entity{
		Domain:   "",
		IP:       net.ParseIP("10.2.3.4"),
		Protocol: 6,
		Port:     443,
	}
	entity.Init(0)
	testEndpointMatch(t, ep, entity, Permitted)

	// Lists

	// Skip test that need the filter lists in CI.
	if !testing.Short() {
		_, err = parseEndpoint("+ L:A,B,C")
		if err != nil {
			t.Fatal(err)
		}
	}

	// TODO: write test for lists matcher
}

func getLineNumberOfCaller(levels int) int {
	_, _, line, _ := runtime.Caller(levels + 1) //nolint:dogsled
	return line
}
