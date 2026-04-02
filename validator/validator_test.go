package validator

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
