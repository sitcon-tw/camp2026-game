package fronts

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestTerritoryAttackClosesAndCapturesEnclosure(t *testing.T) {
	front := newTerritoryTestFront(5, 5)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	setTerritoryRing(matrix, 1, 1, 3, 3, "team-001", 20)
	matrix[1][2] = territoryCell{Playable: true, Owner: "team-002", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 2, 1

	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "close-pocket", PlayerID: "player-1", TeamID: "team-001", Kind: "attack",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("close pocket: %v", err)
	}
	if command.CapturedCellCount != 2 || command.EnclosedCellCount != 1 || len(command.AffectedCells) != 2 {
		t.Fatalf("unexpected capture counts: %#v", command)
	}
	if command.ScoreDelta != 20 || command.FrontOpenPowerCost != 15 || command.FrontOpenPowerDelta != -15 || command.EmergencyResupplyAmount != 0 {
		t.Fatalf("unexpected command accounting: %#v", command)
	}
	nextMatrix, err := decodeTerritoryRows(next.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for _, coordinate := range []mongomodel.FrontCoordinate{{X: 2, Y: 1}, {X: 2, Y: 2}} {
		cell := nextMatrix[coordinate.Y][coordinate.X]
		if cell.Owner != "team-001" || cell.Defense != territoryInitialDefense {
			t.Fatalf("captured cell has wrong owner/defense at %#v: %#v", coordinate, cell)
		}
	}
	if next.Teams[0].ControlledCells != 9 || next.Teams[0].Score != 0 {
		t.Fatalf("unexpected team accounting: %#v", next.Teams[0])
	}
	if len(next.Leaderboard) == 0 || next.Leaderboard[0].Score != 90 {
		t.Fatalf("controlled territory should contribute current score: %#v", next.Leaderboard)
	}
}

func TestCaptureEnclosedTerritoryLeavesOutsideConnectedPocket(t *testing.T) {
	matrix := makePlayableTerritoryMatrix(5, 5)
	setTerritoryRing(matrix, 1, 1, 3, 3, "team-001", 20)
	matrix[1][2] = territoryCell{Playable: true}

	captured := captureEnclosedTerritory(matrix, "team-001", nil)
	if len(captured) != 0 {
		t.Fatalf("outside-connected pocket must remain unchanged: %#v", captured)
	}
	if matrix[2][2].Owner != "" {
		t.Fatalf("outside-connected center was captured: %#v", matrix[2][2])
	}
}

func TestCaptureEnclosedTerritoryProtectsEntireEnemyBasePocket(t *testing.T) {
	matrix := makePlayableTerritoryMatrix(7, 7)
	setTerritoryRing(matrix, 1, 1, 5, 5, "team-001", 20)
	matrix[3][3] = territoryCell{Playable: true, Owner: "team-002", Defense: territoryMaxDefense}
	bases := map[[2]int]string{{3, 3}: "team-002"}

	captured := captureEnclosedTerritory(matrix, "team-001", bases)
	if len(captured) != 0 {
		t.Fatalf("enemy-base pocket must be protected as a component: %#v", captured)
	}
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 4; x++ {
			if matrix[y][x].Owner == "team-001" {
				t.Fatalf("base pocket cell was captured at (%d,%d)", x, y)
			}
		}
	}
	if matrix[3][3].Owner != "team-002" || matrix[3][3].Defense != territoryMaxDefense {
		t.Fatalf("enemy base changed: %#v", matrix[3][3])
	}
}

func TestCaptureEnclosedTerritoryProtectsOnlyTheEnemyBaseComponent(t *testing.T) {
	matrix := makePlayableTerritoryMatrix(11, 7)
	setTerritoryRing(matrix, 1, 1, 4, 5, "team-001", 20)
	setTerritoryRing(matrix, 6, 1, 9, 5, "team-001", 20)
	matrix[3][2] = territoryCell{Playable: true, Owner: "team-002", Defense: territoryMaxDefense}
	bases := map[[2]int]string{{2, 3}: "team-002"}

	captured := captureEnclosedTerritory(matrix, "team-001", bases)
	if len(captured) != 6 {
		t.Fatalf("expected only the six-cell base-free pocket to be captured, got %#v", captured)
	}
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 3; x++ {
			if matrix[y][x].Owner == "team-001" {
				t.Fatalf("enemy-base component was partially captured at (%d,%d)", x, y)
			}
		}
		for x := 7; x <= 8; x++ {
			cell := matrix[y][x]
			if cell.Owner != "team-001" || cell.Defense != territoryInitialDefense {
				t.Fatalf("base-free pocket was not captured at (%d,%d): %#v", x, y, cell)
			}
		}
	}
}

