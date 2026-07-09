package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var validRarities = map[string]struct{}{
	"base":    {},
	"common":  {},
	"rare":    {},
	"limited": {},
}

type Store struct {
	sitones           []Sitone
	sitonesByID       map[string]Sitone
	items             []Item
	itemsByID         map[string]Item
	fusionRecipes     []FusionRecipe
	fusionRecipesByID map[string]FusionRecipe
	quizQuestions     []QuizQuestion
	quizQuestionsByID map[string]QuizQuestion
	frontMaps         []FrontMapTemplate
	frontMapsByID     map[string]FrontMapTemplate
}

func Load(dir string) (*Store, error) {
	resolvedDir, err := resolveDir(dir)
	if err != nil {
		return nil, err
	}

	sitonesPath := filepath.Join(resolvedDir, sitonesFile)
	doc, err := loadTOMLFile[sitonesDocument](sitonesPath)
	if err != nil {
		return nil, err
	}

	sitones, sitonesByID, err := validateSitones(sitonesPath, doc.Sitones)
	if err != nil {
		return nil, err
	}

	itemsPath := filepath.Join(resolvedDir, itemsFile)
	itemsDoc, err := loadTOMLFile[itemsDocument](itemsPath)
	if err != nil {
		return nil, err
	}

	items, itemsByID, err := validateItems(itemsPath, itemsDoc.Items)
	if err != nil {
		return nil, err
	}

	fusionRecipesPath := filepath.Join(resolvedDir, fusionRecipesFile)
	fusionRecipesDoc, err := loadTOMLFile[fusionRecipesDocument](fusionRecipesPath)
	if err != nil {
		return nil, err
	}

	fusionRecipes, fusionRecipesByID, err := validateFusionRecipes(fusionRecipesPath, fusionRecipesDoc.FusionRecipes, sitonesByID, itemsByID)
	if err != nil {
		return nil, err
	}

	quizQuestionsPath := filepath.Join(resolvedDir, quizQuestionsFile)
	quizQuestions, quizQuestionsByID, err := loadQuizQuestions(quizQuestionsPath)
	if err != nil {
		return nil, err
	}

	frontMapsPath := filepath.Join(resolvedDir, frontMapsFile)
	frontMapsDoc, err := loadTOMLFile[frontMapsDocument](frontMapsPath)
	var frontMaps []FrontMapTemplate
	frontMapsByID := map[string]FrontMapTemplate{}
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else {
		frontMaps, frontMapsByID, err = validateFrontMaps(frontMapsPath, frontMapsDoc.FrontMaps)
		if err != nil {
			return nil, err
		}
	}

	return &Store{
		sitones:           sitones,
		sitonesByID:       sitonesByID,
		items:             items,
		itemsByID:         itemsByID,
		fusionRecipes:     fusionRecipes,
		fusionRecipesByID: fusionRecipesByID,
		quizQuestions:     quizQuestions,
		quizQuestionsByID: quizQuestionsByID,
		frontMaps:         frontMaps,
		frontMapsByID:     frontMapsByID,
	}, nil
}

func resolveDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", errors.New("content dir is required")
	}

	for _, candidate := range contentDirCandidates(dir) {
		stat, err := os.Stat(candidate)
		if err == nil {
			if !stat.IsDir() {
				return "", fmt.Errorf("content dir %q is not a directory", candidate)
			}
			return candidate, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stat content dir %q: %w", candidate, err)
		}
	}

	return "", fmt.Errorf("content dir %q does not exist", dir)
}

func contentDirCandidates(dir string) []string {
	candidates := []string{dir}
	if filepath.IsAbs(dir) {
		return candidates
	}

	if stripped, ok := strings.CutPrefix(dir, "server"+string(filepath.Separator)); ok {
		candidates = append(candidates, stripped)
	} else {
		candidates = append(candidates, filepath.Join("server", dir))
	}
	return candidates
}

func loadTOMLFile[T any](path string) (T, error) {
	var out T

	data, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("%s: read: %w", path, err)
	}
	if err := toml.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("%s: decode toml: %w", path, err)
	}

	return out, nil
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
