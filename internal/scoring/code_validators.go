package scoring

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spboyer/waza/internal/models"
)

// CodeValidator validates using assertion expressions
type CodeValidator struct {
	identifier string
	assertions []string
}

func NewCodeValidator(identifier string, params map[string]any) Validator {
	assertions := []string{}
	if a, ok := params["assertions"].([]any); ok {
		for _, item := range a {
			if s, ok := item.(string); ok {
				assertions = append(assertions, s)
			}
		}
	}

	return &CodeValidator{
		identifier: identifier,
		assertions: assertions,
	}
}

func (v *CodeValidator) Identifier() string { return v.identifier }
func (v *CodeValidator) Category() string   { return "code" }

func (v *CodeValidator) Validate(ctx *ValidationContext) *models.ValidationOut {
	return measureTime(func() *models.ValidationOut {
		if len(v.assertions) == 0 {
			return &models.ValidationOut{
				Identifier: v.identifier,
				Kind:       "code",
				Score:      1.0,
				Passed:     true,
				Feedback:   "No assertions configured",
			}
		}

		passed := 0
		var failures []string

		// Simple assertion evaluation
		for _, assertion := range v.assertions {
			if evaluateAssertion(assertion, ctx) {
				passed++
			} else {
				failures = append(failures, fmt.Sprintf("Failed: %s", assertion))
			}
		}

		score := float64(passed) / float64(len(v.assertions))
		allPassed := len(failures) == 0

		feedback := "All assertions passed"
		if !allPassed {
			feedback = strings.Join(failures, "; ")
		}

		return &models.ValidationOut{
			Identifier: v.identifier,
			Kind:       "code",
			Score:      score,
			Passed:     allPassed,
			Feedback:   feedback,
			Details: map[string]any{
				"total_assertions":  len(v.assertions),
				"passed_assertions": passed,
				"failures":          failures,
			},
		}
	})
}

// evaluateAssertion is a simple assertion evaluator
func evaluateAssertion(assertion string, ctx *ValidationContext) bool {
	// Simple pattern matching for common assertions
	// In a real implementation, you'd use a proper expression evaluator

	// len(output) > N
	if matches := regexp.MustCompile(`len\(output\)\s*>\s*(\d+)`).FindStringSubmatch(assertion); len(matches) > 1 {
		threshold := 0
		if _, err := fmt.Sscanf(matches[1], "%d", &threshold); err != nil {
			return false // Parsing failed
		}
		return len(ctx.Output) > threshold
	}

	// "text" in output.lower()
	if matches := regexp.MustCompile(`['"](.+?)['"]\s+in\s+output\.lower\(\)`).FindStringSubmatch(assertion); len(matches) > 1 {
		text := matches[1]
		return strings.Contains(strings.ToLower(ctx.Output), strings.ToLower(text))
	}

	// 'text' in output
	if matches := regexp.MustCompile(`['"](.+?)['"]\s+in\s+output`).FindStringSubmatch(assertion); len(matches) > 1 {
		text := matches[1]
		return strings.Contains(ctx.Output, text)
	}

	// Unknown pattern - return false to avoid false positives
	// User should be notified that their assertion syntax is not recognized
	return false
}

