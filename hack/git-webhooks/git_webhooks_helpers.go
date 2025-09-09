package gitwebhooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	k8s "k8s.io/api/core/v1"
)

const (
	webhookSecretType = "webhook"
	pacsSecretType    = "pacs"
	gitLabComURL      = "https://gitlab.com"
	gitHubComURL      = "https://github.com"
	gitLabType        = "gitlab"
	gitHubType        = "github"
)

// RepoSecret represents a single Kubernetes Secret resource.
//
//	Name: Name of the secret.
//	Key: Key of the secret.
type RepoSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// GitProvider represents a single Kubernetes GitProvider resource.
//
// Type: Type of git provider. Determines which Git provider API and authentication
//	     flow to use.

//	Supported values:
//	- 'github': GitHub.com or GitHub Enterprise
//	- 'gitlab': GitLab.com or self-hosted GitLab
//	- 'bitbucket-datacenter': Bitbucket Data Center (self-hosted)
//	- 'bitbucket-cloud': Bitbucket Cloud (bitbucket.org)
//	- 'gitea': Gitea instances
//
// URL: URL of the git provider API endpoint. This is the base URL for API requests
//	    to the Git provider (e.g., 'https://api.github.com' for GitHub or a custom
//	    GitLab instance URL).

// WebhookSecret: The secret for the webhook to use.
// PacsSecret: The secret for Pipelines as Code (PaC) to use.
type GitProvider struct {
	Type          string     `json:"type"`
	URL           string     `json:"url"`
	WebhookSecret RepoSecret `json:"webhook_secret"`
	PacsSecret    RepoSecret `json:"secret"`
}

// Repository represents a single Kubernetes Repository resource.

// Metadata:
//
//	Name: Name of the repository.
//	Namespace: Namespace of the repository.
//
// Spec:
//
//	URL: URL of the repository.
//	GitProvider: GitProvider resource for the repository.
type Repository struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		URL         string      `json:"url"`
		GitProvider GitProvider `json:"git_provider"`
	} `json:"spec"`
}

// doesRepoHaveSecrets checks if a Repository has both webhook and pacs secrets.
func doesRepoHaveSecrets(repo Repository) bool {
	return repo.Spec.GitProvider.WebhookSecret != RepoSecret{} &&
		repo.Spec.GitProvider.PacsSecret != RepoSecret{}
}

// CommandExecutor interface for executing commands (real or mocked).
type CommandExecutor interface {
	Output() ([]byte, error)
}

// RealCommandExecutor wraps exec.Cmd to implement CommandExecutor.
type RealCommandExecutor struct {
	cmd *exec.Cmd
}

// Implements the CommandExecutor interface.
func (r *RealCommandExecutor) Output() ([]byte, error) {
	return r.cmd.Output()
}

// NewRealCommandExecutor creates a new real command executor.
func NewRealCommandExecutor(cmd *exec.Cmd) CommandExecutor {
	return &RealCommandExecutor{cmd: cmd}
}

// getSecretToken retrieves the one of the Repository's secret tokens from the cluster
// using a command executor.
func getSecretToken(repo Repository, secretType string, executor CommandExecutor) (string, error) {
	// Determine which secret to retrieve.
	var repoSecret RepoSecret
	if secretType == webhookSecretType {
		repoSecret = repo.Spec.GitProvider.WebhookSecret
	} else if secretType == pacsSecretType {
		repoSecret = repo.Spec.GitProvider.PacsSecret
	} else {
		return "", fmt.Errorf("invalid secret type: %s", secretType)
	}

	// If no executor is provided, create a new real command executor to get the referenced
	// secret from the cluster.
	if executor == nil {
		executor = NewRealCommandExecutor(exec.Command("oc", "get", "secret", repoSecret.Name, "-n",
			repo.Metadata.Namespace, "-o", "json"))
	}
	secretOutput, err := executor.Output()
	if err != nil {
		return "", fmt.Errorf("error retrieving secret '%s': %v\n", repoSecret.Name, err)
	}

	// Unmarshal the secret JSON (also decodes the secret token from base64).
	var secret k8s.Secret
	err = json.Unmarshal(secretOutput, &secret)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling secret JSON for '%s': %v\n", repoSecret.Name, err)
	}

	// Retrieve and return the secret token.
	secretKeyToken, ok := secret.Data[repoSecret.Key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in secret '%s'", repoSecret.Key, repoSecret.Name)
	}
	fmt.Printf("DEBUG: Retrieved %s secret for repo '%s' successfully\n", secretType, repo.Metadata.Name)
	return string(secretKeyToken), nil
}

// getSpecialExternalRepos retrieves all Repositories that either have:
// 1. a git provider URL of "gitlab.com" (external to Red Hat's gitlab.cee.redhat.com) OR
// 2. a git provider URL of "github.com" and have secrets (thus, not using the Konflux GitHub App)
// using an executor (in the provided namespace, if any).
func getSpecialExternalRepos(executor CommandExecutor, namespace string) ([]Repository, error) {
	var repos []Repository

	fmt.Println("Searching for Repository resources with external Git provider URLs '" + gitLabComURL +
		"' and '" + gitHubComURL + "'...")
	if namespace != "" {
		fmt.Println("Searching in namespace: " + namespace + "...")
	}

	// If no executor is provided, create a new real command executor to get the Repository resources
	// resources with a git provider URL of either 'https://gitlab.com'
	// or 'https://github.com'.
	if executor == nil {
		if namespace != "" {
			executor = NewRealCommandExecutor(exec.Command("bash", "-c", `oc get repository -n `+namespace+` -o json | jq -c '.items[] |
	select(.spec.git_provider.url == "`+gitLabComURL+`" or .spec.git_provider.url == "`+gitHubComURL+`")'`))
		} else {
			executor = NewRealCommandExecutor(exec.Command("bash", "-c", `oc get repository -A -o json | jq -c '.items[] |
	select(.spec.git_provider.url == "`+gitLabComURL+`" or .spec.git_provider.url == "`+gitHubComURL+`")'`))
		}
	}
	output, err := executor.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing retrieving repository resources: %v\n", err)
	}

	// Process each line as a separate JSON object.
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var repo Repository
		err := json.Unmarshal([]byte(line), &repo)
		if err != nil {
			fmt.Printf("Warning: Error unmarshalling repository JSON: %v\nSkipping repository %s\n",
				err, repo.Metadata.Name)
			continue
		}

		// Skip if the repo does not have a git provider.
		if repo.Spec.GitProvider == (GitProvider{}) {
			continue
		}

		// Print a warning message if the repository does not have secrets but has a type of GitLab.
		// Repositories with a type of GitHub and no secrets are using the GitHub App.
		if !doesRepoHaveSecrets(repo) {
			if repo.Spec.GitProvider.Type != gitHubType {
				fmt.Printf("Warning: Repository %s does not have webhook and PACs secrets and has a "+
					"%s URL. Skipping...\n", repo.Metadata.Name, repo.Spec.GitProvider.Type)
			}
			continue
		}

		fmt.Println("---")
		fmt.Printf("Found repository %s in namespace: %s\n", repo.Metadata.Name, repo.Metadata.Namespace)
		fmt.Printf("DEBUG: Secret name: %s\n", repo.Spec.GitProvider.WebhookSecret.Name)
		fmt.Printf("DEBUG: Secret key name: %s\n", repo.Spec.GitProvider.WebhookSecret.Key)

		repos = append(repos, repo)
	}

	return repos, nil
}
