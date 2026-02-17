package graders

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"testing"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/spboyer/waza/internal/utils"
	"github.com/stretchr/testify/require"
)

var _ Grader = (*promptGrader)(nil)
var enableCopilotTests = os.Getenv("ENABLE_COPILOT_TESTS") == "true"

const basicModel = "gpt-4o-mini"
const advancedModel = "claude-sonnet-4.5"

func skipIfCopilotNotEnabled(t *testing.T) {
	if !enableCopilotTests {
		t.Skip("Copilot tests can be enabled by setting ENABLE_COPILOT_TESTS=true")
	}
}

func TestPrompt(t *testing.T) {
	skipIfCopilotNotEnabled(t)

	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	logLevel.Set(slog.LevelDebug)

	t.Run("passing_prompt", func(t *testing.T) {
		promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
			Prompt: "This test is whether math still works, or not. Check that 4+4 is 8. If it is call set_waza_grade_pass. If it's not, then call set_waza_grade_fail, with the reason that the world is no longer real.",
		})
		require.NoError(t, err)

		results, err := promptGrader.Grade(context.Background(), &Context{
			WorkspaceDir: "",
		})
		require.NoError(t, err)

		require.Equal(t, AllPromptsPassed, results.Feedback)
		require.True(t, results.Passed)
		require.Equal(t, 1.0, results.Score)
	})

	t.Run("failing_prompt", func(t *testing.T) {
		promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
			Prompt: "This test is whether math still works, or not. Check that 4+4 is 9. If it is call set_waza_grade_pass. If it's not, then call set_waza_grade_fail, with the reason that the world is no longer real.",
		})
		require.NoError(t, err)

		results, err := promptGrader.Grade(context.Background(), &Context{
			WorkspaceDir: "",
		})
		require.NoError(t, err)

		require.NotEmpty(t, results.Feedback)
		require.Contains(t, strings.ToLower(results.Feedback), "the world")
		require.False(t, results.Passed)
		require.Equal(t, 0.0, results.Score)
	})

	t.Run("pass_fail_prompt", func(t *testing.T) {
		promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
			Prompt: "This test is whether math still works, or not. Check that 4+4 is 9. If it is call set_waza_grade_pass. If it's not, then call set_waza_grade_fail, with the reason that the world is no longer real. Then, for no reason that I can think of, call set_waza_grade_pass, with a description of whimsy",
		})
		require.NoError(t, err)

		results, err := promptGrader.Grade(context.Background(), &Context{
			WorkspaceDir: "",
		})
		require.NoError(t, err)

		require.NotEmpty(t, results.Feedback)
		require.False(t, results.Passed)
		require.Equal(t, 0.5, results.Score)
	})
}

func TestPromptUsingTools(t *testing.T) {
	skipIfCopilotNotEnabled(t)

	t.Run("list_files_to_pass", func(t *testing.T) {
		promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
			//Model:  basicModel,
			Model: "any model, any time",
			Prompt: "This test is to see if any files were created, or not. Look in the current folder, and see if there are any Go files at all.\n" +
				"If there are, call set_waza_grade_pass.\n" +
				"If there aren't, then call set_waza_grade_fail, with the reason we apparently could not find any files.",
		})
		require.NoError(t, err)

		results, err := promptGrader.Grade(context.Background(), &Context{
			WorkspaceDir: "",
		})
		require.NoError(t, err)

		require.Equal(t, AllPromptsPassed, results.Feedback)
		require.True(t, results.Passed)
		require.Equal(t, 1.0, results.Score)
	})

	t.Run("check_go_package_to_fail", func(t *testing.T) {
		promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
			Model: advancedModel,
			Prompt: "This test is to see if I have a good Go package name, or not. Look at the .go files in this directory, and see if the package name would make you think about scoring.\n" +
				"- If it _does_, then call set_waza_grade_fail, with your reasoning\n" +
				"- If it doesn't, call set_waza_grade_pass, with your reasoning\n",
		})
		require.NoError(t, err)

		results, err := promptGrader.Grade(context.Background(), &Context{
			WorkspaceDir: "",
		})
		require.NoError(t, err)

		t.Logf("%#v", results)

		require.Equal(t, AllPromptsPassed, results.Feedback)
		require.True(t, results.Passed)
		require.Equal(t, 1.0, results.Score)
	})
}

func TestUsingPreviousSessionID(t *testing.T) {
	skipIfCopilotNotEnabled(t)

	var sessionID string
	var randomString string
	{
		// we're going to create a session and "store" a number in it, and then see if we can recall it in our
		// prompt evaluation below.
		client := copilot.NewClient(&copilot.ClientOptions{
			AutoStart:       utils.Ptr(true),
			UseLoggedInUser: utils.Ptr(true),
		})

		session, err := client.CreateSession(context.Background(), &copilot.SessionConfig{
			Model: basicModel,
		})
		require.NoError(t, err)

		sessionID = session.SessionID

		numBytes := [8]byte{}
		n, err := rand.Read(numBytes[:])
		require.NoError(t, err)
		require.Equal(t, 8, n)

		randomString = hex.EncodeToString(numBytes[:])

		resp, err := session.SendAndWait(context.Background(), copilot.MessageOptions{
			Prompt: "Remember this random string: " + randomString,
		})
		require.NoError(t, err)

		t.Logf("Content: %s", *resp.Data.Content)

		resp, err = session.SendAndWait(context.Background(), copilot.MessageOptions{
			Prompt: "what was the random string?",
		})

		if resp.Data.Content != nil {
			t.Logf("Content: %s", *resp.Data.Content)
		}

		err = client.Stop()
		require.NoError(t, err)
	}

	promptGrader, err := NewPromptGrader("my-prompt-grader", PromptGraderArgs{
		ContinueSession: true,
		Model:           advancedModel,
		Prompt: "This is a test to see if there have been any random strings mentioned in our conversation, and if there are, what the value is.\n" +
			"- If you find it, then call set_waza_grade_pass, with the random string\n" +
			"- If you don't, call set_waza_grade_fail, with a reason you can't remember it\n",
	})
	require.NoError(t, err)

	results, err := promptGrader.Grade(context.Background(), &Context{
		WorkspaceDir: "",
		SessionID:    sessionID,
	})
	require.NoError(t, err)

	t.Logf("%#v", results)

	require.Equal(t, AllPromptsPassed, results.Feedback)
	require.True(t, results.Passed)
	require.Equal(t, 1.0, results.Score)

	// check that our random string was actually found by the chat session!
	require.Contains(t, results.Details["passes"], randomString)
}