func TestTerritoryResourceLandmarkRequiresExplicitOneShotSupport(t *testing.T) {
	store := testcontent.Load(t)
	template, ok := store.GetTerritoryMap("campus_territory_v1")
	if !ok {
		t.Fatal("campus territory template not loaded")
	}
	front := frontFromTerritoryTemplate(template)
	landmark, ok := territoryLandmarkAt(front.Territory.Landmarks, 8, 27)
	if !ok || landmark.Kind != "resource" || landmark.InitialResource != 40 {
		t.Fatalf("resource landmark missing: %#v", landmark)
	}
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	neighbors := territoryNeighbors(matrix, landmark.X, landmark.Y)
	if len(neighbors) == 0 {
		t.Fatal("resource landmark has no playable neighbor")
	}
	matrix[neighbors[0].Y][neighbors[0].X].Owner = "team-001"
	matrix[neighbors[0].Y][neighbors[0].X].Defense = territoryInitialDefense
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := landmark.X, landmark.Y

	capturedFront, captureCommand, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "capture-resource", PlayerID: "player-1", TeamID: "team-001", Kind: "expand",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("capture resource landmark: %v", err)
	}
	capturedMatrix, err := decodeTerritoryRows(capturedFront.Territory)
	if err != nil {
		t.Fatal(err)
	}
	if capturedMatrix[y][x].Owner != "team-001" {
		t.Fatalf("resource landmark was not captured: %#v", capturedMatrix[y][x])
	}
	if captureCommand.FrontOpenPowerCost != 10 || captureCommand.FrontOpenPowerDelta != -10 || captureCommand.EmergencyResupplyAmount != 0 {
		t.Fatalf("capturing resource landmark must not grant front power: %#v", captureCommand)
	}
	if activeEventKindAtFrontCell(capturedFront, landmark.ID) != "resource" {
		t.Fatal("capturing resource landmark removed its explicit support event")
	}

	supportedFront, supportCommand, err := applyCommandToFront(capturedFront, mongomodel.FrontCommand{
		ID: "support-resource", PlayerID: "player-1", TeamID: "team-001", Kind: "support",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("support resource landmark: %v", err)
	}
	if supportCommand.FrontOpenPowerCost != 0 || supportCommand.FrontOpenPowerDelta != landmark.InitialResource || supportCommand.EmergencyResupplyAmount != 0 {
		t.Fatalf("support did not report the one-shot resource grant: %#v", supportCommand)
	}
	if supportedFront.Teams[0].FrontOpenPower != territoryInitialFrontOpenPower-10+landmark.InitialResource {
		t.Fatalf("unexpected front power after support: %#v", supportedFront.Teams[0])
	}
	if activeEventKindAtFrontCell(supportedFront, landmark.ID) != "" {
		t.Fatal("support should consume the resource event")
	}

	beforePower := supportedFront.Teams[0].FrontOpenPower
	rejected, _, err := applyCommandToFront(supportedFront, mongomodel.FrontCommand{
		ID: "support-resource-again", PlayerID: "player-1", TeamID: "team-001", Kind: "support",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("resolved resource landmark should reject a second support command")
	}
	if rejected.Teams[0].FrontOpenPower != beforePower || rejected.Revision != supportedFront.Revision {
		t.Fatalf("rejected second support mutated the front: %#v", rejected.Teams[0])
	}
}

func TestTerritoryRescueAssignsOneDeterministicReward(t *testing.T) {
	store := testcontent.Load(t)
	template, ok := store.GetTerritoryMap("campus_territory_v1")
	if !ok {
		t.Fatal("campus territory template not loaded")
	}
	front := frontFromTerritoryTemplate(template)
	landmark, ok := territoryLandmarkAt(front.Territory.Landmarks, 53, 31)
	if !ok || landmark.Kind != "rescue" {
		t.Fatalf("rescue landmark missing: %#v", landmark)
	}
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	matrix[landmark.Y][landmark.X].Owner = "team-001"
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := landmark.X, landmark.Y

	_, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "rescue-reward", ClientCommandID: "client-rescue", FrontID: front.ID,
		PlayerID: "acting-player", TeamID: "team-001", Kind: "rescue",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("resolve rescue landmark: %v", err)
	}
	handler := New(Dependencies{Content: store})
	handler.assignFrontCommandReward(&command)
	if command.RewardSitoneID == "" || command.RewardSitoneQuantity != 1 {
		t.Fatalf("rescue did not grant one base sitone: %#v", command)
	}
	rewardID := command.RewardSitoneID
	handler.assignFrontCommandReward(&command)
	if command.RewardSitoneID != rewardID || command.RewardSitoneQuantity != 1 {
		t.Fatalf("repeated rescue reward assignment was not idempotent: %#v", command)
	}
}

