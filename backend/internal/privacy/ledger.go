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
	"runtime"
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
	ErrLedgerBusy      = errors.New("privacy: deletion ledger is already owned by another writer")
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
	lockPath := filepath.Join(l.Dir, ".append.lock")
	release, err := acquireAppendLock(lockPath)
	if err != nil {
		return LedgerRecord{}, err
	}
	defer release()
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
			if err := l.writeWitness(records); err != nil {
				return LedgerRecord{}, err
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
	if err != nil {
		return LedgerRecord{}, err
	}
	if err := syncDir(dir); err != nil {
		_ = dir.Close()
		return LedgerRecord{}, err
	}
	if err := dir.Close(); err != nil {
		return LedgerRecord{}, err
	}
	records = append(records, record)
	if err := l.writeWitness(records); err != nil {
		return LedgerRecord{}, err
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
	WitnessSeq      uint64    `json:"witness_seq"`
	WitnessHash     string    `json:"witness_hash"`
	VerifiedAt      time.Time `json:"verified_at"`
	CoverageThrough time.Time `json:"coverage_through"`
	TargetIdentity  string    `json:"target_identity"`
	Erasures        []string  `json:"erasures"`
	Verified        bool      `json:"verified"`
	WitnessVerified bool      `json:"witness_verified"`
}

// Witness is the independently durable local high-watermark. It proves what the
// ledger process durably published; it cannot prove that a destroyed primary had no
// later unexported deletion, so callers must treat absent/reconciled witness evidence
// as a blocker.
type Witness struct {
	Version   int       `json:"version"`
	LedgerID  string    `json:"ledger_id"`
	LastSeq   uint64    `json:"last_seq"`
	HeadHash  string    `json:"head_hash"`
	UpdatedAt time.Time `json:"updated_at"`
	AckRef    string    `json:"ack_ref"`
	Auth      string    `json:"auth"`
}

func (l *Ledger) witnessPath() string { return filepath.Join(l.Dir, "head.witness") }

func (l *Ledger) writeWitness(records []LedgerRecord) error {
	head := strings.Repeat("0", 64)
	seq := uint64(0)
	if len(records) > 0 {
		seq = records[len(records)-1].Seq
		head = records[len(records)-1].Hash
	}
	ack := head
	if seq > 0 {
		ack = fmt.Sprintf("%s:%d:%s", l.ID, seq, head)
	}
	witness := Witness{Version: LedgerFormatVersion, LedgerID: l.ID, LastSeq: seq, HeadHash: head, UpdatedAt: time.Now().UTC(), AckRef: ack}
	witness.Auth = l.witnessAuth(witness)
	encoded, err := json.MarshalIndent(witness, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.witnessPath() + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- owner-only ledger root
	if err != nil {
		return err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, l.witnessPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	dir, err := os.Open(l.Dir)
	if err != nil {
		return err
	}
	if err := syncDir(dir); err != nil {
		_ = dir.Close()
		return err
	}
	if err := dir.Close(); err != nil {
		return err
	}
	return nil
}

func (l *Ledger) readWitness() (Witness, error) {
	raw, err := os.ReadFile(l.witnessPath()) // #nosec G304 -- owner-only ledger root
	if err != nil {
		return Witness{}, err
	}
	var witness Witness
	if err := json.Unmarshal(raw, &witness); err != nil {
		return Witness{}, ErrLedgerIntegrity
	}
	if witness.Version != LedgerFormatVersion || witness.LedgerID != l.ID || witness.AckRef == "" || witness.Auth != l.witnessAuth(witness) {
		return Witness{}, ErrLedgerIntegrity
	}
	return witness, nil
}

func (l *Ledger) witnessAuth(w Witness) string {
	payload := fmt.Sprintf("%d|%s|%d|%s|%s|%s", w.Version, w.LedgerID, w.LastSeq, w.HeadHash, w.UpdatedAt.UTC().Format(time.RFC3339Nano), w.AckRef)
	mac := hmac.New(sha256.New, l.chainKey)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

type Snapshot struct {
	Proof     CoverageProof
	Deletions []Deletion
}

// VerifySnapshot verifies and enumerates one immutable ledger head under a single
// synchronization boundary. A concurrent append cannot enter the returned run.
func (l *Ledger) VerifySnapshot(targetIdentity string) (Snapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	proof, deletions, err := l.verifyLocked(targetIdentity)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Proof: proof, Deletions: deletions}, nil
}

func (l *Ledger) Verify(targetIdentity string) (CoverageProof, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	proof, _, err := l.verifyLocked(targetIdentity)
	return proof, err
}

func (l *Ledger) verifyLocked(targetIdentity string) (CoverageProof, []Deletion, error) {
	records, err := l.readRecords()
	if err != nil {
		return CoverageProof{}, nil, err
	}
	witness, err := l.readWitness()
	if err != nil {
		return CoverageProof{}, nil, ErrLedgerIntegrity
	}
	prev := strings.Repeat("0", 64)
	erasures := make([]string, 0, len(records))
	deletions := make([]Deletion, 0, len(records))
	for index, record := range records {
		if record.Seq != uint64(index+1) || record.PrevHash != prev {
			return CoverageProof{}, nil, ErrLedgerSequence
		}
		payload, err := l.decryptPayload(record)
		if err != nil {
			return CoverageProof{}, nil, err
		}
		erasures = append(erasures, record.TombstoneID)
		deletions = append(deletions, Deletion{Seq: record.Seq, Payload: payload})
		prev = record.Hash
	}
	head := prev
	if len(records) > 0 {
		head = records[len(records)-1].Hash
	}
	if witness.LastSeq != uint64(len(records)) || witness.HeadHash != head {
		return CoverageProof{}, nil, ErrLedgerIntegrity
	}
	return CoverageProof{
		LedgerID: l.ID, LastSeq: uint64(len(records)), HeadHash: head, WitnessSeq: witness.LastSeq,
		WitnessHash: witness.HeadHash, VerifiedAt: time.Now().UTC(), CoverageThrough: witness.UpdatedAt,
		TargetIdentity: targetIdentity, Erasures: erasures, Verified: true, WitnessVerified: true,
	}, deletions, nil
}

func (l *Ledger) decryptRecord(record LedgerRecord) error {
	_, err := l.decryptPayload(record)
	return err
}

// AllowClientServing requires a verifier-produced proof, not caller booleans.
func AllowClientServing(proof CoverageProof, requiredHorizon time.Time, requiredErasures []string) error {
	if !proof.Verified || !proof.WitnessVerified || proof.LedgerID == "" || proof.LastSeq == 0 ||
		proof.WitnessSeq != proof.LastSeq || proof.WitnessHash != proof.HeadHash || proof.CoverageThrough.Before(requiredHorizon) {
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
	proof, err := l.Verify("")
	if err != nil {
		return nil, err
	}
	return append([]string(nil), proof.Erasures...), nil
}

type Deletion struct {
	Seq     uint64
	Payload LedgerPayload
}

// Deletions returns authenticated, decrypted deletion payloads in sequence order.
// Plaintext stays in memory only; callers must not log it.
func (l *Ledger) Deletions() ([]Deletion, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	records, err := l.readRecords()
	if err != nil {
		return nil, err
	}
	out := make([]Deletion, 0, len(records))
	for _, record := range records {
		payload, err := l.decryptPayload(record)
		if err != nil {
			return nil, err
		}
		out = append(out, Deletion{Seq: record.Seq, Payload: payload})
	}
	return out, nil
}

func (l *Ledger) decryptPayload(record LedgerRecord) (LedgerPayload, error) {
	block, err := aes.NewCipher(l.encKey)
	if err != nil {
		return LedgerPayload{}, ErrLedgerKey
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return LedgerPayload{}, ErrLedgerKey
	}
	nonce, err := hex.DecodeString(record.Nonce)
	if err != nil {
		return LedgerPayload{}, ErrLedgerIntegrity
	}
	ciphertext, err := hex.DecodeString(record.Ciphertext)
	if err != nil {
		return LedgerPayload{}, ErrLedgerIntegrity
	}
	ad := []byte(l.ID + "|" + strconv.FormatUint(record.Seq, 10) + "|" + record.PrevHash)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return LedgerPayload{}, ErrLedgerIntegrity
	}
	var payload LedgerPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil || payload.TombstoneID != record.TombstoneID {
		return LedgerPayload{}, ErrLedgerIntegrity
	}
	return payload, nil
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
func syncDir(dir *os.File) error {
	err := dir.Sync()
	if err != nil && runtime.GOOS == "windows" && os.IsPermission(err) {
		return nil
	}
	return err
}
func acquireAppendLock(path string) (func(), error) {
	now := time.Now().UTC()
	owner := map[string]any{"pid": os.Getpid(), "started": now.Format(time.RFC3339Nano), "lease_until": now.Add(30 * time.Second).Format(time.RFC3339Nano)}
	payload, _ := json.Marshal(owner)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- owner-only ledger root
	if err == nil {
		_, _ = file.Write(payload)
		_ = file.Sync()
		_ = file.Close()
		return func() { _ = os.Remove(path) }, nil
	}
	raw, readErr := os.ReadFile(path) // #nosec G304 -- owner-only ledger root
	if readErr != nil {
		return nil, ErrLedgerBusy
	}
	var existing struct {
		LeaseUntil time.Time `json:"lease_until"`
	}
	if json.Unmarshal(raw, &existing) != nil || existing.LeaseUntil.After(now) {
		return nil, ErrLedgerBusy
	}
	stale := path + ".stale." + strconv.Itoa(os.Getpid())
	if err := os.Rename(path, stale); err != nil {
		return nil, ErrLedgerBusy
	}
	file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- owner-only ledger root
	if err != nil {
		return nil, ErrLedgerBusy
	}
	_, _ = file.Write(payload)
	_ = file.Sync()
	_ = file.Close()
	return func() { _ = os.Remove(path); _ = os.Remove(stale) }, nil
}
