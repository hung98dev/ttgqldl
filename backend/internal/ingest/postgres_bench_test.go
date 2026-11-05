package ingest

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

func benchPool(b *testing.B) *storage.Pool {
	b.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		b.Skip("TTG_TEST_DATABASE_URL is not set")
	}
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(pool.Close)
	return pool
}

func BenchmarkDurableAppendBatchSizes(b *testing.B) {
	pool := benchPool(b)
	session := createSession(b, pool, "mock", "running", 1, 200, 0)
	appender, err := NewPostgresAppender(pool)
	if err != nil {
		b.Fatal(err)
	}
	actor := actorFor(30)
	action := mustActionB(b, "XÂY")
	for _, size := range []int{1, 32, 128, 256} {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				candidates := make([]DurableCandidate, size)
				for index := range candidates {
					candidates[index] = candidateFor(session.ID, fmt.Sprintf("bench-%d-%d-%d", size, iteration, index), actor, action)
				}
				if _, err := appender.AppendSession(context.Background(), session.ID, candidates); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*size), "ns/input")
		})
	}
}

func BenchmarkDurableDuplicate(b *testing.B) {
	pool := benchPool(b)
	session := createSession(b, pool, "mock", "running", 1, 210, 0)
	appender, err := NewPostgresAppender(pool)
	if err != nil {
		b.Fatal(err)
	}
	candidate := candidateFor(session.ID, "bench-duplicate", actorFor(31), mustActionB(b, "THỦ"))
	if _, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate})
		if err != nil || !result[0].Duplicate {
			b.Fatalf("%+v/%v", result, err)
		}
	}
}

func BenchmarkDurableThousandInputs(b *testing.B) {
	pool := benchPool(b)
	session := createSession(b, pool, "mock", "running", 1, 220, 0)
	appender, err := NewPostgresAppender(pool)
	if err != nil {
		b.Fatal(err)
	}
	actor := actorFor(32)
	action := mustActionB(b, "BUILD")
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		for batch := 0; batch < 4; batch++ {
			size := 256
			if batch == 3 {
				size = 232
			}
			candidates := make([]DurableCandidate, size)
			for index := range candidates {
				candidates[index] = candidateFor(session.ID, fmt.Sprintf("thousand-%d-%d-%d", iteration, batch, index), actor, action)
			}
			if _, err := appender.AppendSession(context.Background(), session.ID, candidates); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*1000), "ns/input")
}

func BenchmarkBatcher256(b *testing.B) {
	pool := benchPool(b)
	session := createSession(b, pool, "mock", "running", 1, 230, 0)
	appender, err := NewPostgresAppender(pool)
	if err != nil {
		b.Fatal(err)
	}
	batcher, err := NewBatcher(appender, BatcherConfig{QueueCapacity: 1024, MaxBatch: 256, FlushInterval: 50 * time.Millisecond})
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = batcher.Close(ctx)
	}()
	actor := actorFor(33)
	action := mustActionB(b, "DEFEND")
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		var wait sync.WaitGroup
		wait.Add(256)
		for index := 0; index < 256; index++ {
			go func(index int) {
				defer wait.Done()
				if _, err := batcher.Submit(context.Background(), candidateFor(session.ID, fmt.Sprintf("batch-bench-%d-%d", iteration, index), actor, action), nil); err != nil {
					b.Errorf("submit: %v", err)
				}
			}(index)
		}
		wait.Wait()
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*256), "ns/input")
}

func TestDurableBatchLatencyPercentiles(t *testing.T) {
	if os.Getenv("TTG_TEST_LATENCY") != "1" {
		t.Skip("latency soak runs only when TTG_TEST_LATENCY=1")
	}
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 240, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(34)
	action := mustAction(t, "ATTACK")
	durations := make([]time.Duration, 0, 100)
	for iteration := 0; iteration < 100; iteration++ {
		candidates := make([]DurableCandidate, 32)
		for index := range candidates {
			candidates[index] = candidateFor(session.ID, fmt.Sprintf("latency-%d-%d", iteration, index), actor, action)
		}
		started := time.Now()
		if _, err := appender.AppendSession(context.Background(), session.ID, candidates); err != nil {
			t.Fatal(err)
		}
		durations = append(durations, time.Since(started))
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	percentile := func(p float64) time.Duration {
		index := int(float64(len(durations))*p) - 1
		if index < 0 {
			index = 0
		}
		return durations[index]
	}
	t.Logf("batch32 commit latency: p50=%s p95=%s p99=%s max=%s", percentile(0.50), percentile(0.95), percentile(0.99), durations[len(durations)-1])
}

func mustActionB(b *testing.B, message string) commands.Action {
	b.Helper()
	action, err := commands.ParseComment(message)
	if err != nil {
		b.Fatal(err)
	}
	return action
}
