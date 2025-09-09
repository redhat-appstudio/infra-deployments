package gitwebhooks

import (
	"fmt"
	"strings"
	"testing"
)

// Test doesRepoHaveSecrets - Pure function tests
func TestDoesRepoHaveSecrets(t *testing.T) {
	tests := []struct {
		name     string
		repo     Repository
		expected bool
	}{
		{
			name: "both secrets present",
			repo: Repository{
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "token"},
						PacsSecret:    RepoSecret{Name: "test-secret", Key: "token"},
					},
				},
			},
			expected: true,
		},
		{
			name:     "no secrets present",
			repo:     Repository{},
			expected: false,
		},
		{
			name: "no webhook secret present",
			repo: Repository{
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "token"},
					},
				},
			},
			expected: false,
		},
		{
			name: "no pacs secret present",
			repo: Repository{
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "token"},
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := doesRepoHaveSecrets(tc.repo)
			if result != tc.expected {
				t.Errorf("doesRepoHaveSecrets() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	output []byte
	err    error
}

// implements the CommandExecutor interface.
func (m *MockCommandExecutor) Output() ([]byte, error) {
	return m.output, m.err
}

// MockCommandExecutorBuilder for easy test setup
type MockCommandExecutorBuilder struct {
	output []byte
	err    error
}

func NewMockCommandExecutor() *MockCommandExecutorBuilder {
	return &MockCommandExecutorBuilder{}
}

func (b *MockCommandExecutorBuilder) WithOutput(output []byte) *MockCommandExecutorBuilder {
	b.output = output
	return b
}

func (b *MockCommandExecutorBuilder) WithError(err error) *MockCommandExecutorBuilder {
	b.err = err
	return b
}

func (b *MockCommandExecutorBuilder) Build() CommandExecutor {
	return &MockCommandExecutor{
		output: b.output,
		err:    b.err,
	}
}

const (
	testSecret       = `{"data":{"pacs-token":"dGVzdC1wYWNzLXRva2Vu","webhook-token":"dGVzdC13ZWJob29rLXRva2Vu"}}` // notsecret
	testPacsToken    = "test-pacs-token"
	testWebhookToken = "test-webhook-token"
)

func TestGetSecretToken(t *testing.T) {
	tests := []struct {
		name          string
		repo          Repository
		secretType    string
		mockOutput    []byte
		mockError     error
		expectedToken string
		expectedError string
	}{
		{
			name: "successful webhook secret retrieval",
			repo: Repository{
				Metadata: struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				}{
					Name:      "test-repo",
					Namespace: "test-namespace",
				},
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "webhook-token"},
					},
				},
			},
			secretType:    webhookSecretType,
			mockOutput:    []byte(testSecret),
			expectedToken: testWebhookToken,
		},
		{
			name: "successful pacs secret retrieval",
			repo: Repository{
				Metadata: struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				}{
					Name:      "test-repo",
					Namespace: "test-namespace",
				},
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						PacsSecret: RepoSecret{Name: "test-secret", Key: "pacs-token"},
					},
				},
			},
			secretType:    pacsSecretType,
			mockOutput:    []byte(testSecret),
			expectedToken: testPacsToken,
		},
		{
			name:          "invalid secret type",
			repo:          Repository{},
			secretType:    "invalid",
			expectedError: "invalid secret type: invalid",
		},
		{
			name: "secret not found error",
			repo: Repository{
				Metadata: struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				}{
					Name:      "test-repo",
					Namespace: "test-namespace",
				},
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "webhook-token"},
					},
				},
			},
			secretType:    webhookSecretType,
			mockError:     fmt.Errorf("secret not found"),
			expectedError: "error retrieving secret 'test-secret': secret not found",
		},
		{
			name: "could not unmarshal secret",
			repo: Repository{
				Metadata: struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				}{
					Name:      "test-repo",
					Namespace: "test-namespace",
				},
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "invalid-secret", Key: "webhook-token"},
					},
				},
			},
			secretType:    webhookSecretType,
			mockOutput:    []byte("invalid-secret"),
			expectedError: "error unmarshalling secret JSON for 'invalid-secret': invalid character 'i' looking for beginning of value",
		},
		{
			name: "key not found in secret",
			repo: Repository{
				Metadata: struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				}{
					Name:      "test-repo",
					Namespace: "test-namespace",
				},
				Spec: struct {
					URL         string      `json:"url"`
					GitProvider GitProvider `json:"git_provider"`
				}{
					GitProvider: GitProvider{
						WebhookSecret: RepoSecret{Name: "test-secret", Key: "invalid-key"},
					},
				},
			},
			secretType:    webhookSecretType,
			mockOutput:    []byte(testSecret),
			expectedError: "key 'invalid-key' not found in secret 'test-secret'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock executor
			mockExecutor := NewMockCommandExecutor().
				WithOutput(tc.mockOutput).
				WithError(tc.mockError).
				Build()

			// Test the function
			result, err := getSecretToken(tc.repo, tc.secretType, mockExecutor)

			// Verify results
			if tc.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tc.expectedError)
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("expected error containing '%s', got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tc.expectedToken {
					t.Errorf("expected token '%s', got '%s'", tc.expectedToken, result)
				}
			}
		})
	}
}

