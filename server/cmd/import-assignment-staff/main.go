package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/config"
	"github.com/sitcon-tw/camp2026-game/internal/mongodb"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	defaultTargetTeamID   = "team-010"
	defaultTargetTeamName = "Team 010"
	playerRoleStaff       = "staff"
	randomTokenBytes      = 32
)

var defaultInitialSitoneIDs = []string{
	"stone_explorer_base",
	"stone_inspiration_base",
	"stone_resonance_base",
	"stone_engineering_base",
	"stone_entertainment_base",
}

type assignmentFile struct {
	Teams []staffTeamAssignment `json:"teams"`
}

type staffTeamAssignment struct {
	TeamID string   `json:"teamId"`
	Name   string   `json:"name"`
	Staff  []string `json:"staff"`
}

type tokenGenerator func(prefix string) (string, error)

type importPlan struct {
	TargetTeam  mongomodel.Team
	SourceNames []string
	Creates     []staffCreate
	Updates     []staffUpdate
	Skips       []staffSkip
	Grants      []staffGrant
}

type staffCreate struct {
	Player mongomodel.Player
}

type staffUpdate struct {
	PlayerID         string
	Nickname         string
	TeamID           string
	AuthToken        string
	QRCodeToken      string
	DefaultSitoneIDs []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type staffSkip struct {
	PlayerID  string
	Nickname  string
	TeamID    string
	SkipCause string
}

type staffGrant struct {
	PlayerID  string
	Nickname  string
	SitoneIDs []string
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "import assignment staff failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("import-assignment-staff", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	jsonPath := flags.String("json", "", "path to staff team assignment JSON")
	targetTeamID := flags.String("target-team", defaultTargetTeamID, "team ID for non-counselor staff")
	dryRun := flags.Bool("dry-run", false, "validate and print planned changes without writing to MongoDB")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*jsonPath) == "" {
		return errors.New("-json is required")
	}

	file, err := os.Open(*jsonPath)
	if err != nil {
		return fmt.Errorf("open json: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	assignments, err := readAssignments(file)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client, err := mongodb.NewClient(ctx, cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	db := client.Database(cfg.MongoDatabase)
	targetNames, err := targetStaffNames(assignments, *targetTeamID)
	if err != nil {
		return err
	}

	playersCollection := db.Collection(mongomodel.PlayersCollection)
	existingStaff, err := findStaffPlayers(ctx, playersCollection, targetNames)
	if err != nil {
		return err
	}
	issued := issuedStaffTokens(existingStaff)

	now := time.Now().UTC()
	plan, err := buildImportPlan(assignments, *targetTeamID, existingStaff, issued, randomPrefixedToken, now)
	if err != nil {
		return err
	}
	if err := ensureGeneratedTokensAvailable(ctx, playersCollection, plan); err != nil {
		return err
	}

	if *dryRun {
		printSummary(plan, true)
		return nil
	}

	if err := upsertTeam(ctx, db.Collection(mongomodel.TeamsCollection), plan.TargetTeam); err != nil {
		return err
	}
	if err := applyPlan(ctx, playersCollection, db.Collection(mongomodel.PlayerSitonesCollection), plan); err != nil {
		return err
	}

	printSummary(plan, false)
	return nil
}

func readAssignments(reader io.Reader) (assignmentFile, error) {
	var assignments assignmentFile
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&assignments); err != nil {
		return assignmentFile{}, fmt.Errorf("decode assignment json: %w", err)
	}
	return normalizeAssignments(assignments), nil
}

func normalizeAssignments(assignments assignmentFile) assignmentFile {
	out := assignmentFile{Teams: make([]staffTeamAssignment, 0, len(assignments.Teams))}
	for _, team := range assignments.Teams {
		normalized := staffTeamAssignment{
			TeamID: strings.TrimSpace(team.TeamID),
			Name:   strings.TrimSpace(team.Name),
			Staff:  make([]string, 0, len(team.Staff)),
		}
		for _, staff := range team.Staff {
			normalized.Staff = append(normalized.Staff, strings.TrimSpace(staff))
		}
		out.Teams = append(out.Teams, normalized)
	}
	return out
}

func targetStaffNames(assignments assignmentFile, targetTeamID string) ([]string, error) {
	targetTeamID = strings.TrimSpace(targetTeamID)
	if targetTeamID == "" {
		return nil, errors.New("-target-team is required")
	}

	seen := make(map[string]string)
	var target []string
	for _, team := range assignments.Teams {
		if team.TeamID == "" {
			return nil, errors.New("assignment contains an empty team id")
		}
		if team.Name == "" {
			return nil, fmt.Errorf("team %s has an empty name", team.TeamID)
		}
		if len(team.Staff) == 0 {
			return nil, fmt.Errorf("team %s must contain at least one staff", team.TeamID)
		}
		for _, name := range team.Staff {
			if name == "" {
				return nil, fmt.Errorf("team %s contains an empty staff nickname", team.TeamID)
			}
			if previousTeamID, ok := seen[name]; ok {
				return nil, fmt.Errorf("staff %q appears in both %s and %s", name, previousTeamID, team.TeamID)
			}
			seen[name] = team.TeamID
			if team.TeamID == targetTeamID {
				target = append(target, name)
			}
		}
	}
	if len(target) == 0 {
		return nil, fmt.Errorf("assignment has no staff for target team %s", targetTeamID)
	}
	sort.Strings(target)
	return target, nil
}

func buildImportPlan(assignments assignmentFile, targetTeamID string, existingStaff []mongomodel.Player, issued map[string]struct{}, newToken tokenGenerator, now time.Time) (importPlan, error) {
	targetTeamID = strings.TrimSpace(targetTeamID)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	names, err := targetStaffNames(assignments, targetTeamID)
	if err != nil {
		return importPlan{}, err
	}

	team := mongomodel.Team{ID: targetTeamID, Name: targetTeamName(assignments, targetTeamID)}
	staffByNickname := make(map[string]mongomodel.Player, len(existingStaff))
	for _, player := range existingStaff {
		if player.Role != playerRoleStaff {
			continue
		}
		if player.Nickname == "" {
			return importPlan{}, fmt.Errorf("existing staff %q has empty nickname", player.ID)
		}
		if _, ok := staffByNickname[player.Nickname]; ok {
			return importPlan{}, fmt.Errorf("multiple existing staff players with nickname %q", player.Nickname)
		}
		staffByNickname[player.Nickname] = player
	}

	plan := importPlan{
		TargetTeam:  team,
		SourceNames: names,
		Creates:     make([]staffCreate, 0),
		Updates:     make([]staffUpdate, 0),
		Skips:       make([]staffSkip, 0),
		Grants:      make([]staffGrant, 0, len(names)),
	}
	for _, nickname := range names {
		player, ok := staffByNickname[nickname]
		if !ok {
			authToken, err := uniqueToken("staff_", issued, newToken)
			if err != nil {
				return importPlan{}, fmt.Errorf("generate auth token for %q: %w", nickname, err)
			}
			qrCodeToken, err := uniqueToken("qr_", issued, newToken)
			if err != nil {
				return importPlan{}, fmt.Errorf("generate qr code token for %q: %w", nickname, err)
			}
			plan.Creates = append(plan.Creates, staffCreate{Player: mongomodel.Player{
				ID:               authToken,
				AuthToken:        authToken,
				QRCodeToken:      qrCodeToken,
				Nickname:         nickname,
				TeamID:           targetTeamID,
				Role:             playerRoleStaff,
				DefaultSitoneIDs: append([]string(nil), defaultInitialSitoneIDs...),
				CreatedAt:        now,
				UpdatedAt:        now,
			}})
			plan.Grants = append(plan.Grants, staffGrant{
				PlayerID:  authToken,
				Nickname:  nickname,
				SitoneIDs: append([]string(nil), defaultInitialSitoneIDs...),
			})
			continue
		}

		if player.TeamID != "" && player.TeamID != targetTeamID {
			plan.Skips = append(plan.Skips, staffSkip{
				PlayerID:  player.ID,
				Nickname:  nickname,
				TeamID:    player.TeamID,
				SkipCause: "existing staff already has a team",
			})
			continue
		}

		missingAuthToken := strings.TrimSpace(player.AuthToken) == ""
		missingQRCodeToken := strings.TrimSpace(player.QRCodeToken) == ""
		missingDefaultSitones := len(player.DefaultSitoneIDs) == 0
		missingCreatedAt := player.CreatedAt.IsZero()
		missingUpdatedAt := player.UpdatedAt.IsZero()
		plan.Grants = append(plan.Grants, staffGrant{
			PlayerID:  player.ID,
			Nickname:  nickname,
			SitoneIDs: append([]string(nil), defaultInitialSitoneIDs...),
		})
		if player.TeamID == targetTeamID && !missingAuthToken && !missingQRCodeToken && !missingDefaultSitones && !missingCreatedAt && !missingUpdatedAt {
			plan.Skips = append(plan.Skips, staffSkip{
				PlayerID:  player.ID,
				Nickname:  nickname,
				TeamID:    player.TeamID,
				SkipCause: "already imported",
			})
			continue
		}

		update := staffUpdate{
			PlayerID: player.ID,
			Nickname: nickname,
			TeamID:   targetTeamID,
		}
		if missingAuthToken {
			authToken, err := uniqueToken("staff_", issued, newToken)
			if err != nil {
				return importPlan{}, fmt.Errorf("generate auth token for %q: %w", nickname, err)
			}
			update.AuthToken = authToken
		}
		if missingQRCodeToken {
			qrCodeToken, err := uniqueToken("qr_", issued, newToken)
			if err != nil {
				return importPlan{}, fmt.Errorf("generate qr code token for %q: %w", nickname, err)
			}
			update.QRCodeToken = qrCodeToken
		}
		if missingDefaultSitones {
			update.DefaultSitoneIDs = append([]string(nil), defaultInitialSitoneIDs...)
		}
		if missingCreatedAt {
			update.CreatedAt = now
		}
		if missingUpdatedAt {
			update.UpdatedAt = now
		}
		plan.Updates = append(plan.Updates, update)
	}

	sort.Slice(plan.Creates, func(i, j int) bool {
		return plan.Creates[i].Player.Nickname < plan.Creates[j].Player.Nickname
	})
	sort.Slice(plan.Updates, func(i, j int) bool {
		return plan.Updates[i].Nickname < plan.Updates[j].Nickname
	})
	sort.Slice(plan.Skips, func(i, j int) bool {
		return plan.Skips[i].Nickname < plan.Skips[j].Nickname
	})
	return plan, nil
}

func targetTeamName(assignments assignmentFile, targetTeamID string) string {
	for _, team := range assignments.Teams {
		if team.TeamID == targetTeamID && strings.TrimSpace(team.Name) != "" {
			return team.Name
		}
	}
	if targetTeamID == defaultTargetTeamID {
		return defaultTargetTeamName
	}
	return targetTeamID
}

func uniqueToken(prefix string, issued map[string]struct{}, newToken tokenGenerator) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		token, err := newToken(prefix)
		if err != nil {
			return "", err
		}
		if _, ok := issued[token]; ok {
			continue
		}
		issued[token] = struct{}{}
		return token, nil
	}
	return "", errors.New("generate unique token")
}

