package content

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const itemsFile = "items.toml"

const ShopPurchaseLimit = 5

var validItemTypes = map[string]struct{}{
	"material": {},
	"charm":    {},
	"cosmetic": {},
	"event":    {},
	"sitone":   {},
}

var validItemSources = map[string]struct{}{
	"shop":  {},
	"drop":  {},
	"both":  {},
	"event": {},
}

type Item struct {
	ID             string `toml:"id"`
	Name           string `toml:"name"`
	Type           string `toml:"type"`
	Rarity         string `toml:"rarity"`
	Description    string `toml:"description"`
	IconPath       string `toml:"icon_path"`
	Source         string `toml:"source"`
	Purchasable    bool   `toml:"purchasable"`
	Enabled        bool   `toml:"enabled"`
	Locked         bool   `toml:"locked"`
	PriceOpenPower int    `toml:"price_open_power"`
	Repeatable     bool   `toml:"repeatable"`
}

type itemsDocument struct {
	Items []Item `toml:"items"`
}

func (s *Store) ListItems() []Item {
	if s == nil || len(s.items) == 0 {
		return nil
	}

	items := make([]Item, len(s.items))
	copy(items, s.items)
	return items
}

func (s *Store) GetItem(id string) (Item, bool) {
	if s == nil {
		return Item{}, false
	}

	item, ok := s.itemsByID[id]
	return item, ok
}

func validateItems(path string, items []Item) ([]Item, map[string]Item, error) {
	var errs []error
	seen := make(map[string]struct{}, len(items))
	normalized := make([]Item, 0, len(items))

	for i, item := range items {
		item = normalizeItem(item)
		location := fmt.Sprintf("%s: items[%d]", path, i)

		if item.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", location))
		} else if _, ok := seen[item.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate item id %q", path, item.ID))
		} else {
			seen[item.ID] = struct{}{}
		}
		if item.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name is required", location))
		}
		if item.Type == "" {
			errs = append(errs, fmt.Errorf("%s.type is required", location))
		} else if _, ok := validItemTypes[item.Type]; !ok {
			errs = append(errs, fmt.Errorf("%s.type must be one of %s", location, sortedKeys(validItemTypes)))
		}
		if item.Rarity == "" {
			errs = append(errs, fmt.Errorf("%s.rarity is required", location))
		} else if _, ok := validRarities[item.Rarity]; !ok {
			errs = append(errs, fmt.Errorf("%s.rarity must be one of %s", location, sortedKeys(validRarities)))
		}
		if item.Source != "" {
			if _, ok := validItemSources[item.Source]; !ok {
				errs = append(errs, fmt.Errorf("%s.source must be one of %s", location, sortedKeys(validItemSources)))
			}
			if item.Source == "drop" && item.Purchasable {
				errs = append(errs, fmt.Errorf("%s.purchasable must be false when source is drop", location))
			}
			if (item.Source == "shop" || item.Source == "both") && !item.Purchasable {
				errs = append(errs, fmt.Errorf("%s.purchasable must be true when source is %s", location, item.Source))
			}
		}
		if item.Purchasable && item.PriceOpenPower <= 0 {
			errs = append(errs, fmt.Errorf("%s.price_open_power must be greater than 0 when purchasable is true", location))
		}

		normalized = append(normalized, item)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].ID < normalized[j].ID
	})

	byID := make(map[string]Item, len(normalized))
	for _, item := range normalized {
		byID[item.ID] = item
	}

	return normalized, byID, nil
}

func normalizeItem(item Item) Item {
	item.ID = strings.TrimSpace(item.ID)
	item.Name = strings.TrimSpace(item.Name)
	item.Type = strings.TrimSpace(item.Type)
	item.Rarity = strings.TrimSpace(item.Rarity)
	item.Description = strings.TrimSpace(item.Description)
	item.IconPath = strings.TrimSpace(item.IconPath)
	item.Source = strings.TrimSpace(item.Source)
	return item
}
