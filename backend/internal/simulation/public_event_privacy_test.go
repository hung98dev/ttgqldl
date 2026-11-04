package simulation

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
)

// hexActorKeyPattern matches a scoped 64-hex actor identity inside serialized JSON.
var hexActorKeyPattern = regexp.MustCompile(`[0-9a-f]{64}`)

// TestPublicEventsNeverCarryActorIdentity verifies the M02 public/private split:
// domain events are public-safe and must never expose an actor key, provider
// identifier, alias or comment. The canonical projection and the private checkpoint
// payload do contain actor keys, but they are private artifacts and are never
// published by the simulation package.
func TestPublicEventsNeverCarryActorIdentity(t *testing.T) {
	for name, value := range map[string]any{
		"Event":     Event{},
		"RoleEvent": RoleEvent{},
	} {
		typ := reflect.TypeOf(value)
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			switch tag {
			case "actor_key", "actor", "provider_id", "alias", "comment":
				t.Fatalf("%s.%s carries private identity in its public JSON payload", name, field.Name)
			}
		}
	}

	state, queue := queuedCheckpointFixture(t)
	var events []Event
	roleEvents := make([]RoleEvent, 0, 4)
	for state.Tick() < 1201 {
		next, nextQueue, boundaryEvents, boundaryRoleEvents, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance at %d: %v", state.Tick(), err)
		}
		state, queue = next, nextQueue
		events = append(events, boundaryEvents...)
		roleEvents = append(roleEvents, boundaryRoleEvents...)
	}
	if len(events) == 0 {
		t.Fatal("fixture produced no events")
	}
	if len(roleEvents) == 0 {
		t.Fatal("fixture produced no role events")
	}

	actorKeys := make([]string, 0, len(state.Members))
	for _, member := range state.Members {
		actorKeys = append(actorKeys, string(member.ActorKey))
	}
	check := func(label string, value any) {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		text := string(encoded)
		for _, key := range actorKeys {
			if strings.Contains(text, key) {
				t.Fatalf("%s leaked an actor key: %s", label, text)
			}
		}
		if match := hexActorKeyPattern.FindString(text); match != "" {
			t.Fatalf("%s leaked a 64-hex identity-shaped token: %s", label, text)
		}
	}
	for index, event := range events {
		check("event", event)
		_ = index
	}
	for index, event := range roleEvents {
		check("role event", event)
		_ = index
	}
	// The queued inputs and the private payload do contain actor keys: that is the
	// private side of the same contract, so this test would catch a future change
	// that accidentally moved identity into the public event types only.
	if len(actorKeys) == 0 {
		t.Fatal("fixture has no actor keys")
	}
}

// TestCanonicalProjectionRemainsPrivateNotPublished documents the boundary the
// simulation package is responsible for: canonical hashing consumes the full
// authoritative state, while the only values this package exposes to a public
// consumer are events, which the test above proves carry no identity.
func TestCanonicalProjectionRemainsPrivateNotPublished(t *testing.T) {
	state, _ := queuedCheckpointFixture(t)
	projection, err := canonical.Project(state)
	if err != nil {
		t.Fatalf("canonical projection: %v", err)
	}
	if _, exists := projection["private_digest"]; exists {
		t.Fatal("canonical projection unexpectedly exposes private integrity material")
	}
	// The projection itself is private; this assertion only pins that it is a
	// game-state object and not an event/telemetry payload.
	if _, exists := projection["events"]; exists {
		t.Fatal("canonical projection must not embed events")
	}
}