func randomPrefixedToken(prefix string) (string, error) {
	randomBytes := make([]byte, randomTokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func findStaffPlayers(ctx context.Context, collection *mongo.Collection, nicknames []string) ([]mongomodel.Player, error) {
	cursor, err := collection.Find(
		ctx,
		bson.M{
			"nickname": bson.M{"$in": nicknames},
			"role":     playerRoleStaff,
		},
		options.Find().
			SetSort(bson.D{
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			}),
	)
	if err != nil {
		return nil, fmt.Errorf("find staff players: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, fmt.Errorf("decode staff players: %w", err)
	}
	if players == nil {
		return []mongomodel.Player{}, nil
	}
	return players, nil
}

func issuedStaffTokens(players []mongomodel.Player) map[string]struct{} {
	issued := make(map[string]struct{})
	for _, player := range players {
		for _, token := range []string{player.ID, player.AuthToken, player.QRCodeToken} {
			if strings.TrimSpace(token) != "" {
				issued[token] = struct{}{}
			}
		}
	}
	return issued
}

func ensureGeneratedTokensAvailable(ctx context.Context, collection *mongo.Collection, plan importPlan) error {
	tokens := generatedTokens(plan)
	if len(tokens) == 0 {
		return nil
	}
	count, err := collection.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"_id": bson.M{"$in": tokens}},
			{"auth_token": bson.M{"$in": tokens}},
			{"qrcode_token": bson.M{"$in": tokens}},
		},
	})
	if err != nil {
		return fmt.Errorf("check generated token collisions: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("generated token collision detected")
	}
	return nil
}

