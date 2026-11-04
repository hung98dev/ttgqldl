package source

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	replayManifestSchema = 1
	replayRecordSchema   = 1
	maxManifestBytes     = 1 << 20
	// maxRecordBytes bounds one NDJSON line so a hostile fixture cannot force
	// unbounded allocation. Scanner returns bufio.ErrTooLong beyond this.
	maxRecordBytes = 64 * 1024
)

// replayManifest pins fixture bytes and record counts. The SHA256 digest also
// becomes the adapter source identity, so a cursor from one fixture cannot select a
// position in another.
type replayManifest struct {
	Schema   int                  `json:"schema"`
	Fixtures []replayFixtureEntry `json:"fixtures"`
}

type replayFixtureEntry struct {
	Name          string `json:"name"`
	SchemaVersion int    `json:"schema_version"`
	SHA256        string `json:"sha256"`
	Records       int    `json:"records"`
}

// replayRecord is the exact NDJSON record schema. Unknown fields are rejected by
// DisallowUnknownFields; optional fields may be omitted.
type replayRecord struct {
	SchemaVersion    int    `json:"schema_version"`
	SourceSession    string `json:"source_session"`
	EventSeq         int64  `json:"event_seq"`
	Provider         string `json:"provider"`
	ProviderEventID  string `json:"provider_event_id"`
	ProviderViewerID string `json:"provider_viewer_id"`
	Alias            string `json:"alias"`
	Message          string `json:"message"`
	EventKind        string `json:"event_kind"`
	ReceivedAt       string `json:"received_at"`
	UpstreamAt       string `json:"upstream_at"`
}

// ReplayConfig selects one committed fixture from a fixture directory.
type ReplayConfig struct {
	Session   Session
	Directory string
	Fixture   string
}

// Replay is a deterministic, network-free NDJSON source. It streams records instead
// of loading the fixture, verifies the pinned digest before delivery and never falls
// back to another source.
type Replay struct {
	session   Session
	directory string
	fixture   string
	sourceID  string
	records   int
	// open is overridable in tests to inject reader failures. Production always opens
	// the verified fixture path.
	open func() (io.ReadCloser, error)
	lc   lifecycle
}

// NewReplay loads the manifest, verifies the fixture digest and record count, and
// prepares the adapter. Malformed or unsupported fixtures fail here, before any
// delivery.
func NewReplay(config ReplayConfig) (*Replay, error) {
	if config.Session.Mode != ModeReplay {
		return nil, fmt.Errorf("%w: replay requires replay mode", ErrInvalidSession)
	}
	if !config.Session.Valid() {
		return nil, ErrInvalidSession
	}
	if config.Fixture == "" || filepath.Base(config.Fixture) != config.Fixture || strings.Contains(config.Fixture, "..") {
		return nil, fmt.Errorf("%w: fixture name must be a plain file name", ErrInvalidFixture)
	}
	manifestPath, err := resolveFixturePath(config.Directory, "manifest.json")
	if err != nil {
		return nil, err
	}
	manifest, err := loadReplayManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	entry, ok := manifest.fixture(config.Fixture)
	if !ok {
		return nil, fmt.Errorf("%w: fixture %q is not listed in the manifest", ErrInvalidFixture, config.Fixture)
	}
	if entry.SchemaVersion != replayRecordSchema {
		return nil, fmt.Errorf("%w: fixture schema %d", ErrSchemaUnsupported, entry.SchemaVersion)
	}
	path, err := resolveFixturePath(config.Directory, entry.Name)
	if err != nil {
		return nil, err
	}
	digest, records, err := inspectFixture(path)
	if err != nil {
		return nil, err
	}
	if digest != entry.SHA256 {
		return nil, fmt.Errorf("%w: fixture bytes do not match the manifest", ErrFixtureDigest)
	}
	if records != entry.Records {
		return nil, fmt.Errorf("%w: fixture record count %d does not match the manifest", ErrInvalidFixture, records)
	}
	replay := &Replay{
		session:   config.Session,
		directory: config.Directory,
		fixture:   entry.Name,
		sourceID:  digest,
		records:   records,
	}
	// #nosec G304 -- path is resolved and confined to the operator-configured fixture
	// directory; entry.Name is a plain manifest-declared base name.
	replay.open = func() (io.ReadCloser, error) { return os.Open(path) }
	return replay, nil
}

