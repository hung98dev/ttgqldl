// Package privacy implements the W0406 erasure workflow, HMAC alias cleanup and
// the independent encrypted append-only deletion ledger.
package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const LedgerFormat = "ttg-deletion-ledger-v1"

var (
	ErrLedgerIntegrity = errors.New("privacy: deletion ledger integrity failed")
	ErrLedgerSequence  = errors.New("privacy: deletion ledger sequence gap")
	ErrLedgerKey       = errors.New("privacy: deletion ledger key is invalid")
	ErrLedgerCoverage  = errors.New("privacy: deletion ledger coverage is insufficient")
)

type LedgerRecord struct {
	Version           int    `json:"version"`
	LedgerID          string `json:"ledger_id"`
	Seq               uint64 `json:"seq"`
	TombstoneID       string `json:"tombstone_id"`
	Scope             string `json:"scope"`
	ActorRefProtected string `json:"actor_ref_protected"`
	PrevHash          string `json:"prev_hash"`
	Nonce             string `json:"nonce"`
	Ciphertext        string `json:"ciphertext"`
	Hash              string `json:"hash"`
	AckRef            string `json:"ack_ref"`
}

type LedgerPayload struct {
	TombstoneID       string `json:"tombstone_id"`
	SessionID         string `json:"session_id"`
	Scope             string `json:"scope"`
	ActorRefProtected string `json:"actor_ref_protected"`
	PrivacyEpoch      int64  `json:"privacy_epoch"`
	CreatedAt         string `json:"created_at"`
}

type Ledger struct {
	Dir      string
	ID       string
	encKey   []byte
	chainKey []byte
	mu       sync.Mutex
}

