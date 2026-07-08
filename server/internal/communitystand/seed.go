package communitystand

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var Stands = []mongomodel.CommunityStand{
	{
		ID:          "ab93e6b7-aea7-4cf5-b2a9-c34b3efe0791",
		Name:        "SITCON 學生計算機年會",
		Description: "由學生自主籌辦的資訊社群，鼓勵學生分享技術、參與開源，並在活動中找到一起實作與交流的夥伴。",
		LogoURL:     "/game-icons/features/team.png",
		WebsiteURL:  "https://sitcon.org",
		Enabled:     true,
		Reward: mongomodel.StandReward{
			Kind:     "item",
			RefID:    "item_student_community_card",
			Quantity: 1,
		},
	},
	{
		ID:          "dc00c410-6a37-4d2a-9a1a-a3a9e626aef7",
		Name:        "開源路線攤位",
		Description: "介紹開源專案協作、議題追蹤、版本控制與社群溝通，協助學員把第一次貢獻拆成可以完成的小任務。",
		LogoURL:     "/game-icons/items/item_open_source_roadmap.png",
		WebsiteURL:  "https://sitcon.camp",
		Enabled:     true,
		Reward: mongomodel.StandReward{
			Kind:     "item",
			RefID:    "item_open_source_roadmap",
			Quantity: 1,
		},
	},
	{
		ID:          "4e7f1371-4720-435b-87f8-3db51f9202e5",
		Name:        "社群交流攤位",
		Description: "透過現場交流認識不同社群的活動方式，留下聯絡方式、交換想法，讓營隊之外也能持續參與。",
		LogoURL:     "/game-icons/items/item_star_village_badge.png",
		WebsiteURL:  "https://sitcon.camp",
		Enabled:     true,
		Reward: mongomodel.StandReward{
			Kind:     "item",
			RefID:    "item_star_village_badge",
			Quantity: 1,
		},
	},
}

func EnsureDefaults(ctx context.Context, db *mongo.Database) error {
	if db == nil {
		return nil
	}
	now := time.Now().UTC()
	for _, stand := range Stands {
		stand.CreatedAt = now
		stand.UpdatedAt = now
		_, err := db.Collection(mongomodel.CommunityStandsCollection).UpdateOne(
			ctx,
			bson.M{"_id": stand.ID},
			bson.M{
				"$setOnInsert": bson.M{
					"_id":         stand.ID,
					"name":        stand.Name,
					"description": stand.Description,
					"logo_url":    stand.LogoURL,
					"website_url": stand.WebsiteURL,
					"enabled":     stand.Enabled,
					"reward":      stand.Reward,
					"created_at":  now,
					"updated_at":  now,
				},
			},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
