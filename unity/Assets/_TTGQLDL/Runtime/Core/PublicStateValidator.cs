using System;
using System.Collections.Generic;
using System.Globalization;

namespace Ttgqldl.Core
{
    /// <summary>Outcome of validating a published snapshot before display.</summary>
    public sealed class ValidationResult
    {
        private readonly List<string> _errors;

        private ValidationResult(List<string> errors)
        {
            _errors = errors;
        }

        public bool IsValid => _errors.Count == 0;
        public IReadOnlyList<string> Errors => _errors;

        internal static ValidationResult From(List<string> errors) => new ValidationResult(errors);
    }

    /// <summary>
    /// Rejects null, malformed and out-of-range public state before a single
    /// pixel is drawn. Validation never repairs data: an invalid snapshot is
    /// refused so the UI can show STALE/DISCONNECTED instead of inventing truth.
    /// </summary>
    public static class PublicStateValidator
    {
        public const int MaxSpotlights = 12;
        public const int MaxResource = 1_000_000;
        public const int MaxWallSeconds = 300;
        public const int MaxTownhallHp = 500;
        public const int MaxAliasLength = 64;
        public const int MaxPublicRefLength = 64;
        public const int MaxErrors = 32;

        private static readonly string[] ValidRoles = { "builder", "guard", "fighter" };

        public static ValidationResult Validate(PublicSnapshot snapshot)
        {
            var errors = new List<string>();
            if (snapshot == null)
            {
                errors.Add("snapshot is null");
                return ValidationResult.From(errors);
            }

            void Fail(string message)
            {
                if (errors.Count < MaxErrors)
                {
                    errors.Add(message);
                }
            }

            if (snapshot.SchemaVersion != PublicSnapshot.SupportedSchemaVersion)
            {
                Fail($"unsupported schema_version {snapshot.SchemaVersion.ToString(CultureInfo.InvariantCulture)}");
            }

            if (string.IsNullOrWhiteSpace(snapshot.Revision))
            {
                Fail("revision is empty");
            }

            if (snapshot.WaveNo < 1)
            {
                Fail("wave_no must be >= 1");
            }

            if (!Enum.IsDefined(typeof(VillageLifecycle), snapshot.Lifecycle))
            {
                Fail("lifecycle is not a defined value");
            }

            if (!Enum.IsDefined(typeof(Phase), snapshot.Phase))
            {
                Fail("phase is not a defined value");
            }

            if (!Enum.IsDefined(typeof(SourceStatus), snapshot.SourceStatus))
            {
                Fail("source_status is not a defined value");
            }

            if (snapshot.RemainingSeconds < 0 || snapshot.RemainingSeconds > MaxWallSeconds)
            {
                Fail("remaining_seconds outside 0..300");
            }

            if (snapshot.WallMax <= 0)
            {
                Fail("wall_max must be positive");
            }

            if (snapshot.WallHp < 0 || snapshot.WallHp > snapshot.WallMax)
            {
                Fail("wall_hp outside 0..wall_max");
            }

            if (snapshot.TownhallHp < 0 || snapshot.TownhallHp > MaxTownhallHp)
            {
                Fail("townhall_hp outside 0..500");
            }

            if (snapshot.TownhallMax != MaxTownhallHp)
            {
                Fail("townhall_max must be 500");
            }

            CheckResource(snapshot.Wood, "wood", Fail);
            CheckResource(snapshot.Stone, "stone", Fail);
            CheckResource(snapshot.Food, "food", Fail);

            if (snapshot.Counts.Builder < 0 || snapshot.Counts.Guard < 0 ||
                snapshot.Counts.Fighter < 0 || snapshot.Counts.UniqueJoined < 0)
            {
                Fail("counts must not be negative");
            }

            if (snapshot.Lanes.Count != 3)
            {
                Fail("exactly three lanes are required");
            }
            else
            {
                for (var i = 0; i < 3; i++)
                {
                    var lane = snapshot.Lanes[i];
                    if (lane == null)
                    {
                        Fail("lane entry is null");
                        continue;
                    }

                    if ((int)lane.Id != i)
                    {
                        Fail("lanes must be ordered west, center, east");
                    }

                    if (lane.OrdinaryHp < 0 || lane.BossHp < 0)
                    {
                        Fail("lane hit points must not be negative");
                    }
                }
            }

            if (snapshot.Vote != null)
            {
                if (snapshot.Vote.CountA < 0 || snapshot.Vote.CountB < 0 || snapshot.Vote.CountC < 0)
                {
                    Fail("vote counts must not be negative");
                }

                if (snapshot.Vote.Open && !HasAllVoteOptions(snapshot.Vote))
                {
                    Fail("an open vote must offer A, B and C");
                }

                if (!string.IsNullOrEmpty(snapshot.Vote.Selected) &&
                    snapshot.Vote.Selected != "A" && snapshot.Vote.Selected != "B" && snapshot.Vote.Selected != "C")
                {
                    Fail("selected vote must be A, B or C");
                }
            }

            if (snapshot.Spotlights.Count > MaxSpotlights)
            {
                Fail($"at most {MaxSpotlights} spotlights may be published");
            }

            foreach (var spotlight in snapshot.Spotlights)
            {
                if (spotlight == null)
                {
                    Fail("spotlight entry is null");
                    continue;
                }

                if (string.IsNullOrWhiteSpace(spotlight.PublicRef) || spotlight.PublicRef.Length > MaxPublicRefLength)
                {
                    Fail("spotlight public_ref is empty or too long");
                }
                else if (LooksLikeActorKey(spotlight.PublicRef))
                {
                    Fail("spotlight public_ref must never be an actor key");
                }

                if (string.IsNullOrWhiteSpace(spotlight.DisplayAlias) || spotlight.DisplayAlias.Length > MaxAliasLength)
                {
                    Fail("spotlight alias is empty or too long");
                }

                if (Array.IndexOf(ValidRoles, spotlight.Role) < 0)
                {
                    Fail("spotlight role must be builder, guard or fighter");
                }
            }

            if (snapshot.Banner != null)
            {
                if (string.IsNullOrWhiteSpace(snapshot.Banner.Kind) || string.IsNullOrWhiteSpace(snapshot.Banner.TextKey))
                {
                    Fail("event banner needs a kind and a text key");
                }

                if (snapshot.Banner.ExpiryMs < 0)
                {
                    Fail("event banner expiry must not be negative");
                }
            }

            if (snapshot.BossIncoming && snapshot.Phase == Phase.Results)
            {
                Fail("a boss cannot still be incoming during RESULTS");
            }

            return ValidationResult.From(errors);
        }

        private static void CheckResource(int value, string name, Action<string> fail)
        {
            if (value < 0 || value > MaxResource)
            {
                fail($"{name} outside 0..{MaxResource.ToString(CultureInfo.InvariantCulture)}");
            }
        }

        private static bool HasAllVoteOptions(VoteState vote)
        {
            if (vote.Options == null || vote.Options.Count != 3)
            {
                return false;
            }

            return vote.Options[0] == "A" && vote.Options[1] == "B" && vote.Options[2] == "C";
        }

        /// <summary>
        /// A scoped HMAC actor key is 64 lowercase hex characters. The public wire
        /// must never carry one, so a value of that shape is rejected outright.
        /// </summary>
        public static bool LooksLikeActorKey(string value)
        {
            if (string.IsNullOrEmpty(value) || value.Length != 64)
            {
                return false;
            }

            foreach (var c in value)
            {
                var isHex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F');
                if (!isHex)
                {
                    return false;
                }
            }

            return true;
        }
    }
}
