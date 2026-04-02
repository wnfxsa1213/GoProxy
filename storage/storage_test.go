package storage

import (
	"path/filepath"
	"testing"
)

func TestNewMigratesQualityColumns(t *testing.T) {
	t.Parallel()

	store := newTestStorage(t)

	for _, column := range []string{"quality_score", "risk_count"} {
		var count int
		err := store.GetDB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name = ?`, column).Scan(&count)
		if err != nil {
			t.Fatalf("query column %s: %v", column, err)
		}
		if count != 1 {
			t.Fatalf("column %s missing after migration", column)
		}
	}
}

func TestUpdateExitInfoAndOutcomeRecalculateQuality(t *testing.T) {
	t.Parallel()

	store := newTestStorage(t)
	address := "127.0.0.1:8080"
	if err := store.AddProxy(address, "http"); err != nil {
		t.Fatalf("add proxy: %v", err)
	}
	if err := store.UpdateExitInfo(address, "1.1.1.1", "US Test", 250, "US", "UTC"); err != nil {
		t.Fatalf("update exit info: %v", err)
	}

	proxy := mustFindProxy(t, store, address)
	if proxy.QualityScore != 70 || proxy.QualityGrade != "A" {
		t.Fatalf("after update exit info score=%d grade=%s, want 70/A", proxy.QualityScore, proxy.QualityGrade)
	}

	score, grade, err := store.RecordProxyOutcome(proxy.ID, address, "success", false)
	if err != nil {
		t.Fatalf("record success outcome: %v", err)
	}
	if score != 75 || grade != "A" {
		t.Fatalf("after success outcome score=%d grade=%s, want 75/A", score, grade)
	}

	score, grade, err = store.RecordProxyOutcome(proxy.ID, address, "risk_blocked", true)
	if err != nil {
		t.Fatalf("record risk outcome: %v", err)
	}
	if score != 58 || grade != "B" {
		t.Fatalf("after risk outcome score=%d grade=%s, want 58/B", score, grade)
	}

	proxy = mustFindProxy(t, store, address)
	if proxy.RiskCount != 1 {
		t.Fatalf("risk_count = %d, want 1", proxy.RiskCount)
	}
}

func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "proxy.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close storage: %v", err)
		}
	})
	return store
}

func mustFindProxy(t *testing.T, store *Storage, address string) Proxy {
	t.Helper()

	proxies, err := store.GetAll()
	if err != nil {
		t.Fatalf("get all proxies: %v", err)
	}
	for _, proxy := range proxies {
		if proxy.Address == address {
			return proxy
		}
	}
	t.Fatalf("proxy %s not found", address)
	return Proxy{}
}
