package main

import (
	"strings"
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestBuildImportPlanTargetsOnlyNonCounselorTeam(t *testing.T) {
	now := time.Date(2026, 7, 2, 3, 10, 0, 0, time.UTC)
	assignments := assignmentFile{Teams: []staffTeamAssignment{
		{TeamID: "team-001", Name: "Team 001", Staff: []string{"Yuto", "牛排"}},
		{TeamID: "team-010", Name: "Team 010", Staff: []string{"New Staff", "Needs QR", "Needs Both", "Already Team", "Already Imported"}},
	}}
	existingStaff := []mongomodel.Player{
		{
			ID:        "staff-needs-qr",
			AuthToken: "existing-auth-token",
			Nickname:  "Needs QR",
			Role:      playerRoleStaff,
		},
		{
			ID:       "staff-needs-both",
			Nickname: "Needs Both",
			Role:     playerRoleStaff,
		},
		{
			ID:        "staff-already-team",
			AuthToken: "already-team-auth-token",
			Nickname:  "Already Team",
			TeamID:    "team-003",
			Role:      playerRoleStaff,
		},
		{
			ID:               "staff-already-imported",
			AuthToken:        "already-imported-auth-token",
			QRCodeToken:      "already-imported-qr-token",
			Nickname:         "Already Imported",
			TeamID:           defaultTargetTeamID,
			Role:             playerRoleStaff,
			DefaultSitoneIDs: append([]string(nil), defaultInitialSitoneIDs...),
			CreatedAt:        now.Add(-1 * time.Hour),
			UpdatedAt:        now.Add(-1 * time.Hour),
		},
	}
	issued := map[string]struct{}{
		"staff-needs-qr":              {},
		"existing-auth-token":         {},
		"staff-needs-both":            {},
		"staff-already-team":          {},
		"already-team-auth-token":     {},
		"staff-already-imported":      {},
		"already-imported-auth-token": {},
		"already-imported-qr-token":   {},
	}
	nextToken := sequenceTokenGenerator()

	plan, err := buildImportPlan(assignments, defaultTargetTeamID, existingStaff, issued, nextToken, now)
	if err != nil {
		t.Fatalf("build import plan: %v", err)
	}

	if plan.TargetTeam.ID != defaultTargetTeamID || plan.TargetTeam.Name != defaultTargetTeamName {
		t.Fatalf("unexpected target team: %#v", plan.TargetTeam)
	}
	assertNicknames(t, plan.SourceNames, []string{"Already Imported", "Already Team", "Needs Both", "Needs QR", "New Staff"})
	if len(plan.Creates) != 1 || plan.Creates[0].Player.Nickname != "New Staff" {
		t.Fatalf("expected one create for New Staff, got %#v", plan.Creates)
	}
	if plan.Creates[0].Player.TeamID != defaultTargetTeamID || plan.Creates[0].Player.Role != playerRoleStaff {
		t.Fatalf("created staff missing team/role: %#v", plan.Creates[0].Player)
	}
	if !strings.HasPrefix(plan.Creates[0].Player.AuthToken, "staff_") || !strings.HasPrefix(plan.Creates[0].Player.QRCodeToken, "qr_") {
		t.Fatalf("created staff tokens have unexpected prefixes: %#v", plan.Creates[0].Player)
	}
	if len(plan.Creates[0].Player.DefaultSitoneIDs) != len(defaultInitialSitoneIDs) || plan.Creates[0].Player.CreatedAt != now || plan.Creates[0].Player.UpdatedAt != now {
		t.Fatalf("created staff missing default sitones or timestamps: %#v", plan.Creates[0].Player)
	}

	if len(plan.Updates) != 2 {
		t.Fatalf("expected two updates, got %#v", plan.Updates)
	}
	updates := updatesByNickname(plan.Updates)
	if updates["Needs QR"].AuthToken != "" || !strings.HasPrefix(updates["Needs QR"].QRCodeToken, "qr_") || len(updates["Needs QR"].DefaultSitoneIDs) != len(defaultInitialSitoneIDs) || updates["Needs QR"].CreatedAt != now || updates["Needs QR"].UpdatedAt != now {
		t.Fatalf("expected Needs QR to preserve auth token and generate qrcode token, got %#v", updates["Needs QR"])
	}
	if !strings.HasPrefix(updates["Needs Both"].AuthToken, "staff_") || !strings.HasPrefix(updates["Needs Both"].QRCodeToken, "qr_") || len(updates["Needs Both"].DefaultSitoneIDs) != len(defaultInitialSitoneIDs) || updates["Needs Both"].CreatedAt != now || updates["Needs Both"].UpdatedAt != now {
		t.Fatalf("expected Needs Both to generate both tokens, got %#v", updates["Needs Both"])
	}
	if len(plan.Grants) != 4 || countGrantSitones(plan.Grants) != 4*len(defaultInitialSitoneIDs) {
		t.Fatalf("expected grants for target-team staff except already-team skip, got %#v", plan.Grants)
	}

	if len(plan.Skips) != 2 {
		t.Fatalf("expected two skips, got %#v", plan.Skips)
	}
	skips := skipsByNickname(plan.Skips)
	if skips["Already Team"].TeamID != "team-003" {
		t.Fatalf("expected Already Team to be skipped with team-003, got %#v", skips["Already Team"])
	}
	if skips["Already Imported"].TeamID != defaultTargetTeamID || skips["Already Imported"].SkipCause != "already imported" {
		t.Fatalf("expected Already Imported to be skipped as already imported, got %#v", skips["Already Imported"])
	}
}

func TestTargetStaffNamesRejectsDuplicateStaff(t *testing.T) {
	assignments := assignmentFile{Teams: []staffTeamAssignment{
		{TeamID: "team-001", Name: "Team 001", Staff: []string{"Repeated"}},
		{TeamID: "team-010", Name: "Team 010", Staff: []string{"Repeated"}},
	}}

	_, err := targetStaffNames(assignments, defaultTargetTeamID)
	if err == nil || !strings.Contains(err.Error(), "appears in both team-001 and team-010") {
		t.Fatalf("expected duplicate staff error, got %v", err)
	}
}

func sequenceTokenGenerator() tokenGenerator {
	next := 0
	return func(prefix string) (string, error) {
		next++
		return prefix + "token-" + string(rune('a'+next-1)), nil
	}
}

func assertNicknames(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("nickname length mismatch: got %#v want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("nickname mismatch at %d: got %#v want %#v", index, got, want)
		}
	}
}

func updatesByNickname(updates []staffUpdate) map[string]staffUpdate {
	out := make(map[string]staffUpdate, len(updates))
	for _, update := range updates {
		out[update.Nickname] = update
	}
	return out
}

func skipsByNickname(skips []staffSkip) map[string]staffSkip {
	out := make(map[string]staffSkip, len(skips))
	for _, skip := range skips {
		out[skip.Nickname] = skip
	}
	return out
}
