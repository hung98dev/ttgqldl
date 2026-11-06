package privacy

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

func ledgerKeys(t *testing.T) ([]byte, []byte) {
	t.Helper()
	a := make([]byte, 32)
	b := make([]byte, 32)
	_, _ = rand.Read(a)
	_, _ = rand.Read(b)
	return a, b
}

func ledgerWithRecords(t *testing.T, count int) (*Ledger, string, []byte, []byte) {
	t.Helper()
	dir := t.TempDir()
	enc, chain := ledgerKeys(t)
	ledger, err := OpenLedger(dir, "matrix-ledger", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= count; i++ {
		if _, err := ledger.Append(LedgerPayload{TombstoneID: "00000000-0000-4000-8000-00000000000" + string(rune('0'+i)), SessionID: "s", Scope: "session", ActorRefProtected: "a", PrivacyEpoch: 1}); err != nil {
			t.Fatal(err)
		}
	}
	return ledger, dir, enc, chain
}

func TestRestoreDefectMatrix(t *testing.T) {
	target := "11111111-1111-4111-8111-111111111111"
	t.Run("01_ledger_directory_absent", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		_ = os.RemoveAll(dir)
		if _, err := ledger.Verify(target); err == nil {
			t.Fatal("missing directory accepted")
		}
	})
	t.Run("02_wrong_aes_key", func(t *testing.T) {
		_, dir, _, chain := ledgerWithRecords(t, 1)
		enc2, _ := ledgerKeys(t)
		other, _ := OpenLedger(dir, "matrix-ledger", enc2, chain)
		if _, err := other.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("wrong AES key = %v", err)
		}
	})
	t.Run("03_wrong_chain_key", func(t *testing.T) {
		_, dir, enc, _ := ledgerWithRecords(t, 1)
		_, chain2 := ledgerKeys(t)
		other, _ := OpenLedger(dir, "matrix-ledger", enc, chain2)
		if _, err := other.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("wrong chain key = %v", err)
		}
	})
	t.Run("04_tampered_header", func(t *testing.T) {
		_, dir, _, _ := ledgerWithRecords(t, 1)
		path := filepath.Join(dir, "00000000000000000001.rec")
		raw, _ := os.ReadFile(path)
		var record LedgerRecord
		_ = json.Unmarshal(raw, &record)
		record.Scope = "village"
		encoded, _ := json.Marshal(record)
		_ = os.WriteFile(path, encoded, 0o600)
		ledger, err := OpenLedger(dir, "matrix-ledger", nil, nil)
		if err == nil {
			if _, verifyErr := ledger.Verify(target); verifyErr == nil {
				t.Fatal("tampered header accepted")
			}
		}
	})
	t.Run("05_tampered_ciphertext", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		path := filepath.Join(dir, "00000000000000000001.rec")
		raw, _ := os.ReadFile(path)
		raw[len(raw)-2] ^= 1
		_ = os.WriteFile(path, raw, 0o600)
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("tamper = %v", err)
		}
	})
	t.Run("06_missing_sequence", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 3)
		_ = os.Remove(filepath.Join(dir, "00000000000000000002.rec"))
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerSequence) {
			t.Fatalf("gap = %v", err)
		}
	})
	t.Run("07_broken_predecessor", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 2)
		path := filepath.Join(dir, "00000000000000000002.rec")
		raw, _ := os.ReadFile(path)
		var record LedgerRecord
		_ = json.Unmarshal(raw, &record)
		record.PrevHash = strings.Repeat("f", 64)
		encoded, _ := json.Marshal(record)
		_ = os.WriteFile(path, encoded, 0o600)
		if _, err := ledger.Verify(target); err == nil {
			t.Fatal("broken predecessor accepted")
		}
	})
	t.Run("08_duplicate_conflict", func(t *testing.T) {
		ledger, _, _, _ := ledgerWithRecords(t, 1)
		if _, err := ledger.Append(LedgerPayload{TombstoneID: "00000000-0000-4000-8000-000000000001", SessionID: "s", Scope: "session", ActorRefProtected: "different", PrivacyEpoch: 1}); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("conflict = %v", err)
		}
	})
	t.Run("09_missing_witness", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		_ = os.Remove(filepath.Join(dir, "head.witness"))
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("missing witness = %v", err)
		}
	})
	t.Run("10_witness_behind", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 2)
		witness, _ := ledger.readWitness()
		witness.LastSeq = 1
		witness.HeadHash = strings.Repeat("a", 64)
		witness.Auth = ledger.witnessAuth(witness)
		encoded, _ := json.Marshal(witness)
		_ = os.WriteFile(filepath.Join(dir, "head.witness"), encoded, 0o600)
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("behind = %v", err)
		}
	})
	t.Run("11_witness_ahead", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		witness, _ := ledger.readWitness()
		witness.LastSeq = 2
		witness.Auth = ledger.witnessAuth(witness)
		encoded, _ := json.Marshal(witness)
		_ = os.WriteFile(filepath.Join(dir, "head.witness"), encoded, 0o600)
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("ahead = %v", err)
		}
	})
	t.Run("12_witness_auth", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		witness, _ := ledger.readWitness()
		witness.Auth = strings.Repeat("0", 64)
		encoded, _ := json.Marshal(witness)
		_ = os.WriteFile(filepath.Join(dir, "head.witness"), encoded, 0o600)
		if _, err := ledger.Verify(target); !errors.Is(err, ErrLedgerIntegrity) {
			t.Fatalf("auth = %v", err)
		}
	})
	t.Run("13_malformed_record", func(t *testing.T) {
		ledger, dir, _, _ := ledgerWithRecords(t, 1)
		_ = os.WriteFile(filepath.Join(dir, "00000000000000000001.rec"), []byte("{"), 0o600)
		if _, err := ledger.Verify(target); err == nil {
			t.Fatal("malformed record accepted")
		}
	})
}

