// Package canonical provides the versioned canonical game-state projection, its
// RFC 8785 (JCS) serialization and the SHA-256 state hash.
//
// It contains no game logic: it only reads an already validated authoritative
// state. Serialization never uses floats, map iteration order or locale-dependent
// formatting, so identical logical states produce identical bytes on every host.
package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// ErrInvalidState reports a state that cannot be canonically projected.
var ErrInvalidState = errors.New("canonical: invalid state")

// SchemaVersion is the canonical projection version covered by the hash.
const SchemaVersion = domain.CanonicalSchemaVersion

// maxSafeInteger is the largest magnitude representable exactly by an IEEE-754
// double (2^53-1). The restricted JCS profile rejects larger integers rather than
// silently losing precision; the binding contract reserves decimal strings for
// such fields, and no current canonical field requires one.
const maxSafeInteger int64 = 1<<53 - 1

// Project returns the canonical, versioned game-state representation. Fields are
// exactly the documented binding set; no timestamps, device or provider data are
// included.
func Project(state domain.SessionState) (map[string]any, error) {
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	if state.Revision.RulesVersion != domain.RulesVersion || state.Revision.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%w: unsupported revision %+v", ErrInvalidState, state.Revision)
	}
	for index := 1; index < len(state.Members); index++ {
		if state.Members[index-1].ActorKey >= state.Members[index].ActorKey {
			return nil, fmt.Errorf("%w: members are not canonically ordered", ErrInvalidState)
		}
	}

	sessionXP := make(map[domain.ActorKey]int64, len(state.XP))
	for _, row := range state.XP {
		sessionXP[row.ActorKey] = row.XP
	}

	members := make([]any, 0, len(state.Members))
	for _, member := range state.Members {
		var lastChange any
		if member.LastChangeTick != nil {
			lastChange = int64(*member.LastChangeTick)
		}
		members = append(members, map[string]any{
			"actor_key":         string(member.ActorKey),
			"role":              member.Role.String(),
			"joined_tick":       int64(member.JoinedTick),
			"last_change_tick":  lastChange,
			"first_switch_used": member.FirstSwitchUsed,
			"xp_awarded":        member.XPAwarded,
			"session_xp":        sessionXP[member.ActorKey],
		})
	}

	// Binding contract: exactly west, center, east, each with runner/grunt/brute HP.
	laneHP := [domain.LaneCount][3]int64{}
	laneSeen := [domain.LaneCount][3]bool{}
	if len(state.Wave.Pools) != 0 && len(state.Wave.Pools) != domain.LaneCount*3 {
		return nil, fmt.Errorf("%w: lane pools are not the canonical nine subpools", ErrInvalidState)
	}
	for _, pool := range state.Wave.Pools {
		lane, archetype := int(pool.Lane), int(pool.Archetype)
		if lane < 0 || lane >= domain.LaneCount || archetype < 0 || archetype >= 3 {
			return nil, fmt.Errorf("%w: invalid pool identity", ErrInvalidState)
		}
		if laneSeen[lane][archetype] {
			return nil, fmt.Errorf("%w: duplicate pool identity", ErrInvalidState)
		}
		laneSeen[lane][archetype] = true
		laneHP[lane][archetype] = pool.HP
	}
	pools := make([]any, 0, domain.LaneCount)
	for lane := 0; lane < domain.LaneCount; lane++ {
		pools = append(pools, map[string]any{
			"lane":      domain.Lane(lane).String(),
			"runner_hp": laneHP[lane][int(domain.ArchetypeRunner)],
			"grunt_hp":  laneHP[lane][int(domain.ArchetypeGrunt)],
			"brute_hp":  laneHP[lane][int(domain.ArchetypeBrute)],
		})
	}

	queued := make([]any, 0, len(state.Pending))
	for _, pending := range state.Pending {
		var lastChange any
		if pending.LastChangeTick != nil {
			lastChange = int64(*pending.LastChangeTick)
		}
		queued = append(queued, map[string]any{
			"actor_key":         string(pending.ActorKey),
			"role":              pending.Role.String(),
			"target_wave":       pending.TargetWave,
			"last_change_tick":  lastChange,
			"first_switch_used": pending.FirstSwitchUsed,
		})
	}

	var selected any
	if state.Wave.Vote.Selected != domain.VoteOptionNone {
		selected = state.Wave.Vote.Selected.String()
	}

	return map[string]any{
		"schema_version":     int64(SchemaVersion),
		"rules_version":      state.Revision.RulesVersion,
		"session_id":         string(state.ID),
		"village_id":         string(state.VillageID),
		"wave_no":            state.Wave.Number,
		"wave_tick":          int64(state.Wave.Tick),
		"lifecycle":          state.Lifecycle.String(),
		"defeated":           state.Wave.Defeated,
		"wall_tier":          int64(state.Village.WallTier),
		"wall_hp":            state.Village.WallHP,
		"townhall_hp":        state.Village.TownhallHP,
		"wood":               state.Village.Resources.Wood,
		"stone":              state.Village.Resources.Stone,
		"food":               state.Village.Resources.Food,
		"construction_work":  state.Village.ConstructionWork,
		"total_wins":         state.Village.TotalWins,
		"consecutive_wins":   state.ConsecutiveWins,
		"consecutive_losses": state.ConsecutiveLosses,
		"anchor_population":  int64(state.Difficulty.PopulationAnchor),
		"difficulty":         int64(state.Difficulty.Level),
		"base_hp":            state.Difficulty.BaseHP,
		"lane_pools":         pools,
		"boss_spawned":       state.Wave.Boss.Exists,
		"boss_hp":            state.Wave.Boss.HP,
		"vote": map[string]any{
			"open":     state.Wave.Vote.Open,
			"closed":   state.Wave.Vote.Closed,
			"selected": selected,
			"counts": map[string]any{
				"a": state.Wave.Vote.Counts.A,
				"b": state.Wave.Vote.Counts.B,
				"c": state.Wave.Vote.Counts.C,
			},
		},
		"members":          members,
		"queued_next_wave": queued,
		"privacy_epoch":    state.PrivacyEpoch,
	}, nil
}