func TestTerritoryEnclosureCanCrossMultipleSitoneMilestones(t *testing.T) {
	front := newTerritoryTestFront(13, 13)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	setTerritoryRing(matrix, 1, 1, 11, 11, "team-001", 20)
	matrix[1][6] = territoryCell{Playable: true, Owner: "team-002", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Teams[0].MaxControlledCells = 39
	front.Teams[0].SitoneMilestonesReached = 1
	front.Teams[0].NextSitoneMilestone = 50
	x, y := 6, 1

	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "multi-milestone", ClientCommandID: "client-multi", FrontID: front.ID,
		PlayerID: "player-1", TeamID: "team-001", Kind: "attack",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("capture large enclosure: %v", err)
	}
	if command.CapturedCellCount != 82 || command.EnclosedCellCount != 81 || command.ScoreDelta != 820 {
		t.Fatalf("unexpected large enclosure result: %#v", command)
	}
	if command.RewardSitoneQuantity != 3 {
		t.Fatalf("expected three newly crossed milestones, got %#v", command)
	}
	if next.Teams[0].ControlledCells != 121 || next.Teams[0].MaxControlledCells != 121 || next.Teams[0].SitoneMilestonesReached != 4 || next.Teams[0].NextSitoneMilestone != 125 {
		t.Fatalf("unexpected milestone state: %#v", next.Teams[0])
	}

	handler := New(Dependencies{Content: testcontent.Load(t)})
	handler.assignFrontCommandReward(&command)
	if command.RewardSitoneID == "" || command.RewardSitoneQuantity != 3 {
		t.Fatalf("reward assignment overwrote milestone quantity: %#v", command)
	}
	rewardID := command.RewardSitoneID
	handler.assignFrontCommandReward(&command)
	if command.RewardSitoneID != rewardID || command.RewardSitoneQuantity != 3 {
		t.Fatalf("reward assignment must be idempotent: %#v", command)
	}
}

func TestTerritorySitoneMilestonesUseHistoricalMaximum(t *testing.T) {
	team := mongomodel.FrontTeam{MaxControlledCells: 24, NextSitoneMilestone: 25}
	if earned := applyTerritorySitoneMilestones(&team, 24, 25); earned != 1 {
		t.Fatalf("expected first threshold reward, got %d", earned)
	}
	if team.MaxControlledCells != 25 || team.SitoneMilestonesReached != 1 || team.NextSitoneMilestone != 50 {
		t.Fatalf("unexpected first milestone state: %#v", team)
	}

	team.MaxControlledCells = 50
	team.SitoneMilestonesReached = 2
	team.NextSitoneMilestone = 75
	if earned := applyTerritorySitoneMilestones(&team, 24, 25); earned != 0 {
		t.Fatalf("recapturing previously held territory re-awarded a milestone: %d", earned)
	}
	if team.MaxControlledCells != 50 || team.SitoneMilestonesReached != 2 || team.NextSitoneMilestone != 75 {
		t.Fatalf("historical milestone state regressed: %#v", team)
	}
}