func TestRestoreCoordinatorFailsClosedOnDatabaseAndAbsentSession(t *testing.T) {
	target := "22222222-2222-4222-8222-222222222222"
	enc, chain := ledgerKeys(t)
	dir := t.TempDir()
	ledger, _ := OpenLedger(dir, "restore-matrix", enc, chain)
	if _, err := ledger.Append(LedgerPayload{TombstoneID: "33333333-3333-4333-8333-333333333333", SessionID: "44444444-4444-4444-8444-444444444444", Scope: "session", ActorRefProtected: strings.Repeat("a", 64), PrivacyEpoch: 0}); err != nil {
		t.Fatal(err)
	}
	pool := testPool(t)
	coordinator := &RestoreCoordinator{Ledger: ledger, Pool: pool,
		NewManager: func(string) (*writer.Manager, error) {
			return writer.New(pool, "restore-matrix", writer.Config{LeaseDuration: 5 * time.Second})
		},
		NewReducer: func(m *writer.Manager) (*reducer.Reducer, error) { return reducer.New(m, pool) }}
	if _, err := coordinator.Reapply(context.Background(), target); !errors.Is(err, ErrRestoreIncomplete) {
		t.Fatalf("absent session = %v", err)
	}
	pool.Close()
	closed := &RestoreCoordinator{Ledger: ledger, Pool: pool,
		NewManager: func(string) (*writer.Manager, error) { return nil, errors.New("db unavailable") },
		NewReducer: func(*writer.Manager) (*reducer.Reducer, error) { return nil, errors.New("db unavailable") }}
	if _, err := closed.Reapply(context.Background(), target); err == nil {
		t.Fatal("closed pool accepted")
	}
}

func TestRestoreTargetRejectsDSN(t *testing.T) {
	ledger, _, _, _ := ledgerWithRecords(t, 1)
	coordinator := &RestoreCoordinator{Ledger: ledger, Pool: nil}
	if _, err := coordinator.Reapply(context.Background(), "postgres://user:secret@example/db"); !errors.Is(err, ErrRestoreTarget) {
		t.Fatalf("DSN target = %v", err)
	}
}
