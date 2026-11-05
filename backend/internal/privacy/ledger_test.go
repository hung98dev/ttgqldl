package privacy

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func keys(t *testing.T) ([]byte, []byte) {
	t.Helper()
	enc := make([]byte, 32)
	chain := make([]byte, 32)
	if _, err := rand.Read(enc); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(chain); err != nil {
		t.Fatal(err)
	}
	return enc, chain
}

func TestLedgerAppendVerifyAndCoverage(t *testing.T) {
	enc, chain := keys(t)
	ledger, err := OpenLedger(t.TempDir(), "ledger-a", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	payload := LedgerPayload{TombstoneID: "t1", SessionID: "s1", Scope: "session", ActorRefProtected: "a1", PrivacyEpoch: 1}
	record, err := ledger.Append(payload)
	if err != nil {
		t.Fatal(err)
	}
	if record.Seq != 1 || record.AckRef == "" {
		t.Fatalf("record %+v", record)
	}
	if _, err := ledger.Append(LedgerPayload{TombstoneID: "t2", SessionID: "s1", Scope: "session", ActorRefProtected: "a2", PrivacyEpoch: 1}); err != nil {
		t.Fatal(err)
	}
	proof, err := ledger.Verify("db-a", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Verified || proof.LastSeq != 2 || len(proof.Erasures) != 2 {
		t.Fatalf("proof %+v", proof)
	}
	if err := AllowClientServing(proof, time.Now().Add(-time.Minute), []string{"t1", "t2"}); err != nil {
		t.Fatal(err)
	}
	if err := AllowClientServing(proof, time.Now().Add(time.Hour), []string{"t1"}); !errors.Is(err, ErrLedgerCoverage) {
		t.Fatalf("horizon gate %v", err)
	}
	if err := AllowClientServing(proof, time.Time{}, []string{"missing"}); !errors.Is(err, ErrLedgerCoverage) {
		t.Fatalf("missing erasure %v", err)
	}
}

func TestLedgerTamperGapAndWrongKey(t *testing.T) {
	enc, chain := keys(t)
	dir := t.TempDir()
	ledger, err := OpenLedger(dir, "ledger-a", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 3; index++ {
		if _, err := ledger.Append(LedgerPayload{TombstoneID: "t" + string(rune('0'+index)), SessionID: "s", Scope: "session", ActorRefProtected: "a", PrivacyEpoch: 1}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "00000000000000000002.rec")
	raw, _ := os.ReadFile(path)
	raw[len(raw)-2] ^= 0x01
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Verify("", time.Time{}); !errors.Is(err, ErrLedgerIntegrity) {
		t.Fatalf("tamper = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Verify("", time.Time{}); !errors.Is(err, ErrLedgerSequence) {
		t.Fatalf("gap = %v", err)
	}
	enc2, chain2 := keys(t)
	other, err := OpenLedger(dir, "ledger-a", enc2, chain2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Verify("", time.Time{}); !errors.Is(err, ErrLedgerIntegrity) {
		t.Fatalf("wrong key = %v", err)
	}
}

func TestLedgerAppendRejectsMissingFields(t *testing.T) {
	enc, chain := keys(t)
	ledger, err := OpenLedger(t.TempDir(), "l", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Append(LedgerPayload{}); !errors.Is(err, ErrLedgerIntegrity) {
		t.Fatalf("empty = %v", err)
	}
}
