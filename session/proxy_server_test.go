package session

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestReadSOCKS5RequestFormatsIPv6Target(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		req := []byte{0x05, 0x01, 0x00, 0x04}
		ip := net.ParseIP("2606:4700:3031::6815:2d50").To16()
		req = append(req, ip...)
		port := make([]byte, 2)
		binary.BigEndian.PutUint16(port, 443)
		req = append(req, port...)
		_, _ = clientConn.Write(req)
	}()

	target, err := readSOCKS5Request(serverConn)
	<-done
	if err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if target != "[2606:4700:3031::6815:2d50]:443" {
		t.Fatalf("target = %s", target)
	}
}
