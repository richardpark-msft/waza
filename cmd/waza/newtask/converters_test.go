package newtask

import (
	"path/filepath"
	"testing"

	"github.com/microsoft/waza/internal/models"
	"github.com/stretchr/testify/require"
)

func TestCreateTestCaseFromCopilotLog_UsingSkillFixture(t *testing.T) {
	testCopilotLog := filepath.Join("..", "..", "..", "internal", "testdata", "copilot_events_using_skill.json")

	tc, err := CreateTestCaseFromCopilotLog(testCopilotLog, &CreateTestCaseFromCopilotLogOptions{
		DisplayName: "fixture-case",
		TestID:      "fixture-id",
		Tags:        []string{"from-fixture"},
	})

	require.NoError(t, err)

	expected := &models.TaskSpec{
		DisplayName: "fixture-case",
		TestID:      "fixture-id",
		Tags:        []string{"from-fixture"},
		Inputs: models.TaskInputs{
			Message: "use the example horn",
		},
		Graders: []models.Grader{
			{
				Identifier: "skills-check",
				Type:       models.GraderKindSkillInvocation,
				Parameters: models.SkillInvocationGraderParameters{
					RequiredSkills: []string{"example"},
					Mode:           models.SkillMatchingModeAnyOrder,
				},
			},
			{
				Identifier: "tools-check",
				Type:       models.GraderKindToolConstraint,
				Parameters: models.ToolConstraintGraderParameters{
					ExpectTools: []models.ToolSpecParameters{{
						Tool:         "skill",
						SkillPattern: "example",
					}},
				},
			},
			{
				Identifier: "check-response",
				Type:       models.GraderKindText,
				Parameters: models.TextGraderParameters{
					ContainsCS: []string{"yesyes"}, // response from the assistant in our test file
				},
			},
		},
	}

	require.Equal(t, expected, tc)
}