func generatedTokens(plan importPlan) []string {
	seen := make(map[string]struct{})
	var tokens []string
	add := func(token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	for _, create := range plan.Creates {
		add(create.Player.ID)
		add(create.Player.AuthToken)
		add(create.Player.QRCodeToken)
	}
	for _, update := range plan.Updates {
		add(update.AuthToken)
		add(update.QRCodeToken)
	}
	return tokens
}

func upsertTeam(ctx context.Context, collection *mongo.Collection, team mongomodel.Team) error {
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": team.ID},
		bson.M{"$set": bson.M{"name": team.Name}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("upsert %s/%s: %w", collection.Name(), team.ID, err)
	}
	return nil
}

func applyPlan(ctx context.Context, playersCollection *mongo.Collection, playerSitonesCollection *mongo.Collection, plan importPlan) error {
	for _, create := range plan.Creates {
		if _, err := playersCollection.InsertOne(ctx, create.Player); err != nil {
			return fmt.Errorf("insert %s staff %q: %w", playersCollection.Name(), create.Player.Nickname, err)
		}
	}
	for _, update := range plan.Updates {
		set := bson.M{
			"nickname": update.Nickname,
			"role":     playerRoleStaff,
			"team_id":  update.TeamID,
		}
		if update.AuthToken != "" {
			set["auth_token"] = update.AuthToken
		}
		if update.QRCodeToken != "" {
			set["qrcode_token"] = update.QRCodeToken
		}
		if len(update.DefaultSitoneIDs) > 0 {
			set["default_sitone_ids"] = update.DefaultSitoneIDs
		}
		if !update.CreatedAt.IsZero() {
			set["created_at"] = update.CreatedAt
		}
		if !update.UpdatedAt.IsZero() {
			set["updated_at"] = update.UpdatedAt
		}
		result, err := playersCollection.UpdateOne(
			ctx,
			bson.M{
				"_id":  update.PlayerID,
				"role": playerRoleStaff,
			},
			bson.M{"$set": set},
		)
		if err != nil {
			return fmt.Errorf("update %s staff %q: %w", playersCollection.Name(), update.Nickname, err)
		}
		if result.MatchedCount != 1 {
			return fmt.Errorf("update %s staff %q: matched %d staff players", playersCollection.Name(), update.Nickname, result.MatchedCount)
		}
	}
	if err := applyInitialSitoneGrants(ctx, playerSitonesCollection, plan.Grants); err != nil {
		return err
	}
	return nil
}

func applyInitialSitoneGrants(ctx context.Context, collection *mongo.Collection, grants []staffGrant) error {
	for _, grant := range grants {
		for _, sitoneID := range grant.SitoneIDs {
			_, err := collection.UpdateOne(
				ctx,
				bson.M{
					"player_id": grant.PlayerID,
					"sitone_id": sitoneID,
				},
				bson.M{
					"$setOnInsert": bson.M{
						"_id":       newID("player_sitone"),
						"player_id": grant.PlayerID,
						"sitone_id": sitoneID,
					},
					"$max": bson.M{"quantity": 1},
				},
				options.UpdateOne().SetUpsert(true),
			)
			if err != nil {
				return fmt.Errorf("grant initial sitone %s to %s/%s: %w", sitoneID, grant.Nickname, grant.PlayerID, err)
			}
		}
	}
	return nil
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}

func printSummary(plan importPlan, dryRun bool) {
	prefix := "import complete"
	if dryRun {
		prefix = "dry run complete"
	}
	fmt.Printf("%s: target_team=%s source_staff=%d create=%d update=%d skip=%d grant_players=%d grant_sitones=%d\n",
		prefix,
		plan.TargetTeam.ID,
		len(plan.SourceNames),
		len(plan.Creates),
		len(plan.Updates),
		len(plan.Skips),
		len(plan.Grants),
		countGrantSitones(plan.Grants),
	)
	for _, create := range plan.Creates {
		fmt.Printf("create: %s -> %s\n", create.Player.Nickname, create.Player.TeamID)
	}
	for _, update := range plan.Updates {
		details := []string{"team_id"}
		if update.AuthToken != "" {
			details = append(details, "auth_token")
		}
		if update.QRCodeToken != "" {
			details = append(details, "qrcode_token")
		}
		if len(update.DefaultSitoneIDs) > 0 {
			details = append(details, "default_sitone_ids")
		}
		if !update.CreatedAt.IsZero() {
			details = append(details, "created_at")
		}
		if !update.UpdatedAt.IsZero() {
			details = append(details, "updated_at")
		}
		fmt.Printf("update: %s -> %s (%s)\n", update.Nickname, update.TeamID, strings.Join(details, ", "))
	}
	for _, skip := range plan.Skips {
		fmt.Printf("skip: %s existing_team=%s (%s)\n", skip.Nickname, skip.TeamID, skip.SkipCause)
	}
}

func countGrantSitones(grants []staffGrant) int {
	total := 0
	for _, grant := range grants {
		total += len(grant.SitoneIDs)
	}
	return total
}