func TestGetSpecialExternalRepos(t *testing.T) {
	tests := []struct {
		name          string
		mockOutput    []byte
		mockError     error
		expectedRepos []Repository
		expectedError string
	}{
		{
			name:       "successful retrieval with a GitLab repo",
			mockOutput: []byte(`{"metadata":{"name":"test-gitlab-repo","namespace":"test-ns"},"spec":{"url":"https://gitlab.com/test/repo","git_provider":{"url":"https://gitlab.com", "type":"gitlab","webhook_secret":{"name":"test-gitlab-webhook-secret","key":"token"},"secret":{"name":"test-gitlab-pacs-secret","key":"token"}}}}`),
			expectedRepos: []Repository{
				{
					Metadata: struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}{
						Name:      "test-gitlab-repo",
						Namespace: "test-ns",
					},
					Spec: struct {
						URL         string      `json:"url"`
						GitProvider GitProvider `json:"git_provider"`
					}{
						URL: "https://gitlab.com/test/repo",
						GitProvider: GitProvider{
							Type:          gitLabType,
							URL:           gitLabComURL,
							WebhookSecret: RepoSecret{Name: "test-gitlab-webhook-secret", Key: "token"},
							PacsSecret:    RepoSecret{Name: "test-gitlab-pacs-secret", Key: "token"},
						},
					},
				},
			},
		},
		{
			name:       "successful retrieval with GitHub repo",
			mockOutput: []byte(`{"metadata":{"name":"test-github-repo","namespace":"test-ns"},"spec":{"url":"https://github.com/test/repo","type":"github","git_provider":{"url":"https://github.com","webhook_secret":{"name":"test-github-webhook-secret","key":"token"},"secret":{"name":"test-github-pacs-secret","key":"token"}}}}`),
			expectedRepos: []Repository{
				{
					Metadata: struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}{
						Name:      "test-github-repo",
						Namespace: "test-ns",
					},
					Spec: struct {
						URL         string      `json:"url"`
						GitProvider GitProvider `json:"git_provider"`
					}{
						URL: "https://github.com/test/repo",
						GitProvider: GitProvider{
							Type:          gitHubType,
							URL:           gitHubComURL,
							WebhookSecret: RepoSecret{Name: "test-github-webhook-secret", Key: "token"},
							PacsSecret:    RepoSecret{Name: "test-github-pacs-secret", Key: "token"},
						},
					},
				},
			},
		},
		{
			name:          "GitHub repo without secrets (using GitHub App)",
			mockOutput:    []byte(`{"metadata":{"name":"test-github-app-repo","namespace":"test-ns"},"spec":{"url":"https://github.com/test/repo","git_provider":{}}}`),
			expectedRepos: []Repository{},
		},
		{
			name:          "GitLab repo without secrets (should be skipped)",
			mockOutput:    []byte(`{"metadata":{"name":"test-gitlab-no-secrets","namespace":"test-ns"},"spec":{"url":"https://gitlab.com/test/repo","git_provider":{"url":"https://gitlab.com", "type":"gitlab","webhook_secret":{},"secret":{}}}}`),
			expectedRepos: []Repository{},
		},
		{
			name: "multiple repos with mixed results",
			mockOutput: []byte(`{"metadata":{"name":"test-gitlab-repo","namespace":"test-ns"},"spec":{"url":"https://gitlab.com/test/repo","git_provider":{"url":"https://gitlab.com", "type":"gitlab","webhook_secret":{"name":"test-gitlab-webhook-secret","key":"token"},"secret":{"name":"test-gitlab-pacs-secret","key":"token"}}}}
{"metadata":{"name":"test-github-repo","namespace":"test-ns"},"spec":{"url":"https://github.com/test/repo","git_provider":{"url":"https://github.com", "type":"github","webhook_secret":{"name":"test-github-webhook-secret","key":"token"},"secret":{"name":"test-github-pacs-secret","key":"token"}}}}
{"metadata":{"name":"test-github-app-repo","namespace":"test-ns"},"spec":{"url":"https://github.com/test/repo","git_provider":{}}}`),
			expectedRepos: []Repository{
				{
					Metadata: struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}{
						Name:      "test-gitlab-repo",
						Namespace: "test-ns",
					},
					Spec: struct {
						URL         string      `json:"url"`
						GitProvider GitProvider `json:"git_provider"`
					}{
						URL: "https://gitlab.com/test/repo",
						GitProvider: GitProvider{
							Type:          gitLabType,
							URL:           gitLabComURL,
							WebhookSecret: RepoSecret{Name: "test-gitlab-webhook-secret", Key: "token"},
							PacsSecret:    RepoSecret{Name: "test-gitlab-pacs-secret", Key: "token"},
						},
					},
				},
				{
					Metadata: struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}{
						Name:      "test-github-repo",
						Namespace: "test-ns",
					},
					Spec: struct {
						URL         string      `json:"url"`
						GitProvider GitProvider `json:"git_provider"`
					}{
						URL: "https://github.com/test/repo",
						GitProvider: GitProvider{
							Type:          gitHubType,
							URL:           gitHubComURL,
							WebhookSecret: RepoSecret{Name: "test-github-webhook-secret", Key: "token"},
							PacsSecret:    RepoSecret{Name: "test-github-pacs-secret", Key: "token"},
						},
					},
				},
			},
		},
		{
			name:          "command execution error",
			mockError:     fmt.Errorf("oc command failed"),
			expectedError: "error executing retrieving repository resources",
		},
		{
			name:          "invalid JSON in output",
			mockOutput:    []byte(`{"invalid": json}`),
			expectedRepos: []Repository{},
		},
		{
			name:          "empty output",
			mockOutput:    []byte(``),
			expectedRepos: []Repository{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock executor
			mockExecutor := NewMockCommandExecutor().
				WithOutput(tc.mockOutput).
				WithError(tc.mockError).
				Build()

			// Test the function
			fmt.Println("\nRunning test: ", tc.name)
			result, err := getSpecialExternalRepos(mockExecutor)

			// Verify results
			if tc.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tc.expectedError)
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("expected error containing '%s', got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(result) != len(tc.expectedRepos) {
					t.Errorf("expected %d repos, got %d", len(tc.expectedRepos), len(result))
				}
				for i, expectedRepo := range tc.expectedRepos {
					if i >= len(result) {
						t.Errorf("expected repo %d but got none", i)
						continue
					}
					if result[i].Metadata.Name != expectedRepo.Metadata.Name {
						t.Errorf("expected repo name '%s', got '%s'", expectedRepo.Metadata.Name, result[i].Metadata.Name)
					}
					if result[i].Metadata.Namespace != expectedRepo.Metadata.Namespace {
						t.Errorf("expected repo namespace '%s', got '%s'", expectedRepo.Metadata.Namespace, result[i].Metadata.Namespace)
					}
				}
			}
		})
	}
}