// resolveFixturePath joins a plain fixture name to the configured directory and
// verifies the result stays inside that directory. It rejects traversal and absolute
// names before any file operation.
func resolveFixturePath(directory, name string) (string, error) {
	if name == "" || filepath.Base(name) != name || strings.Contains(name, "..") {
		return "", fmt.Errorf("%w: fixture name must be a plain file name", ErrInvalidFixture)
	}
	base, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("%w: fixture directory", ErrInvalidFixture)
	}
	candidate := filepath.Join(base, name)
	relative, err := filepath.Rel(base, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: fixture path escapes the fixture directory", ErrInvalidFixture)
	}
	return candidate, nil
}

// SourceID returns the digest-bound adapter identity used by cursors.
func (r *Replay) SourceID() string { return r.sourceID }

// Records returns the verified record count.
func (r *Replay) Records() int { return r.records }

func (m replayManifest) fixture(name string) (replayFixtureEntry, bool) {
	for _, entry := range m.Fixtures {
		if entry.Name == name {
			return entry, true
		}
	}
	return replayFixtureEntry{}, false
}

func loadReplayManifest(path string) (replayManifest, error) {
	// #nosec G304 -- path is resolved and confined by the caller before this call.
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxManifestBytes {
		return replayManifest{}, fmt.Errorf("%w: manifest is missing or too large", ErrInvalidFixture)
	}
	// #nosec G304 -- path is resolved and confined by the caller before this call.
	data, err := os.ReadFile(path)
	if err != nil {
		return replayManifest{}, fmt.Errorf("%w: manifest cannot be read", ErrInvalidFixture)
	}
	var manifest replayManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return replayManifest{}, fmt.Errorf("%w: manifest is malformed", ErrInvalidFixture)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return replayManifest{}, fmt.Errorf("%w: manifest has trailing data", ErrInvalidFixture)
	}
	if manifest.Schema != replayManifestSchema {
		return replayManifest{}, fmt.Errorf("%w: manifest schema %d", ErrSchemaUnsupported, manifest.Schema)
	}
	seen := make(map[string]bool, len(manifest.Fixtures))
	for _, entry := range manifest.Fixtures {
		if entry.Name == "" || filepath.Base(entry.Name) != entry.Name || seen[entry.Name] {
			return replayManifest{}, fmt.Errorf("%w: invalid or duplicate fixture name", ErrInvalidFixture)
		}
		seen[entry.Name] = true
		if entry.SchemaVersion != replayRecordSchema {
			return replayManifest{}, fmt.Errorf("%w: fixture %q schema %d", ErrSchemaUnsupported, entry.Name, entry.SchemaVersion)
		}
		if len(entry.SHA256) != 64 {
			return replayManifest{}, fmt.Errorf("%w: fixture %q digest", ErrInvalidFixture, entry.Name)
		}
		if _, err := hex.DecodeString(entry.SHA256); err != nil {
			return replayManifest{}, fmt.Errorf("%w: fixture %q digest is not hex", ErrInvalidFixture, entry.Name)
		}
		if entry.Records < 0 {
			return replayManifest{}, fmt.Errorf("%w: fixture %q record count", ErrInvalidFixture, entry.Name)
		}
	}
	return manifest, nil
}

