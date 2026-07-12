package fronts

import "testing"

func TestTerritoryBattleCommands(t *testing.T) {
	for _, tt := range []struct {
		kind    string
		blocked bool
	}{
		{kind: "expand", blocked: true},
		{kind: "attack", blocked: true},
		{kind: "reinforce", blocked: true},
		{kind: "repair", blocked: true},
		{kind: "rescue", blocked: true},
		{kind: "support", blocked: true},
		{kind: "answer_challenge", blocked: true},
		{kind: "station", blocked: true},
		{kind: "withdraw", blocked: true},
		{kind: "unknown", blocked: false},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			if got := isTerritoryBattleCommand(tt.kind); got != tt.blocked {
				t.Fatalf("isTerritoryBattleCommand(%q) = %v, want %v", tt.kind, got, tt.blocked)
			}
		})
	}
}
