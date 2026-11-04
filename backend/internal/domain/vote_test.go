package domain

import "testing"

func TestVoteAvailabilityAndWindow(t *testing.T) {
	if VoteAvailable(1) || !VoteAvailable(2) || VoteAvailable(3) || !VoteAvailable(10) {
		t.Fatal("vote must be available only on even waves")
	}
	if BallotTickEligible(799) || !BallotTickEligible(800) || !BallotTickEligible(1199) || BallotTickEligible(1200) {
		t.Fatal("ballot window must be [800,1200)")
	}
}

func TestVoteCostsAndAffordability(t *testing.T) {
	if wood, food := VoteCost(VoteOptionA); wood != 50 || food != 0 {
		t.Fatalf("A cost = %d/%d", wood, food)
	}
	if wood, food := VoteCost(VoteOptionB); wood != 0 || food != 50 {
		t.Fatalf("B cost = %d/%d", wood, food)
	}
	if wood, food := VoteCost(VoteOptionC); wood != 0 || food != 0 {
		t.Fatalf("C cost = %d/%d", wood, food)
	}
	if !VoteAffordable(VoteOptionA, 50, 0) || VoteAffordable(VoteOptionA, 49, 100) {
		t.Fatal("A affordability wrong")
	}
	if VoteAffordable(VoteOptionB, 100, 49) || !VoteAffordable(VoteOptionB, 0, 50) {
		t.Fatal("B affordability wrong")
	}
	if !VoteAffordable(VoteOptionC, 0, 0) {
		t.Fatal("C must always be affordable")
	}
}

func TestVoteSelectionExamples(t *testing.T) {
	cases := []struct {
		name       string
		counts     VoteCounts
		wood, food int64
		want       VoteOption
	}{
		{"tie picks A", VoteCounts{A: 5, B: 5, C: 2}, 100, 100, VoteOptionA},
		{"unaffordable A falls to B", VoteCounts{A: 10, B: 7, C: 1}, 20, 80, VoteOptionB},
		{"both paid unavailable falls to C", VoteCounts{A: 10, B: 7, C: 1}, 20, 30, VoteOptionC},
		{"zero ballots with wood picks A", VoteCounts{}, 80, 0, VoteOptionA},
		{"zero ballots without wood picks C", VoteCounts{}, 20, 100, VoteOptionC},
		{"highest count wins", VoteCounts{A: 1, B: 4, C: 2}, 100, 100, VoteOptionB},
		{"tie B/C picks B", VoteCounts{A: 0, B: 3, C: 3}, 0, 100, VoteOptionB},
	}
	for _, tc := range cases {
		got, err := SelectVote(tc.counts, tc.wood, tc.food)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestVotePercentModifiers(t *testing.T) {
	// floor rounding must be exact: 8*115/100 = 9, 8*90/100 = 7.
	guardA, err := ApplyVotePercent(8, VoteGuardPercent(VoteOptionA))
	if err != nil || guardA != 9 {
		t.Fatalf("A guard = %d, %v", guardA, err)
	}
	guardC, err := ApplyVotePercent(8, VoteGuardPercent(VoteOptionC))
	if err != nil || guardC != 7 {
		t.Fatalf("C guard = %d, %v", guardC, err)
	}
	attackB, err := ApplyVotePercent(58, VoteAttackPercent(VoteOptionB))
	if err != nil || attackB != 66 {
		t.Fatalf("B attack = %d, %v", attackB, err)
	}
	if VoteAttackPercent(VoteOptionA) != 100 || VoteGuardPercent(VoteOptionB) != 100 {
		t.Fatal("unrelated modifiers must stay 100")
	}
}

func TestLossPenaltyContractForC(t *testing.T) {
	halved, err := LossPenaltyAfterC(101)
	if err != nil || halved != 50 {
		t.Fatalf("LossPenaltyAfterC(101) = %d, %v", halved, err)
	}
	if _, err := LossPenaltyAfterC(-1); err == nil {
		t.Fatal("negative penalty must fail")
	}
}