// inspectFixture hashes and counts one fixture in a bounded streaming pass. It never
// loads the whole file.
func inspectFixture(path string) (string, int, error) {
	// #nosec G304 -- path is resolved and confined by the caller before this call.
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("%w: fixture cannot be opened", ErrInvalidFixture)
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	records := 0
	var last byte
	var total int
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
			for _, b := range buffer[:n] {
				if b == '\n' {
					records++
				}
			}
			for _, b := range buffer[:n] {
				last = b
			}
			total += n
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", 0, fmt.Errorf("%w: fixture read failure", ErrInvalidFixture)
		}
	}
	if total > 0 && last != '\n' {
		records++
	}
	return hex.EncodeToString(hash.Sum(nil)), records, nil
}

// RecordError adds the deterministic fixture location to a rejection without echoing
// record content, provider identifiers or raw bytes.
type RecordError struct {
	Line   int
	Record int
	Err    error
}

func (e *RecordError) Error() string {
	return fmt.Sprintf("source: replay record %d (line %d): %s", e.Record, e.Line, errorCategory(e.Err))
}

func (e *RecordError) Unwrap() error { return e.Err }

func recordError(line, record int, err error) error {
	var existing *RecordError
	if errors.As(err, &existing) {
		return existing
	}
	return &RecordError{Line: line, Record: record, Err: err}
}

// decodeReplayRecord decodes exactly one record and applies the source trust
// boundary. Unknown fields, unknown schema versions, duplicate/out-of-order
// sequences, session mismatches and invalid metadata fail with typed errors.
func decodeReplayRecord(line []byte, session Session, expectedSeq int64) (Event, error) {
	if len(line) == 0 {
		return Event{}, fmt.Errorf("%w: empty record", ErrMalformedRecord)
	}
	if !utf8.Valid(line) {
		return Event{}, ErrRecordEncoding
	}
	var record replayRecord
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Event{}, fmt.Errorf("%w: JSON syntax", ErrMalformedRecord)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Event{}, fmt.Errorf("%w: trailing JSON", ErrMalformedRecord)
	}
	if record.SchemaVersion != replayRecordSchema {
		return Event{}, fmt.Errorf("%w: record schema %d", ErrSchemaUnsupported, record.SchemaVersion)
	}
	if record.EventSeq != expectedSeq {
		return Event{}, fmt.Errorf("%w: duplicate or out-of-order event_seq", ErrMalformedRecord)
	}
	if record.Provider != ModeReplay.Provider() {
		return Event{}, ErrProviderMismatch
	}
	if record.SourceSession != session.ID {
		return Event{}, ErrSessionMismatch
	}
	kind, err := ParseEventKind(record.EventKind)
	if err != nil {
		return Event{}, fmt.Errorf("%w: record event kind", ErrEventKindInvalid)
	}
	var received time.Time
	if record.ReceivedAt != "" {
		received, err = time.Parse(time.RFC3339Nano, record.ReceivedAt)
		if err != nil {
			return Event{}, fmt.Errorf("%w: received_at", ErrMalformedRecord)
		}
	}
	var upstream *time.Time
	if record.UpstreamAt != "" {
		stamp, err := time.Parse(time.RFC3339Nano, record.UpstreamAt)
		if err != nil {
			return Event{}, fmt.Errorf("%w: upstream_at", ErrMalformedRecord)
		}
		upstream = &stamp
	}
	event := Event{
		Provider:         record.Provider,
		SourceSession:    record.SourceSession,
		ProviderEventID:  record.ProviderEventID,
		ProviderViewerID: record.ProviderViewerID,
		Alias:            record.Alias,
		Message:          record.Message,
		Kind:             kind,
		ReceivedAt:       received,
		UpstreamAt:       upstream,
	}
	if err := ValidateEvent(event, session, ModeReplay); err != nil {
		return Event{}, err
	}
	return event, nil
}

// Start streams the fixture from the beginning.
func (r *Replay) Start(ctx context.Context, session Session, sink Sink) error {
	return r.run(ctx, session, Cursor{}, sink, false)
}

// Stop cancels delivery and waits for the emission loop to exit.
func (r *Replay) Stop(ctx context.Context) error { return r.lc.stop(ctx) }

