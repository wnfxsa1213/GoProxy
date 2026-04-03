package validator

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultHTTPSTargetsUseSampledSites(t *testing.T) {
	want := []string{
		"https://www.google.com",
		"https://www.openai.com",
		"https://www.github.com",
		"https://www.cloudflare.com",
		"https://httpbin.org/ip",
	}
	if len(httpsTestTargets) != len(want) {
		t.Fatalf("httpsTestTargets len = %d, want %d", len(httpsTestTargets), len(want))
	}
	for i, target := range want {
		if httpsTestTargets[i] != target {
			t.Fatalf("httpsTestTargets[%d] = %s, want %s", i, httpsTestTargets[i], target)
		}
	}
}

func TestCheckHTTPSReachabilityRejectsUntrustedChain(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalTargets := httpsTestTargets
	httpsTestTargets = []string{server.URL}
	defer func() { httpsTestTargets = originalTargets }()

	untrustedClient := &http.Client{}
	if checkHTTPSReachability(untrustedClient) {
		t.Fatalf("expected untrusted TLS chain to fail")
	}

	trustedClient := server.Client()
	if !checkHTTPSReachability(trustedClient) {
		t.Fatalf("expected trusted TLS client to pass")
	}
}
