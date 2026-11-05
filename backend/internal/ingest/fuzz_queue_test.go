package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
)

type fuzzOutcomeAppender struct {
	outcomes []byte
	calls    int
}

func (a *fuzzOutcomeAppender) AppendSession(_ context.Context, _ string, candidates []DurableCandidate) ([]DurableResult, error) {
	code := byte(0)
	if len(a.outcomes) > 0 {
		code = a.outcomes[a.calls%len(a.outcomes)]
	}
	a.calls++
	switch code % 6 {
	case 0:
		results := make([]DurableResult, len(candidates))
		for index := range results {
			results[index] = DurableResult{InputSeq: int64(index + 1)}
		}
		return results, nil
	case 1:
		results := make([]DurableResult, len(candidates))
		for index := range results {
			results[index] = DurableResult{Duplicate: true, InputSeq: 1}
		}
		return results, nil
	case 2:
		results := make([]DurableResult, len(candidates))
		for index := range results {
			results[index] = DurableResult{Conflict: true}
		}
		return results, nil
	case 3:
		return nil, ErrStorageUnavailable
	case 4:
		return nil, ErrCommitUnknown
	default:
		return nil, ErrRetryExhausted
	}
}

func (a *fuzzOutcomeAppender) LookupReceipt(context.Context, string, string, string) (DurableResult, bool, error) {
	return DurableResult{}, false, nil
}

func (a *fuzzOutcomeAppender) LookupCandidate(context.Context, DurableCandidate) (DurableResult, bool, error) {
	return DurableResult{}, false, nil
}

// FuzzBatcherOutcomes mutates bounded fake-appender outcomes through the real Batcher.
// It asserts no panic, no false durable success, no simultaneous duplicate+conflict,
// bounded batch size and no acknowledgement of an error as success.
func FuzzBatcherOutcomes(f *testing.F) {
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Add([]byte{4, 4, 4, 4})
	f.Add([]byte{2, 2, 2, 2})
	f.Add([]byte{5, 5, 5})
	f.Fuzz(func(t *testing.T, outcomes []byte) {
		if len(outcomes) > 16 {
			outcomes = outcomes[:16]
		}
		batcher, err := NewBatcher(&fuzzOutcomeAppender{outcomes: outcomes}, BatcherConfig{
			QueueCapacity: 16, MaxBatch: 4, FlushInterval: time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		action, err := commands.ParseComment("XÂY")
		if err != nil {
			t.Fatal(err)
		}
		submitted := 0
		for index := 0; index < 1+len(outcomes)%8; index++ {
			candidate := DurableCandidate{
				SessionID: "s", Provider: "mock", ProviderEventID: "fuzz-" + string(rune('a'+index)),
				ActorKey: actorFor(index + 1), Action: action,
			}
			result, err := batcher.Submit(context.Background(), candidate, nil)
			submitted++
			if err != nil {
				if !errors.Is(err, ErrStorageUnavailable) && !errors.Is(err, ErrCommitUnknown) && !errors.Is(err, ErrRetryExhausted) {
					t.Fatalf("unexpected submit error: %v", err)
				}
				if result != (DurableResult{}) {
					t.Fatalf("error returned a durable result: %+v", result)
				}
				continue
			}
			if result.Duplicate && result.Conflict {
				t.Fatalf("result is both duplicate and conflict: %+v", result)
			}
			if !result.Duplicate && !result.Conflict && result.InputSeq < 1 {
				t.Fatalf("committed result has no sequence: %+v", result)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := batcher.Close(ctx); err != nil {
			t.Fatalf("close: %v", err)
		}
		stats := batcher.Stats()
		if stats.Inputs > int64(submitted) || stats.Batches > int64(submitted) || stats.MaxBatch > 4 {
			t.Fatalf("unbounded batch stats: %+v", stats)
		}
		if stats.QueueDepth != 0 {
			t.Fatalf("queue not drained: %+v", stats)
		}
	})
}
