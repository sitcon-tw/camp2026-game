package content

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	quizQuestionsFile     = "quiz_questions.csv"
	minQuizQuestionCount  = 10
	quizQuestionCSVFields = 8
)

var validQuizChoices = map[string]struct{}{
	"A": {},
	"B": {},
	"C": {},
	"D": {},
}

type QuizQuestion struct {
	ID            string
	Prompt        string
	ChoiceA       string
	ChoiceB       string
	ChoiceC       string
	ChoiceD       string
	CorrectChoice string
	Explanation   string
}

func (s *Store) ListQuizQuestions() []QuizQuestion {
	if s == nil || len(s.quizQuestions) == 0 {
		return nil
	}

	questions := make([]QuizQuestion, len(s.quizQuestions))
	copy(questions, s.quizQuestions)
	return questions
}

func (s *Store) GetQuizQuestion(id string) (QuizQuestion, bool) {
	if s == nil {
		return QuizQuestion{}, false
	}

	question, ok := s.quizQuestionsByID[id]
	return question, ok
}

func loadQuizQuestions(path string) ([]QuizQuestion, map[string]QuizQuestion, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: read: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: decode csv: %w", path, err)
	}
	return validateQuizQuestions(path, rows)
}

func validateQuizQuestions(path string, rows [][]string) ([]QuizQuestion, map[string]QuizQuestion, error) {
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("%s: quiz question header is required", path)
	}
	if err := validateQuizHeader(path, rows[0]); err != nil {
		return nil, nil, err
	}

	var errs []error
	seen := make(map[string]struct{}, len(rows)-1)
	questions := make([]QuizQuestion, 0, len(rows)-1)

	for i, row := range rows[1:] {
		location := fmt.Sprintf("%s: quiz_questions[%d]", path, i)
		if len(row) != quizQuestionCSVFields {
			errs = append(errs, fmt.Errorf("%s must have %d columns", location, quizQuestionCSVFields))
			continue
		}

		question := normalizeQuizQuestion(QuizQuestion{
			ID:            row[0],
			Prompt:        row[1],
			ChoiceA:       row[2],
			ChoiceB:       row[3],
			ChoiceC:       row[4],
			ChoiceD:       row[5],
			CorrectChoice: row[6],
			Explanation:   row[7],
		})

		if question.ID == "" {
			errs = append(errs, fmt.Errorf("%s.id is required", location))
		} else if _, ok := seen[question.ID]; ok {
			continue
		} else {
			seen[question.ID] = struct{}{}
		}
		if question.Prompt == "" {
			errs = append(errs, fmt.Errorf("%s.prompt is required", location))
		}
		if question.ChoiceA == "" {
			errs = append(errs, fmt.Errorf("%s.choice_a is required", location))
		}
		if question.ChoiceB == "" {
			errs = append(errs, fmt.Errorf("%s.choice_b is required", location))
		}
		if question.ChoiceC == "" {
			errs = append(errs, fmt.Errorf("%s.choice_c is required", location))
		}
		if question.ChoiceD == "" {
			errs = append(errs, fmt.Errorf("%s.choice_d is required", location))
		}
		if question.CorrectChoice == "" {
			errs = append(errs, fmt.Errorf("%s.correct_choice is required", location))
		} else if _, ok := validQuizChoices[question.CorrectChoice]; !ok {
			errs = append(errs, fmt.Errorf("%s.correct_choice must be one of %s", location, sortedKeys(validQuizChoices)))
		}
		if question.Explanation == "" {
			errs = append(errs, fmt.Errorf("%s.explanation is required", location))
		}

		if question.ID != "" {
			questions = append(questions, question)
		}
	}

	if len(questions) < minQuizQuestionCount {
		errs = append(errs, fmt.Errorf("%s: at least %d quiz questions are required", path, minQuizQuestionCount))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	sort.Slice(questions, func(i, j int) bool {
		return questions[i].ID < questions[j].ID
	})

	byID := make(map[string]QuizQuestion, len(questions))
	for _, question := range questions {
		byID[question.ID] = question
	}
	return questions, byID, nil
}

func validateQuizHeader(path string, header []string) error {
	want := []string{
		"id",
		"prompt",
		"choice_a",
		"choice_b",
		"choice_c",
		"choice_d",
		"correct_choice",
		"explanation",
	}
	if len(header) != len(want) {
		return fmt.Errorf("%s: quiz question header must have %d columns", path, len(want))
	}
	for i, wantColumn := range want {
		if strings.TrimSpace(header[i]) != wantColumn {
			return fmt.Errorf("%s: quiz question header column %d must be %q", path, i, wantColumn)
		}
	}
	return nil
}

func normalizeQuizQuestion(question QuizQuestion) QuizQuestion {
	question.ID = strings.TrimSpace(question.ID)
	question.Prompt = strings.TrimSpace(question.Prompt)
	question.ChoiceA = strings.TrimSpace(question.ChoiceA)
	question.ChoiceB = strings.TrimSpace(question.ChoiceB)
	question.ChoiceC = strings.TrimSpace(question.ChoiceC)
	question.ChoiceD = strings.TrimSpace(question.ChoiceD)
	question.CorrectChoice = strings.ToUpper(strings.TrimSpace(question.CorrectChoice))
	question.Explanation = strings.TrimSpace(question.Explanation)
	return question
}
