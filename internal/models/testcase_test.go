package models

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadTestCase_InputResources(t *testing.T) {
	t.Run("git resource with worktree", func(t *testing.T) {
		testCase, err := LoadTestCase(filepath.Join("testdata", "git-resources-task-example.yaml"))
		require.NoError(t, err)

		require.Equal(t, *testCase.Stimulus.Git, GitResource{
			Commit: "HEAD",
			Source: ".",
			Type:   GitTypeWorktree,
		})
	})

	t.Run("files only", func(t *testing.T) {
		testCase, err := LoadTestCase(filepath.Join("testdata", "file-resources-task-example.yaml"))
		require.NoError(t, err)

		require.Nil(t, testCase.Stimulus.Git)

		require.Equal(t, []ResourceRef{
			{
				Location: "helpers.js",
				Body:     "",
			},
		}, testCase.Stimulus.Resources)
	})
}

func TestResourceRef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ref     ResourceRef
		wantErr bool
	}{
		{
			name: "valid path",
			ref:  ResourceRef{Location: "file.txt"},
		},
		{
			name: "valid content",
			ref:  ResourceRef{Body: "inline"},
		},
		{
			name:    "empty resource",
			ref:     ResourceRef{},
			wantErr: true,
		},
		{
			name: "path and content",
			ref:  ResourceRef{Location: "f.txt", Body: "inline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadTestCase_ShouldTriggerField(t *testing.T) {
	tests := []struct {
		name     string
		yamlFile string
		wantNil  bool
		wantVal  bool
	}{
		{
			name:     "should_trigger true",
			yamlFile: filepath.Join("testdata", "trigger-true-task-example.yaml"),
			wantNil:  false,
			wantVal:  true,
		},
		{
			name:     "should_trigger false",
			yamlFile: filepath.Join("testdata", "trigger-false-task-example.yaml"),
			wantNil:  false,
			wantVal:  false,
		},
		{
			name:     "should_trigger omitted",
			yamlFile: filepath.Join("testdata", "trigger-omit-task-example.yaml"),
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, err := LoadTestCase(tt.yamlFile)
			if err != nil {
				t.Fatalf("LoadTestCase: %v", err)
			}

			if tt.wantNil {
				if tc.Expectation.ExpectedTrigger != nil {
					t.Errorf("expected ExpectedTrigger nil, got %v", *tc.Expectation.ExpectedTrigger)
				}
				return
			}

			if tc.Expectation.ExpectedTrigger == nil {
				t.Fatal("expected ExpectedTrigger non-nil, got nil")
			}
			if *tc.Expectation.ExpectedTrigger != tt.wantVal {
				t.Errorf("ExpectedTrigger = %v, want %v", *tc.Expectation.ExpectedTrigger, tt.wantVal)
			}
		})
	}
}
