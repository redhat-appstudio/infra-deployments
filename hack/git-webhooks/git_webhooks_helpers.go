package gitwebhooks

import (
	"bufio"
	"encoding/base64"
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
)

type RepoSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// Repository represents a single Kubernetes Repository resource.
type Repository struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		URL         string `json:"url"`
		Type        string `json:"type"`
		GitProvider struct {
			URL           string     `json:"url"`
			WebhookSecret RepoSecret `json:"webhook_secret"`
			PacsSecret    RepoSecret `json:"secret"`
		} `json:"git_provider"`
	} `json:"spec"`
}

// doesRepoHaveSecrets checks if a Repository has both webhook and pacs secrets.
func doesRepoHaveSecrets(repo Repository) bool {
	return repo.Spec.GitProvider.WebhookSecret != RepoSecret{} &&
		repo.Spec.GitProvider.PacsSecret != RepoSecret{}
}

// getSecretToken retrieves the one of the Repository's secret tokens from the cluster.
func getSecretToken(repo Repository, secretType string) (string, error) {
	// Determine which secret to retrieve.
	var repoSecret RepoSecret
	if secretType == webhookSecretType {
		repoSecret = repo.Spec.GitProvider.WebhookSecret
	} else if secretType == pacsSecretType {
		repoSecret = repo.Spec.GitProvider.PacsSecret
	} else {
		return "", fmt.Errorf("invalid secret type: %s", secretType)
	}

	// Get the secret from the cluster.
	secretCmd := exec.Command("oc", "get", "secret", repoSecret.Name, "-n", repo.Metadata.Namespace,
		"-o", "json")
	secretOutput, err := secretCmd.Output()
	if err != nil {
		return "", fmt.Errorf("error retrieving secret '%s': %v\n", repoSecret.Name, err)
	}

	// Unmarshal the secret JSON.
	var secret k8s.Secret
	err = json.Unmarshal(secretOutput, &secret)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling secret JSON for '%s': %v\n", repoSecret.Name, err)
	}

	// Retrieve and decode the secret token.
	secretKeyToken, ok := secret.Data[repoSecret.Key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in secret '%s'", repoSecret.Key, repoSecret.Name)
	}
	decodedToken, err := base64.StdEncoding.DecodeString(string(secretKeyToken))
	if err != nil {
		return "", fmt.Errorf("error decoding base64 data for secret '%s': %v\n", repoSecret.Name, err)
	}

	return string(decodedToken), nil
}

// getSpecialExternalRepos retrieves all Repositories that either have:
// 1. a git provider URL of "gitlab.com" (external to Red Hat's gitlab.cee.redhat.com)
// 2. a git provider URL of "github.com" and have secrets (thus, not using the Konflux GitHub App)
func getSpecialExternalRepos() ([]Repository, error) {
	var repos []Repository

	fmt.Println("Searching for Repository resources with external Git provider URLs '" + gitLabComURL +
		"' and '" + gitHubComURL + "'...")

	// Get all Repository resources with a git provider URL of either 'https://gitlab.com'
	// or 'https://github.com'.
	cmd := exec.Command("bash", "-c", `oc get repository -A -o json | jq -c '.items[] |
	 select(.spec.git_provider.url == "`+gitLabComURL+`" or .spec.git_provider.url == "`+gitHubComURL+`")'`)
	output, err := cmd.Output()
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

		// Print a warning message if the repository does not have secrets but has a GitLab URL.
		// Repositories with a GitHub URL and no secrets are using the GitHub App.
		if !doesRepoHaveSecrets(repo) {
			if repo.Spec.GitProvider.URL != gitHubComURL {
				fmt.Printf("Warning: Repository %s does not have webhook and PACs secrets and has a "+
					"GitLab URL. Skipping...\n", repo.Metadata.Name)
			}
			continue
		}

		fmt.Println("---")
		fmt.Printf("Found repository %s in namespace: %s\n", repo.Metadata.Name, repo.Metadata.Namespace)
		fmt.Printf("Secret name: %s\n", repo.Spec.GitProvider.WebhookSecret.Name)
		fmt.Printf("Secret key name: %s\n", repo.Spec.GitProvider.WebhookSecret.Key)

		repos = append(repos, repo)
	}

	return repos, nil
}