// MarshalJCS serializes the projection as RFC 8785 canonical JSON bytes.
func MarshalJCS(value any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := writeValue(&buffer, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// SHA256Hex returns the lowercase hex SHA-256 of raw bytes.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Hash returns the hex SHA-256 of the canonical JSON projection.
func Hash(state domain.SessionState) (string, error) {
	projection, err := Project(state)
	if err != nil {
		return "", err
	}
	encoded, err := MarshalJCS(projection)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func writeValue(buffer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		buffer.WriteString("null")
	case bool:
		if typed {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
	case int:
		if int64(typed) > maxSafeInteger || int64(typed) < -maxSafeInteger {
			return fmt.Errorf("canonical: integer outside the JSON safe range")
		}
		buffer.WriteString(strconv.FormatInt(int64(typed), 10))
	case int64:
		if typed > maxSafeInteger || typed < -maxSafeInteger {
			return fmt.Errorf("canonical: integer %d outside the JSON safe range", typed)
		}
		buffer.WriteString(strconv.FormatInt(typed, 10))
	case string:
		encoded, err := encodeString(typed)
		if err != nil {
			return err
		}
		buffer.WriteString(encoded)
	case []any:
		buffer.WriteByte('[')
		for index, element := range typed {
			if index > 0 {
				buffer.WriteByte(',')
			}
			if err := writeValue(buffer, element); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buffer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				buffer.WriteByte(',')
			}
			encodedKey, err := encodeString(key)
			if err != nil {
				return err
			}
			buffer.WriteString(encodedKey)
			buffer.WriteByte(':')
			if err := writeValue(buffer, typed[key]); err != nil {
				return err
			}
		}
		buffer.WriteByte('}')
	default:
		return fmt.Errorf("canonical: unsupported JCS value type %T", value)
	}
	return nil
}

// encodeString uses Go's JSON string encoder without HTML escaping, matching the
// RFC 8785 escaping rules for the restricted ASCII identifiers in this schema.
func encodeString(value string) (string, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("canonical: string encoding: %w", err)
	}
	encoded := buffer.String()
	if len(encoded) > 0 && encoded[len(encoded)-1] == '\n' {
		encoded = encoded[:len(encoded)-1]
	}
	return encoded, nil
}
