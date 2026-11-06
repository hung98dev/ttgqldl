package pitr

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestEncryptDecryptManifestAndTamper(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "base.tar")
	enc := filepath.Join(dir, "base.tar.enc")
	out := filepath.Join(dir, "out.tar")
	if err := os.WriteFile(plain, []byte("deterministic physical backup fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := testKey(t)
	manifest, err := EncryptFile(plain, enc, key)
	if err != nil {
		t.Fatal(err)
	}
	manifest.BackupID = "b-1"
	manifest.SystemIdentifier = "sys-1"
	manifest.StartWAL = "0/1000000"
	manifest.EndWAL = "0/2000000"
	manifest.Timeline = 1
	manifest.Verified = true
	manifest.RestoreEligible = true
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyCiphertext(enc, loaded); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecryptFile(enc, out, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "deterministic physical backup fixture" {
		t.Fatal("plaintext mismatch")
	}
	if _, err := DecryptFile(enc, out, testKey(t)); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("wrong key = %v", err)
	}
	raw, _ := os.ReadFile(enc)
	raw[len(raw)-1] ^= 0x01
	if err := os.WriteFile(enc, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptFile(enc, out, key); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("tamper = %v", err)
	}
}

func TestManifestRejectsIncompleteOrCorrupt(t *testing.T) {
	now := time.Now().UTC()
	base := Manifest{Format: BackupFormat, BackupID: "b", PostgresVersion: "18.6", SystemIdentifier: "s",
		StartTime: now, EndTime: now.Add(time.Second), StartWAL: "0/1", EndWAL: "0/2", Timeline: 1,
		BackupBytes: 10, CiphertextSHA256: "aa", PlaintextSHA256: "bb", Encryption: EncryptionFormat,
		KeyReference: KeyReferencePrefix, Verified: true, RestoreEligible: true}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := base
	bad.Incomplete = true
	if err := bad.Validate(); !errors.Is(err, ErrManifestInvalid) {
		t.Fatal("incomplete accepted")
	}
	bad = base
	bad.Encryption = "plain"
	if err := bad.Validate(); !errors.Is(err, ErrManifestInvalid) {
		t.Fatal("plain accepted")
	}
}

func TestRetentionProtectsWALAndRequiresSeven(t *testing.T) {
	now := time.Now().UTC()
	points := make([]RestorePoint, 0, 8)
	for index := 0; index < 7; index++ {
		points = append(points, RestorePoint{BackupID: string(rune('a' + index)), CreatedAt: now.Add(-time.Duration(index) * time.Hour), Timeline: 1, Verified: true, WALComplete: true, RequiredWAL: []string{"WAL" + string(rune('a'+index))}})
	}
	old := RestorePoint{BackupID: "old", CreatedAt: now.Add(-48 * time.Hour), Timeline: 1, Verified: true, WALComplete: true, RequiredWAL: []string{"WALa"}}
	points = append(points, old)
	retained, deletion, err := PlanRetention(points, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 7 {
		t.Fatalf("retained=%d", len(retained))
	}
	if len(deletion) != 0 {
		t.Fatalf("deleted WAL-dependent point: %v", deletion)
	}
	if _, _, err := PlanRetention(points[:6], 7); !errors.Is(err, ErrRetention) {
		t.Fatalf("six points = %v", err)
	}
}
