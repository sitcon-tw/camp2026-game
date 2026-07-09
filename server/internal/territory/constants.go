package territory

import "time"

// TeamYansanID is the fixed id of the staff-controlled boss team (研三舍).
// It is a system team: excluded from leaderboards, standings and player
// queries.
const TeamYansanID = "team_yansan"

// TeamStaffID is the staff assignment team. It exists for staff tooling, but
// does not participate in territory standings or boss thresholds.
const TeamStaffID = "team-010"

// YansanTeamName is the display name of the boss team.
const YansanTeamName = "研三舍"

func IsTerritoryParticipantTeam(teamID string) bool {
	return teamID != "" && teamID != TeamYansanID && teamID != TeamStaffID
}

const (
	// VoteDeadlineDuration is how long an attack mission stays in voting
	// before it expires without effect.
	VoteDeadlineDuration = 3 * time.Minute

	// MissionMaxDuration is the maximum time between deployment and
	// resolution of an attack mission.
	MissionMaxDuration = 45 * time.Minute

	// CaptiveCooldownDuration is how long a captured sitone stays in
	// captive cooldown before it becomes convertible.
	// Config-overridable later via admin tuning.
	CaptiveCooldownDuration = 150 * time.Minute

	// DefenseModifyCooldown is the lockout after each defense
	// configuration change.
	DefenseModifyCooldown = 5 * time.Minute
)

// DeployCap returns the attack/defense sitone cap for a team of the given
// size (5 members → 10 sitones, 6 members → 12 sitones).
func DeployCap(teamSize int) int {
	switch teamSize {
	case 5:
		return 10
	case 6:
		return 12
	default:
		return teamSize * 2
	}
}

// VoteThreshold returns the number of agreeing votes (initiator included)
// required to launch an attack (5 members → 3, 6 members → 4).
func VoteThreshold(teamSize int) int {
	switch teamSize {
	case 5:
		return 3
	case 6:
		return 4
	default:
		return teamSize/2 + 1
	}
}
