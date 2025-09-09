// This module is a stop-gap solution for the rare case of when the smee server is
// migrated to another cluster. This is necessary because any tenant namespace created
// with KubeSaw (before its deprecation) has a webhook that points directly to the smee
// server and the server URL contains the hosting cluster name.
//
// To minimize down-time for these tenants, any Repository resource on the target cluster with either
// 1. a git provider URL of "gitlab.com" (external to Red Hat's gitlab.cee.redhat.com)
// 2. a git provider URL of "github.com" and secrets (not using the Konflux GitHub App)
// will get a new webhook with the new smee server URL. Once functionality for this new smee
// server is verified, the old webhook will be deleted from each of the previous repositories.

package gitwebhooks

import (
	"fmt"

	"github.com/konflux-ci/build-service/pkg/git/github"
	"github.com/konflux-ci/build-service/pkg/git/gitlab"
)

// CreateGitWebhooks creates a new webhook (with a URL of 'webhookURL') for each repository with a
// webhook pointing directly to the smee server.
func CreateGitWebhooks(webhookURL string) error {
	repos, err := getSpecialExternalRepos(nil)
	if err != nil {
		return fmt.Errorf("error getting special external repositories: %v\n", err)
	}

	for _, repo := range repos {
		repoWebhookToken, err := getSecretToken(repo, webhookSecretType, nil)
		if err != nil {
			fmt.Printf("Warning: error getting webhook secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		repoPacsToken, err := getSecretToken(repo, pacsSecretType, nil)
		if err != nil {
			fmt.Printf("Warning: error getting PaC secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		if repo.Spec.GitProvider.URL == gitLabComURL {
			gitlabClient, err := gitlab.NewGitlabClient(repoPacsToken, repo.Spec.GitProvider.URL)
			if err != nil {
				fmt.Printf("Warning: error creating GitLab client for repository %s: %v\n",
					repo.Metadata.Name, err)
				continue
			}
			err = gitlabClient.SetupPaCWebhook(repo.Spec.URL, webhookURL, repoWebhookToken)
			if err != nil {
				fmt.Printf("Warning: error creating a webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			}
		} else {
			githubClient := github.NewGithubClient(repoPacsToken)
			err = githubClient.SetupPaCWebhook(repo.Spec.URL, webhookURL, repoWebhookToken)
			if err != nil {
				fmt.Printf("Warning: error creating a webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			}
		}

	}
	return nil
}

// DeleteGitWebhooks deletes the webhook with URL 'webhookURL' for each repository
// with a webhook pointing directly to the smee server. `webhookURL` should be the URL of the old
// smee server.
func DeleteGitWebhooks(webhookURL string) error {
	repos, err := getSpecialExternalRepos(nil)
	if err != nil {
		return fmt.Errorf("error getting special external repositories: %v\n", err)
	}

	for _, repo := range repos {
		repoPacsToken, err := getSecretToken(repo, pacsSecretType, nil)
		if err != nil {
			fmt.Printf("Warning: error getting PaC secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		if repo.Spec.GitProvider.URL == gitLabComURL {
			gitlabClient, err := gitlab.NewGitlabClient(repoPacsToken, repo.Spec.GitProvider.URL)
			if err != nil {
				fmt.Printf("Warning: error creating GitLab client for repository %s: %v\n",
					repo.Metadata.Name, err)
				continue
			}

			err = gitlabClient.DeletePaCWebhook(repo.Spec.URL, webhookURL)
			if err != nil {
				fmt.Printf("Warning: error deleting the webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			}
		} else {
			githubClient := github.NewGithubClient(repoPacsToken)
			err = githubClient.DeletePaCWebhook(repo.Spec.URL, webhookURL)
			if err != nil {
				fmt.Printf("Warning: error deleting the webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			}
		}
	}
	return nil
}