// Health returns the concurrency-safe adapter health.
func (r *Replay) Health(context.Context) Health { return r.lc.health() }

// Capabilities returns the truthful replay capability set. Identifiers are stable
// only because the committed fixture declares them; they are not authorized provider
// identities.
func (r *Replay) Capabilities(context.Context) Capabilities {
	return Capabilities{
		StableViewerID:       true,
		StableEventID:        true,
		ReplayCursor:         true,
		DeliveryOrder:        true,
		TimestampReliability: TimestampRecorded,
		DisplayName:          true,
		ModerationEvents:     false,
	}
}

// Reconnect resumes at the exact remaining fixture position.
func (r *Replay) Reconnect(ctx context.Context, cursor Cursor, sink Sink) error {
	return r.run(ctx, r.session, cursor, sink, true)
}

func (r *Replay) run(ctx context.Context, session Session, cursor Cursor, sink Sink, reconnect bool) error {
	if session != r.session {
		return ErrSessionMismatch
	}
	start := 0
	if reconnect {
		if err := validateCursorBinding(cursor, session, r.sourceID); err != nil {
			return err
		}
		// #nosec G115 -- r.records is a verified non-negative record count.
		if cursor.NextEvent > uint64(r.records) {
			return fmt.Errorf("%w: position %d beyond %d records", ErrInvalidCursor, cursor.NextEvent, r.records)
		}
		// #nosec G115 -- the bound above proves the value fits in int.
		start = int(cursor.NextEvent)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := r.lc.begin(runCtx, cancel, session, r.sourceID, uint64(start), reconnect); err != nil {
		return err
	}
	status, runErr := StatusEOF, error(nil)
	reader, err := r.open()
	if err != nil {
		status, runErr = StatusFailed, fmt.Errorf("%w: fixture cannot be opened", ErrInvalidFixture)
		r.lc.finish(status, runErr)
		return runErr
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4*1024), maxRecordBytes)
	recordIndex := 0
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		event, decodeErr := decodeReplayRecord(line, session, int64(recordIndex+1))
		if decodeErr != nil {
			status, runErr = StatusFailed, recordError(lineNumber, recordIndex+1, decodeErr)
			break
		}
		if recordIndex < start {
			if r.lc.stopping() {
				status, runErr = StatusStopped, nil
				break
			}
			if err := runCtx.Err(); err != nil {
				status, runErr = StatusDisconnected, err
				break
			}
			recordIndex++
			continue
		}
		if r.lc.stopping() {
			status, runErr = StatusStopped, nil
			break
		}
		if err := runCtx.Err(); err != nil {
			status, runErr = StatusDisconnected, err
			break
		}
		if deliverErr := r.lc.deliver(runCtx, sink, event, uint64(recordIndex+1)); deliverErr != nil {
			switch {
			case errors.Is(deliverErr, ErrSourceStopped):
				status, runErr = StatusStopped, nil
			case errors.Is(deliverErr, context.Canceled), errors.Is(deliverErr, context.DeadlineExceeded):
				status, runErr = StatusDisconnected, deliverErr
			case errors.Is(deliverErr, ErrDeliveryRejected):
				status, runErr = StatusDisconnected, deliverErr
			default:
				status, runErr = StatusFailed, deliverErr
			}
			break
		}
		recordIndex++
	}
	if runErr == nil && status == StatusEOF {
		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				status, runErr = StatusFailed, recordError(lineNumber+1, recordIndex+1, ErrRecordTooLong)
			} else {
				status, runErr = StatusFailed, fmt.Errorf("%w: fixture read failure", ErrInvalidFixture)
			}
		} else if recordIndex != r.records {
			status, runErr = StatusFailed, fmt.Errorf("%w: fixture record count changed after verification", ErrInvalidFixture)
		}
	}
	r.lc.finish(status, runErr)
	return runErr
}
