package netutil

import "testing"

func TestJoinTargetHostPort(t *testing.T) {
	t.Parallel()

	if got := JoinTargetHostPort("2606:4700:3031::6815:2d50", 443); got != "[2606:4700:3031::6815:2d50]:443" {
		t.Fatalf("ipv6 join = %s", got)
	}
	if got := JoinTargetHostPort("example.com", 443); got != "example.com:443" {
		t.Fatalf("domain join = %s", got)
	}
}

func TestSplitTargetHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		host   string
		port   string
	}{
		{
			name:   "bracketed ipv6",
			target: "[2606:4700:3031::6815:2d50]:443",
			host:   "2606:4700:3031::6815:2d50",
			port:   "443",
		},
		{
			name:   "raw ipv6 fallback",
			target: "2606:4700:3031::6815:2d50:443",
			host:   "2606:4700:3031::6815:2d50",
			port:   "443",
		},
		{
			name:   "domain",
			target: "example.com:443",
			host:   "example.com",
			port:   "443",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			host, port, err := SplitTargetHostPort(tc.target)
			if err != nil {
				t.Fatalf("split failed: %v", err)
			}
			if host != tc.host || port != tc.port {
				t.Fatalf("got %s %s, want %s %s", host, port, tc.host, tc.port)
			}
		})
	}
}