func TestWithFrontDefaultsDerivesPersistedTerritoryMilestoneProgress(t *testing.T) {
	front := newTerritoryTestFront(30, 1)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for x := range matrix[0] {
		matrix[0][x] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Teams[0].ControlledCells = 0
	front.Teams[0].MaxControlledCells = 0
	front.Teams[0].SitoneMilestonesReached = 0
	front.Teams[0].NextSitoneMilestone = 0

	front = New(Dependencies{}).withFrontDefaults(front)
	if front.Teams[0].ControlledCells != 30 || front.Teams[0].MaxControlledCells != 30 ||
		front.Teams[0].SitoneMilestonesReached != 1 || front.Teams[0].NextSitoneMilestone != 50 {
		t.Fatalf("persisted territory progress was not derived on read: %#v", front.Teams[0])
	}
}

func TestTerritoryEmergencyResupplyIsExplicitAndOncePerTeam(t *testing.T) {
	front := newTerritoryTestFront(20, 1)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	matrix[0][0] = territoryCell{Playable: true, Owner: "team-001", Defense: 20}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	front.Teams[0].FrontOpenPower = 0

	option := findTerritoryCommandOption(territoryCommandOptionResponses(front, "team-001"), "expand")
	if option == nil || !option.Enabled {
		t.Fatalf("one-time resupply should keep the low-power option enabled: %#v", option)
	}
	x, y := 19, 0
	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "resupply-expand", PlayerID: "player-1", TeamID: "team-001", Kind: "expand",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("apply command with emergency resupply: %v", err)
	}
	if next.Teams[0].EmergencyResupplies != 1 || next.Teams[0].FrontOpenPower != 20 {
		t.Fatalf("unexpected team resupply state: %#v", next.Teams[0])
	}
	if command.FrontOpenPowerCost != 10 || command.EmergencyResupplyAmount != 30 || command.FrontOpenPowerDelta != 20 {
		t.Fatalf("resupply and cost must be reported separately: %#v", command)
	}

	next.Teams[0].FrontOpenPower = 0
	option = findTerritoryCommandOption(territoryCommandOptionResponses(next, "team-001"), "expand")
	if option == nil || option.Enabled {
		t.Fatalf("used resupply must not keep an unaffordable option enabled: %#v", option)
	}
	if _, _, err := applyCommandToFront(next, mongomodel.FrontCommand{
		ID: "resupply-expand-2", PlayerID: "player-1", TeamID: "team-001", Kind: "expand",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	}); err == nil {
		t.Fatal("expected a second zero-power command to be rejected after the resupply was used")
	}
}

