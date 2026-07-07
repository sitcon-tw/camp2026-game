package admin

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	dashboardHistoryBucketHour = "hour"
	dashboardHistoryBucketDay  = "day"
	dashboardHistoryTimezone   = "Asia/Taipei"
)

var dashboardHistoryLocation = time.FixedZone(dashboardHistoryTimezone, 8*60*60)

// History godoc
// @Summary Get admin resource history
// @Description Returns an admin-only cumulative history of total sitones and open power.
// @Tags admin
// @Produce json
// @Param bucket query string false "History bucket: hour or day"
// @Success 200 {object} DashboardHistoryResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/history [get]
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	bucket, err := dashboardHistoryBucket(r.URL.Query().Get("bucket"))
	if err != nil {
		httpx.WriteProblem(w, r, httpx.BadRequest("bucket must be hour or day"))
		return
	}

	if !h.requireDatabase(w, r) {
		return
	}

	raw, err := h.dashboardHistoryRawData(r.Context(), bucket)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("history unavailable", "admin_history_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, buildDashboardHistoryResponse(time.Now().UTC(), bucket, raw))
}

type dashboardHistoryRawData struct {
	CurrentSitoneTotal int
	SitoneDeltas       []dashboardHistoryDeltaStat
	OpenPowerDeltas    []dashboardHistoryDeltaStat
}

type dashboardHistoryDeltaStat struct {
	Timestamp time.Time `bson:"_id"`
	Amount    int       `bson:"amount"`
}

type dashboardHistoryTotalStat struct {
	Total int `bson:"total"`
}

func dashboardHistoryBucket(value string) (string, error) {
	switch value {
	case "", dashboardHistoryBucketHour:
		return dashboardHistoryBucketHour, nil
	case dashboardHistoryBucketDay:
		return dashboardHistoryBucketDay, nil
	default:
		return "", errors.New("unsupported history bucket")
	}
}

