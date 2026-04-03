package session

import (
	"errors"
	"path/filepath"
	"testing"

	"goproxy/config"
	"goproxy/storage"
)

func TestAcquireSkipsUnreachableCandidate(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("DATA_DIR", tempDir)

	cfg := config.Load()
	cfg.ValidateTimeout = 1

	store, err := storage.New(filepath.Join(tempDir, "proxy.db"))
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close storage: %v", err)
		}
	})

	sessionStore := NewStore(store.GetDB())
	manager := NewManager(sessionStore, store, cfg)

	const badAddr = "127.0.0.1:11080"
	const goodAddr = "127.0.0.1:12080"

	if err := store.AddProxy(badAddr, "socks5"); err != nil {
		t.Fatalf("add bad proxy: %v", err)
	}
	if err := store.UpdateExitInfo(badAddr, "1.1.1.1", "US Bad", 200, "US", "UTC"); err != nil {
		t.Fatalf("update bad proxy: %v", err)
	}

	if err := store.AddProxy(goodAddr, "socks5"); err != nil {
		t.Fatalf("add good proxy: %v", err)
	}
	if err := store.UpdateExitInfo(goodAddr, "2.2.2.2", "US Good", 800, "US", "UTC"); err != nil {
		t.Fatalf("update good proxy: %v", err)
	}

	manager.probeCandidate = func(proxy storage.Proxy) error {
		if proxy.Address == badAddr {
			return errors.New("dial tcp: i/o timeout")
		}
		return nil
	}

	resp, err := manager.Acquire(AcquireRequest{
		TaskID:   "task-1",
		Protocol: "socks5",
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	active := manager.sessions[resp.SessionID]
	if active == nil {
		t.Fatalf("session %s not found", resp.SessionID)
	}
	if active.ProxyAddress != goodAddr {
		t.Fatalf("selected proxy = %s, want %s", active.ProxyAddress, goodAddr)
	}

	proxies, err := store.GetAll()
	if err != nil {
		t.Fatalf("get all proxies: %v", err)
	}

	foundBad := false
	for _, proxy := range proxies {
		if proxy.Address != badAddr {
			continue
		}
		foundBad = true
		if proxy.FailCount != 1 {
			t.Fatalf("bad proxy fail_count = %d, want 1", proxy.FailCount)
		}
	}
	if !foundBad {
		t.Fatalf("bad proxy %s not found", badAddr)
	}
}