func TestTerritoryReinforceSpreadsBudgetAcrossPartialCells(t *testing.T) {
	front := newTerritoryTestFront(21, 2)
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for x := 0; x < 21; x++ {
		matrix[0][x] = territoryCell{Playable: true, Owner: "team-001", Defense: 90}
	}
	front.Territory.Rows = encodeTerritoryRows(matrix)
	x, y := 0, 0

	next, command, err := applyCommandToFront(front, mongomodel.FrontCommand{
		ID: "reinforce-budget", PlayerID: "player-1", TeamID: "team-001", Kind: "reinforce",
		TargetX: &x, TargetY: &y, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("reinforce partial boundary: %v", err)
	}
	if len(command.AffectedCells) != 20 || command.FrontOpenPowerCost != 8 || command.FrontOpenPowerDelta != -8 {
		t.Fatalf("unexpected reinforcement accounting: %#v", command)
	}
	nextMatrix, err := decodeTerritoryRows(next.Territory)
	if err != nil {
		t.Fatal(err)
	}
	for x := 0; x < 20; x++ {
		if nextMatrix[0][x].Defense != 100 {
			t.Fatalf("expected cell %d to receive ten defense, got %#v", x, nextMatrix[0][x])
		}
	}
	if nextMatrix[0][20].Defense != 90 {
		t.Fatalf("reinforcement exceeded its 200-point budget: %#v", nextMatrix[0][20])
	}
}

func TestResetFrontCommandDerivedFieldsClearsRetryRewards(t *testing.T) {
	command := mongomodel.FrontCommand{
		ID: "same-command", Kind: "attack", Accepted: true, Applied: true, RejectReason: "stale",
		AffectedCells: []mongomodel.FrontCoordinate{{X: 1, Y: 2}}, CapturedCellCount: 3,
		EnclosedCellCount: 2, ScoreDelta: 30, FrontOpenPowerDelta: 15, FrontOpenPowerCost: 15,
		EmergencyResupplyAmount: 30, RewardSitoneID: "stone_engineering_base", RewardSitoneQuantity: 2,
	}
	resetFrontCommandDerivedFields(&command)
	if command.ID != "same-command" || command.Kind != "attack" {
		t.Fatalf("retry reset changed command identity: %#v", command)
	}
	if command.Accepted || command.Applied || command.RejectReason != "" || len(command.AffectedCells) != 0 ||
		command.CapturedCellCount != 0 || command.EnclosedCellCount != 0 || command.ScoreDelta != 0 ||
		command.FrontOpenPowerDelta != 0 || command.FrontOpenPowerCost != 0 || command.EmergencyResupplyAmount != 0 ||
		command.RewardSitoneID != "" || command.RewardSitoneQuantity != 0 {
		t.Fatalf("retry reset retained derived state: %#v", command)
	}
}

func TestFrontSitoneInventoryPrioritizesOwnedDefaultsWithoutFiveItemCap(t *testing.T) {
	defaults := []string{"default-b", "not-owned", "default-a"}
	owned := []string{"extra-f", "default-a", "extra-e", "default-b", "extra-d", "extra-c", "extra-b", "extra-a"}
	available := prioritizedOwnedSitoneIDs(defaults, owned)
	if len(available) != 8 {
		t.Fatalf("expected all eight owned sitones, got %#v", available)
	}
	if available[0] != "default-b" || available[1] != "default-a" {
		t.Fatalf("owned defaults were not prioritized: %#v", available)
	}
	selected := selectedOwnedSitoneIDs(defaults, owned)
	if len(selected) != 2 || selected[0] != "default-b" || selected[1] != "default-a" {
		t.Fatalf("unowned defaults must not be selected: %#v", selected)
	}
}

func TestTerritoryCommandRequiresSitoneAndRejectsUnknownCatalogID(t *testing.T) {
	x, y := 1, 1
	err := validateCommandRequest(CreateCommandRequest{
		ClientCommandID: "missing-sitone", Kind: "expand", TargetX: &x, TargetY: &y,
	}, contentFrontMapModeTerritoryGrid)
	if err == nil {
		t.Fatal("territory command without sitoneId should fail validation")
	}

	handler := New(Dependencies{Content: testcontent.Load(t)})
	owned, err := handler.playerOwnsFrontSitone(t.Context(), "player-1", "stone_does_not_exist")
	if err != nil {
		t.Fatalf("unknown catalog sitone lookup: %v", err)
	}
	if owned {
		t.Fatal("unknown catalog sitone must not be accepted as owned")
	}
}

func makePlayableTerritoryMatrix(width int, height int) [][]territoryCell {
	matrix := makeTerritoryMatrix(width, height)
	for y := range matrix {
		for x := range matrix[y] {
			matrix[y][x].Playable = true
		}
	}
	return matrix
}

func setTerritoryRing(matrix [][]territoryCell, minX int, minY int, maxX int, maxY int, owner string, defense int) {
	for x := minX; x <= maxX; x++ {
		matrix[minY][x] = territoryCell{Playable: true, Owner: owner, Defense: defense}
		matrix[maxY][x] = territoryCell{Playable: true, Owner: owner, Defense: defense}
	}
	for y := minY + 1; y < maxY; y++ {
		matrix[y][minX] = territoryCell{Playable: true, Owner: owner, Defense: defense}
		matrix[y][maxX] = territoryCell{Playable: true, Owner: owner, Defense: defense}
	}
}

func findTerritoryCommandOption(options []FrontCommandOptionResponse, kind string) *FrontCommandOptionResponse {
	for i := range options {
		if options[i].Kind == kind {
			return &options[i]
		}
	}
	return nil
}
