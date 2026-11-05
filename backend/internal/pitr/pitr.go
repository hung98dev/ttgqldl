// Package pitr contains backup-manifest, encryption, retention and privacy-gate
// primitives used by the W0405 disaster-recovery rehearsal. It never talks to a live
// database by itself; the integration rehearsal supplies the isolated PostgreSQL.
package pitr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const (
	BackupFormat       = "ttg-pitr-backup-v1"
	EncryptionFormat   = "aes-256-gcm"
	KeyReferencePrefix = "env:TTG_PITR_KEY_FILE"
)

var (
	ErrManifestInvalid = errors.New("pitr: manifest invalid")
	ErrDecrypt         = errors.New("pitr: backup decryption failed")
	ErrIntegrity       = errors.New("pitr: backup integrity failed")
	ErrRetention       = errors.New("pitr: unsafe retention plan")
	ErrPrivacyGate     = errors.New("pitr: external deletion ledger is unavailable")
)

type Manifest struct {
	Format           string            `json:"format"`
	BackupID         string            `json:"backup_id"`
	PostgresVersion  string            `json:"postgres_version"`
	SystemIdentifier string            `json:"system_identifier"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	StartWAL         string            `json:"start_wal"`
	EndWAL           string            `json:"end_wal"`
	Timeline         int               `json:"timeline"`
	BackupBytes      int64             `json:"backup_bytes"`
	CiphertextSHA256 string            `json:"ciphertext_sha256"`
	PlaintextSHA256  string            `json:"plaintext_sha256"`
	Encryption       string            `json:"encryption"`
	KeyReference     string            `json:"key_reference"`
	ToolVersions     map[string]string `json:"tool_versions"`
	Verified         bool              `json:"verified"`
	RestoreEligible  bool              `json:"restore_eligible"`
	Incomplete       bool              `json:"incomplete"`
}

func (m Manifest) Validate() error {
	if m.Format != BackupFormat || m.BackupID == "" || m.PostgresVersion == "" ||
		m.SystemIdentifier == "" || m.StartWAL == "" || m.EndWAL == "" || m.Timeline < 1 ||
		m.Encryption != EncryptionFormat || m.KeyReference == "" ||
		m.CiphertextSHA256 == "" || m.PlaintextSHA256 == "" {
		return ErrManifestInvalid
	}
	if m.EndTime.Before(m.StartTime) || m.BackupBytes <= 0 {
		return ErrManifestInvalid
	}
	if m.Incomplete || !m.Verified || !m.RestoreEligible {
		return ErrManifestInvalid
	}
	return nil
}

func EncryptFile(plaintextPath, ciphertextPath string, key []byte) (Manifest, error) {
	if len(key) != 32 {
		return Manifest{}, ErrDecrypt
	}
	plaintext, err := os.ReadFile(plaintextPath) // #nosec G304 -- caller supplies a validated backup path
	if err != nil {
		return Manifest{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Manifest{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Manifest{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Manifest{}, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	out := append(nonce, sealed...)
	if err := os.WriteFile(ciphertextPath, out, 0o600); err != nil { // #nosec G703 -- caller supplies a validated backup path
		return Manifest{}, err
	}
	plainSum := sha256.Sum256(plaintext)
	cipherSum := sha256.Sum256(out)
	now := time.Now().UTC()
	return Manifest{
		Format: BackupFormat, PostgresVersion: "18.6", StartTime: now, EndTime: now,
		BackupBytes: int64(len(out)), CiphertextSHA256: hex.EncodeToString(cipherSum[:]),
		PlaintextSHA256: hex.EncodeToString(plainSum[:]), Encryption: EncryptionFormat,
		KeyReference: KeyReferencePrefix, ToolVersions: map[string]string{"pitr": "1"},
		Verified: false, RestoreEligible: false,
	}, nil
}

func DecryptFile(ciphertextPath, plaintextPath string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrDecrypt
	}
	raw, err := os.ReadFile(ciphertextPath) // #nosec G304 -- caller supplies a validated backup path
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, ErrDecrypt
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	if err := os.WriteFile(plaintextPath, plaintext, 0o600); err != nil { // #nosec G703 -- caller supplies a validated restore path
		return nil, err
	}
	return plaintext, nil
}

func VerifyCiphertext(path string, manifest Manifest) error {
	raw, err := os.ReadFile(path) // #nosec G304 -- caller supplies a validated manifest path
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != manifest.CiphertextSHA256 || int64(len(raw)) != manifest.BackupBytes {
		return ErrIntegrity
	}
	return nil
}

func WriteManifest(path string, manifest Manifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o600) // #nosec G703 -- caller supplies a validated manifest path
}

func ReadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- caller supplies a validated manifest path
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, ErrManifestInvalid
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

type RestorePoint struct {
	BackupID      string    `json:"backup_id"`
	CreatedAt     time.Time `json:"created_at"`
	Timeline      int       `json:"timeline"`
	Verified      bool      `json:"verified"`
	WALComplete   bool      `json:"wal_complete"`
	RequiredWAL   []string  `json:"required_wal"`
	RestoreTarget time.Time `json:"restore_target"`
}

// PlanRetention keeps at least seven verified, WAL-complete restore points and never
// deletes WAL required by a retained point. It returns a dry-run deletion plan.
func PlanRetention(points []RestorePoint, keep int) ([]RestorePoint, []string, error) {
	if keep < 7 {
		return nil, nil, fmt.Errorf("%w: keep must be >= 7", ErrRetention)
	}
	valid := make([]RestorePoint, 0, len(points))
	for _, point := range points {
		if point.Verified && point.WALComplete && point.BackupID != "" && point.Timeline >= 1 {
			valid = append(valid, point)
		}
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].CreatedAt.After(valid[j].CreatedAt) })
	if len(valid) < keep {
		return nil, nil, fmt.Errorf("%w: %d valid points, need %d", ErrRetention, len(valid), keep)
	}
	retained := valid[:keep]
	required := make(map[string]struct{})
	for _, point := range retained {
		for _, wal := range point.RequiredWAL {
			required[wal] = struct{}{}
		}
	}
	delete := make([]string, 0)
	for _, point := range valid[keep:] {
		blocked := false
		for _, wal := range point.RequiredWAL {
			if _, ok := required[wal]; ok {
				blocked = true
				break
			}
		}
		if !blocked {
			delete = append(delete, point.BackupID)
		}
	}
	return retained, delete, nil
}

// PrivacyGate is a fail-closed integration point. PostgreSQL privacy tombstones are not
// an independent ledger; W0406 must provide the external ledger and coverage proof.
type PrivacyGate struct {
	LedgerAvailable bool
	LedgerVerified  bool
	CoverageThrough time.Time
	ErasureReplayed bool
}

func (g PrivacyGate) AllowClientServing(restoreTime time.Time) error {
	if !g.LedgerAvailable || !g.LedgerVerified || !g.ErasureReplayed || g.CoverageThrough.Before(restoreTime) {
		return ErrPrivacyGate
	}
	return nil
}
