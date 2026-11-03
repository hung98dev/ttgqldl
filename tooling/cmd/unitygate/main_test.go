package main

import (
	"strings"
	"testing"
)

const (
	repo       = "hung98dev/ttgqldl"
	currentSHA = "92bd5df28cc791670bbc41a55caf94f080b2c339"
	staleSHA   = "c69ae5549f7fcad5d17fe12e53a3dea6436fafb7"
	unrelated  = "1111111111111111111111111111111111111111"
)

func base() Input {
	return Input{
		RunnerEnabled:      "true",
		EnvironmentReady:   "true",
		Ref:                "refs/heads/main",
		Repository:         repo,
		ExpectedRepository: repo,
		RequestedSHA:       currentSHA,
		CheckoutSHA:        currentSHA,
		RemoteMainSHA:      currentSHA,
	}
}

func TestExactCurrentHeadIsAccepted(t *testing.T) {
	result := evaluate(base())
	if !result.OK {
		t.Fatalf("current HEAD rejected: %v", result.Reasons)
	}
	if result.SHA != currentSHA {
		t.Fatalf("accepted SHA = %q", result.SHA)
	}
}

func TestStaleAncestorIsRejected(t *testing.T) {
	in := base()
	in.RequestedSHA = staleSHA
	in.CheckoutSHA = staleSHA
	in.RemoteMainSHA = currentSHA
	result := evaluate(in)
	if result.OK {
		t.Fatal("stale ancestor was accepted; ancestry is not main HEAD")
	}
	if !contains(result.Reasons, "stale ancestors are rejected") {
		t.Fatalf("missing stale-ancestor reason: %v", result.Reasons)
	}
}

func TestUnrelatedSHAIsRejected(t *testing.T) {
	in := base()
	in.RequestedSHA = unrelated
	in.CheckoutSHA = unrelated
	in.RemoteMainSHA = currentSHA
	if evaluate(in).OK {
		t.Fatal("unrelated SHA was accepted")
	}
}

func TestInvalidSHAIsRejected(t *testing.T) {
	for _, bad := range []string{"", "abc", strings.ToUpper(currentSHA), currentSHA + "x", "main"} {
		in := base()
		in.RequestedSHA = bad
		in.CheckoutSHA = bad
		in.RemoteMainSHA = bad
		if evaluate(in).OK {
			t.Fatalf("invalid SHA %q was accepted", bad)
		}
	}
}

func TestNonMainRefIsRejected(t *testing.T) {
	for _, ref := range []string{"refs/heads/feature", "refs/tags/v1", "refs/pull/1/merge", ""} {
		in := base()
		in.Ref = ref
		if evaluate(in).OK {
			t.Fatalf("ref %q was accepted", ref)
		}
	}
}

func TestDisabledRunnerFlagIsRejected(t *testing.T) {
	in := base()
	in.RunnerEnabled = ""
	if evaluate(in).OK {
		t.Fatal("disabled runner flag was accepted")
	}
	in = base()
	in.RunnerEnabled = "false"
	if evaluate(in).OK {
		t.Fatal("runner flag 'false' was accepted")
	}
}

func TestUnconfirmedEnvironmentIsRejected(t *testing.T) {
	in := base()
	in.EnvironmentReady = ""
	if evaluate(in).OK {
		t.Fatal("unconfirmed protected environment was accepted")
	}
}

func TestWrongRepositoryIsRejected(t *testing.T) {
	in := base()
	in.Repository = "attacker/ttgqldl"
	in.ExpectedRepository = repo
	result := evaluate(in)
	if result.OK {
		t.Fatal("foreign repository was accepted")
	}
	if !contains(result.Reasons, "does not match expected") {
		t.Fatalf("missing repository reason: %v", result.Reasons)
	}
}

func TestEveryReasonIsReported(t *testing.T) {
	result := evaluate(Input{})
	if result.OK {
		t.Fatal("empty input was accepted")
	}
	if len(result.Reasons) < 5 {
		t.Fatalf("expected every failing rule to be reported, got %d: %v", len(result.Reasons), result.Reasons)
	}
}

func TestResultIsDeterministicApartFromTimestamp(t *testing.T) {
	first := evaluate(base())
	second := evaluate(base())
	if first.OK != second.OK || first.SHA != second.SHA {
		t.Fatal("two evaluations of the same input disagree")
	}
	if first.CheckedAt == "" {
		t.Fatal("checkedAt must be populated")
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