func OpenLedger(dir, id string, encKey, chainKey []byte) (*Ledger, error) {
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(id) == "" || len(encKey) != 32 || len(chainKey) != 32 {
		return nil, ErrLedgerKey
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil { // #nosec G302 -- ledger directory must be owner-only 0700
		return nil, err
	}
	return &Ledger{Dir: dir, ID: id, encKey: append([]byte(nil), encKey...), chainKey: append([]byte(nil), chainKey...)}, nil
}

func (l *Ledger) recordPath(seq uint64) string {
	return filepath.Join(l.Dir, fmt.Sprintf("%020d.rec", seq))
}

func (l *Ledger) Append(payload LedgerPayload) (LedgerRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	records, err := l.readRecords()
	if err != nil {
		return LedgerRecord{}, err
	}
	seq := uint64(1)
	prev := strings.Repeat("0", 64)
	for _, existing := range records {
		if existing.TombstoneID == payload.TombstoneID {
			if existing.Scope != payload.Scope || existing.ActorRefProtected != payload.ActorRefProtected {
				return LedgerRecord{}, ErrLedgerIntegrity
			}
			return existing, nil
		}
	}
	if len(records) > 0 {
		last := records[len(records)-1]
		seq = last.Seq + 1
		prev = last.Hash
	}
	if payload.TombstoneID == "" || payload.Scope == "" || payload.ActorRefProtected == "" {
		return LedgerRecord{}, ErrLedgerIntegrity
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return LedgerRecord{}, err
	}
	block, err := aes.NewCipher(l.encKey)
	if err != nil {
		return LedgerRecord{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return LedgerRecord{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return LedgerRecord{}, err
	}
	ad := []byte(l.ID + "|" + strconv.FormatUint(seq, 10) + "|" + prev)
	ciphertext := gcm.Seal(nil, nonce, rawPayload, ad)
	header := fmt.Sprintf("%s|%d|%d|%s|%s|%s|%s|%s", l.ID, seq, LedgerFormatVersion, payload.TombstoneID, payload.Scope, payload.ActorRefProtected, prev, hex.EncodeToString(nonce))
	mac := hmac.New(sha256.New, l.chainKey)
	_, _ = mac.Write([]byte(header))
	_, _ = mac.Write(ciphertext)
	hash := hex.EncodeToString(mac.Sum(nil))
	record := LedgerRecord{
		Version: LedgerFormatVersion, LedgerID: l.ID, Seq: seq, TombstoneID: payload.TombstoneID,
		Scope: payload.Scope, ActorRefProtected: payload.ActorRefProtected, PrevHash: prev,
		Nonce: hex.EncodeToString(nonce), Ciphertext: hex.EncodeToString(ciphertext), Hash: hash, AckRef: hash,
	}
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return LedgerRecord{}, err
	}
	tmp := l.recordPath(seq) + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- path is generated under the owner-only ledger root
	if err != nil {
		return LedgerRecord{}, err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return LedgerRecord{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return LedgerRecord{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return LedgerRecord{}, err
	}
	if err := os.Rename(tmp, l.recordPath(seq)); err != nil {
		_ = os.Remove(tmp)
		return LedgerRecord{}, err
	}
	dir, err := os.Open(l.Dir)
	if err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return record, nil
}

const LedgerFormatVersion = 1

func (l *Ledger) readRecords() ([]LedgerRecord, error) {
	entries, err := os.ReadDir(l.Dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".rec") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	records := make([]LedgerRecord, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(l.Dir, name)) // #nosec G304 -- name is a directory entry under the owner-only ledger root
		if err != nil {
			return nil, err
		}
		var record LedgerRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, ErrLedgerIntegrity
		}
		if err := l.verifyRecord(record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (l *Ledger) verifyRecord(record LedgerRecord) error {
	if record.Version != LedgerFormatVersion || record.LedgerID != l.ID || record.Seq == 0 || record.Hash != record.AckRef || record.TombstoneID == "" || record.Scope == "" || record.ActorRefProtected == "" {
		return ErrLedgerIntegrity
	}
	header := fmt.Sprintf("%s|%d|%d|%s|%s|%s|%s|%s", record.LedgerID, record.Seq, record.Version, record.TombstoneID, record.Scope, record.ActorRefProtected, record.PrevHash, record.Nonce)
	ciphertext, err := hex.DecodeString(record.Ciphertext)
	if err != nil {
		return ErrLedgerIntegrity
	}
	mac := hmac.New(sha256.New, l.chainKey)
	_, _ = mac.Write([]byte(header))
	_, _ = mac.Write(ciphertext)
	if !hmac.Equal([]byte(record.Hash), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return ErrLedgerIntegrity
	}
	return nil
}

type CoverageProof struct {
	LedgerID        string    `json:"ledger_id"`
	LastSeq         uint64    `json:"last_seq"`
	HeadHash        string    `json:"head_hash"`
	VerifiedAt      time.Time `json:"verified_at"`
	CoverageThrough time.Time `json:"coverage_through"`
	TargetIdentity  string    `json:"target_identity"`
	Erasures        []string  `json:"erasures"`
	Verified        bool      `json:"verified"`
}

func (l *Ledger) Verify(targetIdentity string, coverageThrough time.Time) (CoverageProof, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	records, err := l.readRecords()
	if err != nil {
		return CoverageProof{}, err
	}
	prev := strings.Repeat("0", 64)
	erasures := make([]string, 0, len(records))
	for index, record := range records {
		if record.Seq != uint64(index+1) || record.PrevHash != prev {
			return CoverageProof{}, ErrLedgerSequence
		}
		if err := l.decryptRecord(record); err != nil {
			return CoverageProof{}, err
		}
		erasures = append(erasures, record.TombstoneID)
		prev = record.Hash
	}
	head := prev
	if len(records) > 0 {
		head = records[len(records)-1].Hash
	}
	return CoverageProof{
		LedgerID: l.ID, LastSeq: uint64(len(records)), HeadHash: head, VerifiedAt: time.Now().UTC(),
		CoverageThrough: coverageThrough, TargetIdentity: targetIdentity, Erasures: erasures, Verified: true,
	}, nil
}

func (l *Ledger) decryptRecord(record LedgerRecord) error {
	block, err := aes.NewCipher(l.encKey)
	if err != nil {
		return ErrLedgerKey
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ErrLedgerKey
	}
	nonce, err := hex.DecodeString(record.Nonce)
	if err != nil {
		return ErrLedgerIntegrity
	}
	ciphertext, err := hex.DecodeString(record.Ciphertext)
	if err != nil {
		return ErrLedgerIntegrity
	}
	ad := []byte(l.ID + "|" + strconv.FormatUint(record.Seq, 10) + "|" + record.PrevHash)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return ErrLedgerIntegrity
	}
	var payload LedgerPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil || payload.TombstoneID != record.TombstoneID {
		return ErrLedgerIntegrity
	}
	return nil
}

// AllowClientServing requires a verifier-produced proof, not caller booleans.
func AllowClientServing(proof CoverageProof, requiredHorizon time.Time, requiredErasures []string) error {
	if !proof.Verified || proof.LedgerID == "" || proof.LastSeq == 0 || proof.CoverageThrough.Before(requiredHorizon) {
		return ErrLedgerCoverage
	}
	seen := make(map[string]struct{}, len(proof.Erasures))
	for _, id := range proof.Erasures {
		seen[id] = struct{}{}
	}
	for _, id := range requiredErasures {
		if _, ok := seen[id]; !ok {
			return ErrLedgerCoverage
		}
	}
	return nil
}

// ReplayTombstones returns the verified tombstone IDs for a restore. The caller must
// apply each local deletion transactionally before granting readiness.
func (l *Ledger) ReplayTombstones() ([]string, error) {
	proof, err := l.Verify("", time.Time{})
	if err != nil {
		return nil, err
	}
	return append([]string(nil), proof.Erasures...), nil
}

func (l *Ledger) Head() (uint64, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	records, err := l.readRecords()
	if err != nil {
		return 0, "", err
	}
	if len(records) == 0 {
		return 0, strings.Repeat("0", 64), nil
	}
	return records[len(records)-1].Seq, records[len(records)-1].Hash, nil
}

func (l *Ledger) Close() error { return nil }
