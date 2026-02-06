package scoring

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spboyer/waza/internal/models"
)

func init() {
	RegisterValidator("code", NewCodeValidator)
	RegisterValidator("regex", NewRegexValidator)
}

// RegexValidator validates using regex patterns
type RegexValidator struct {
	identifier   string
	mustMatch    []string
	mustNotMatch []string
}

func NewRegexValidator(identifier string, params map[string]any) Validator {
	mustMatch := extractStringSlice(params, "must_match")
	mustNotMatch := extractStringSlice(params, "must_not_match")

	return &RegexValidator{
		identifier:   identifier,
		mustMatch:    mustMatch,
		mustNotMatch: mustNotMatch,
	}
}

func (v *RegexValidator) Identifier() string { return v.identifier }
func (v *RegexValidator) Category() string   { return "regex" }

func (v *RegexValidator) Validate(ctx *ValidationContext) *models.ValidationOut {
	return measureTime(func() *models.ValidationOut {
		var failures []string

		for _, pattern := range v.mustMatch {
			re, err := regexp.Compile(pattern)
			if err != nil {
				failures = append(failures, fmt.Sprintf("Invalid must_match regex pattern %q: %v", pattern, err))
				continue
			}

			if !re.MatchString(ctx.Output) {
				failures = append(failures, fmt.Sprintf("Missing expected pattern: %s", pattern))
			}
		}

		for _, pattern := range v.mustNotMatch {
			re, err := regexp.Compile(pattern)
			if err != nil {
				failures = append(failures, fmt.Sprintf("Invalid must_not_match regex pattern %q: %v", pattern, err))
				continue
			}

			if re.MatchString(ctx.Output) {
				failures = append(failures, fmt.Sprintf("Found forbidden pattern: %s", pattern))
			}
		}

		totalChecks := len(v.mustMatch) + len(v.mustNotMatch)
		passedChecks := totalChecks - len(failures)

		score := 1.0
		if totalChecks > 0 {
			score = float64(passedChecks) / float64(totalChecks)
		}

		feedback := "All patterns matched"
		if len(failures) > 0 {
			feedback = strings.Join(failures, "; ")
		}

		return &models.ValidationOut{
			Identifier: v.identifier,
			Kind:       "regex",
			Score:      score,
			Passed:     len(failures) == 0,
			Feedback:   feedback,
			Details: map[string]any{
				"must_match":     v.mustMatch,
				"must_not_match": v.mustNotMatch,
				"failures":       failures,
			},
		}
	})
}

func extractStringSlice(params map[string]any, key string) []string {
	result := []string{}
	if val, ok := params[key].([]any); ok {
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}