func (h *Handler) dashboardHistoryRawData(ctx context.Context, bucket string) (dashboardHistoryRawData, error) {
	players, err := findAllDashboard[dashboardPlayer](
		ctx,
		h.db,
		mongomodel.PlayersCollection,
		bson.M{},
		options.Find().SetProjection(dashboardPlayerProjection()),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	playerIDs := dashboardPlayerIDs(players)

	currentSitones, err := aggregateOneDashboard[dashboardHistoryTotalStat](
		ctx,
		h.db,
		mongomodel.PlayerSitonesCollection,
		dashboardHistoryCurrentSitoneTotalPipeline(playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}

	staffSitoneDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.StaffRewardsCollection,
		dashboardHistoryStaffSitoneDeltaPipeline(bucket, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	dropSitoneDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.MatchItemDropsCollection,
		dashboardHistoryDropSitoneDeltaPipeline(bucket, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	fusionInputDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.FusionRecordsCollection,
		dashboardHistoryFusionSitoneDeltaPipeline(bucket, "inputs", -1, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	fusionOutputDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.FusionRecordsCollection,
		dashboardHistoryFusionSitoneDeltaPipeline(bucket, "outputs", 1, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	inventoryTrimDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.InventoryTrimsCollection,
		dashboardHistoryInventoryTrimSitoneDeltaPipeline(bucket, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}
	openPowerDeltas, err := aggregateAllDashboard[dashboardHistoryDeltaStat](
		ctx,
		h.db,
		mongomodel.OpenPowerRecordsCollection,
		dashboardHistoryOpenPowerDeltaPipeline(bucket, playerIDs),
	)
	if err != nil {
		return dashboardHistoryRawData{}, err
	}

	sitoneDeltas := make([]dashboardHistoryDeltaStat, 0, len(staffSitoneDeltas)+len(dropSitoneDeltas)+len(fusionInputDeltas)+len(fusionOutputDeltas)+len(inventoryTrimDeltas))
	sitoneDeltas = append(sitoneDeltas, staffSitoneDeltas...)
	sitoneDeltas = append(sitoneDeltas, dropSitoneDeltas...)
	sitoneDeltas = append(sitoneDeltas, fusionInputDeltas...)
	sitoneDeltas = append(sitoneDeltas, fusionOutputDeltas...)
	sitoneDeltas = append(sitoneDeltas, inventoryTrimDeltas...)

	return dashboardHistoryRawData{
		CurrentSitoneTotal: currentSitones.Total,
		SitoneDeltas:       sitoneDeltas,
		OpenPowerDeltas:    openPowerDeltas,
	}, nil
}

func dashboardHistoryCurrentSitoneTotalPipeline(playerIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}},
			{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	}
}

func dashboardHistoryStaffSitoneDeltaPipeline(bucket string, playerIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "recipient_player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}},
			{Key: "kind", Value: "sitone"},
			{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", bucket)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func dashboardHistoryDropSitoneDeltaPipeline(bucket string, playerIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}},
			{Key: "dropped", Value: true},
			{Key: "granted", Value: true},
			{Key: "sitone_id", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: ""}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", bucket)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: dashboardHistoryPositiveQuantityOrOne("$quantity")}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func dashboardHistoryFusionSitoneDeltaPipeline(bucket string, componentField string, sign int, playerIDs []string) mongo.Pipeline {
	componentPath := "$" + componentField
	quantityPath := componentPath + ".quantity"
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}}}}},
		{{Key: "$unwind", Value: componentPath}},
		{{Key: "$match", Value: bson.D{{Key: componentField + ".kind", Value: "sitone"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", bucket)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{quantityPath, sign}}}}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func dashboardHistoryInventoryTrimSitoneDeltaPipeline(bucket string, playerIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}},
			{Key: "sitone_count", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", bucket)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{"$sitone_count", -1}}}}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func dashboardHistoryOpenPowerDeltaPipeline(bucket string, playerIDs []string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: bson.D{{Key: "$in", Value: playerIDs}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: dashboardHistoryBucketExpression("created_at", bucket)},
			{Key: "amount", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}

func dashboardHistoryBucketExpression(dateField string, bucket string) bson.D {
	return bson.D{{Key: "$dateTrunc", Value: bson.D{
		{Key: "date", Value: "$" + dateField},
		{Key: "unit", Value: bucket},
		{Key: "timezone", Value: dashboardHistoryTimezone},
	}}}
}

func dashboardHistoryPositiveQuantityOrOne(quantityField string) bson.D {
	return bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$gt", Value: bson.A{quantityField, 0}}},
		quantityField,
		1,
	}}}
}

func buildDashboardHistoryResponse(now time.Time, bucket string, raw dashboardHistoryRawData) DashboardHistoryResponse {
	byTimestamp := map[time.Time]*DashboardHistoryPointResponse{}
	sitoneEventTotal := 0
	for _, delta := range raw.SitoneDeltas {
		if delta.Timestamp.IsZero() {
			continue
		}
		point := dashboardHistoryPointBucket(byTimestamp, delta.Timestamp)
		point.SitoneDelta += delta.Amount
		sitoneEventTotal += delta.Amount
	}
	for _, delta := range raw.OpenPowerDeltas {
		if delta.Timestamp.IsZero() {
			continue
		}
		point := dashboardHistoryPointBucket(byTimestamp, delta.Timestamp)
		point.OpenPowerDelta += delta.Amount
	}

	nowBucket := dashboardHistoryBucketStart(now, bucket)
	if len(byTimestamp) == 0 {
		byTimestamp[nowBucket] = &DashboardHistoryPointResponse{Timestamp: nowBucket}
	} else if _, ok := byTimestamp[nowBucket]; !ok {
		byTimestamp[nowBucket] = &DashboardHistoryPointResponse{Timestamp: nowBucket}
	}

	timestamps := make([]time.Time, 0, len(byTimestamp))
	for timestamp := range byTimestamp {
		timestamps = append(timestamps, timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	sitoneBaseline := raw.CurrentSitoneTotal - sitoneEventTotal
	sitoneCount := sitoneBaseline
	openPower := 0
	points := make([]DashboardHistoryPointResponse, 0, len(timestamps))
	for _, timestamp := range timestamps {
		point := byTimestamp[timestamp]
		sitoneCount += point.SitoneDelta
		openPower += point.OpenPowerDelta
		points = append(points, DashboardHistoryPointResponse{
			Timestamp:      timestamp,
			SitoneCount:    sitoneCount,
			OpenPower:      openPower,
			SitoneDelta:    point.SitoneDelta,
			OpenPowerDelta: point.OpenPowerDelta,
		})
	}

	return DashboardHistoryResponse{
		GeneratedAt:    now,
		Bucket:         bucket,
		SitoneBaseline: sitoneBaseline,
		OpenPowerStart: 0,
		Points:         points,
	}
}

func dashboardHistoryPointBucket(byTimestamp map[time.Time]*DashboardHistoryPointResponse, timestamp time.Time) *DashboardHistoryPointResponse {
	if point, ok := byTimestamp[timestamp]; ok {
		return point
	}
	point := &DashboardHistoryPointResponse{Timestamp: timestamp}
	byTimestamp[timestamp] = point
	return point
}

func dashboardHistoryBucketStart(at time.Time, bucket string) time.Time {
	local := at.In(dashboardHistoryLocation)
	switch bucket {
	case dashboardHistoryBucketDay:
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, dashboardHistoryLocation).UTC()
	default:
		return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, dashboardHistoryLocation).UTC()
	}
}
