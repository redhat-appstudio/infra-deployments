// This module is a stop-gap solution for the rare case of when the smee server is
// migrated to another cluster. This is necessary because any tenant namespace created
// with KubeSaw (before its deprecation) has a webhook that points directly to the smee
// server and the server URL contains the hosting cluster name.
//
// To minimize down-time for these tenants, any Repository resource on the target cluster with either
// 1. a git provider URL of "gitlab.com" (external to Red Hat's gitlab.cee.redhat.com)
// 2. a git provider URL of "github.com" and secrets (not using the Konflux GitHub App)
// will get a new webhook with the new smee server URL. Once functionality for this new smee
// server is verified, the old webhook should be deleted from each of the previous repositories.

package gitwebhooks

import (
	"fmt"

	"github.com/konflux-ci/build-service/pkg/git/github"
	"github.com/konflux-ci/build-service/pkg/git/gitlab"
)

// CreateGitWebhooks creates a new webhook (with a URL of 'webhookURL') for each repository with a
// webhook pointing directly to the smee server (in the provided namespace). If `dryRun` is true,
// the webhook will not be created, but the function will print a list of repositories that would
// have webhooks created.
func CreateGitWebhooks(webhookURL string, dryRun bool, namespace string) error {
	repos, err := getSpecialExternalRepos(nil, namespace)
	if err != nil {
		return fmt.Errorf("error getting special external repositories: %v\n", err)
	}
	fmt.Println("\nCreating webhooks...\n")

	if dryRun {
		fmt.Printf("DRY RUN: Would create webhooks for %d repositories\n", len(repos))
	}

	for _, repo := range repos {
		fmt.Printf("---\nRepo: %s\n", repo.Metadata.Name)
		repoWebhookToken, err := getSecretToken(repo, webhookSecretType, nil)
		if err != nil {
			fmt.Printf("WARNING: error getting webhook secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		repoPacsToken, err := getSecretToken(repo, pacsSecretType, nil)
		if err != nil {
			fmt.Printf("WARNING: error getting PaC secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		if dryRun {
			fmt.Printf("DRY RUN: Would create webhook for repository %s (%s) with URL %s\n",
				repo.Metadata.Name, repo.Spec.URL, webhookURL)
			continue
		}

		if repo.Spec.GitProvider.URL == gitLabComURL {
			gitlabClient, err := gitlab.NewGitlabClient(repoPacsToken, repo.Spec.GitProvider.URL)
			if err != nil {
				fmt.Printf("WARNING: error creating GitLab client for repository %s: %v\n",
					repo.Metadata.Name, err)
				continue
			}
			fmt.Printf("DEBUG: Created a GitLab client to repo %s\n", repo.Metadata.Name)
			err = gitlabClient.SetupPaCWebhook(repo.Spec.URL, webhookURL, repoWebhookToken)
			if err != nil {
				fmt.Printf("WARNIING: error creating a webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			} else {
				fmt.Printf("Created a webhook for GitLab repo %s successfully!\n", repo.Metadata.Name)
			}
		} else {
			githubClient := github.NewGithubClient(repoPacsToken)
			err = githubClient.SetupPaCWebhook(repo.Spec.URL, webhookURL, repoWebhookToken)
			if err != nil {
				fmt.Printf("WARNING: error creating a webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			} else {
				fmt.Printf("Created a webhook for GitHub repo %s successfully!\n", repo.Metadata.Name)
			}
		}

	}
	return nil
}

// DeleteGitWebhooks deletes the webhook with URL 'webhookURL' for each repository
// with a webhook pointing directly to the smee server (in the provided namespace).
// `webhookURL` should be the URL of the old smee server. If `dryRun` is true,
// the webhook will not be deleted, but the function will print a list of repositories
// that would have webhooks deleted.
func DeleteGitWebhooks(webhookURL string, dryRun bool, namespace string) error {
	repos, err := getSpecialExternalRepos(nil, namespace)
	if err != nil {
		return fmt.Errorf("error getting special external repositories: %v\n", err)
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would delete webhooks for %d repositories\n", len(repos))
	}

	fmt.Println("\nDeleting webhooks...\n")
	for _, repo := range repos {
		fmt.Printf("---\nRepo: %s\n", repo.Metadata.Name)
		repoPacsToken, err := getSecretToken(repo, pacsSecretType, nil)
		if err != nil {
			fmt.Printf("WARNING: error getting PaC secret token for repository %s: %v\n",
				repo.Metadata.Name, err)
			continue
		}

		if dryRun {
			fmt.Printf("DRY RUN: Would delete webhook for repository %s (%s) with URL %s\n",
				repo.Metadata.Name, repo.Spec.URL, webhookURL)
			continue
		}

		if repo.Spec.GitProvider.URL == gitLabComURL {
			gitlabClient, err := gitlab.NewGitlabClient(repoPacsToken, repo.Spec.GitProvider.URL)
			if err != nil {
				fmt.Printf("WARNING: error creating GitLab client for repository %s: %v\n",
					repo.Metadata.Name, err)
				continue
			}
			fmt.Printf("DEBUG: Created a GitLab client to repo %s\n", repo.Metadata.Name)

			err = gitlabClient.DeletePaCWebhook(repo.Spec.URL, webhookURL)
			if err != nil {
				fmt.Printf("WARNING: error deleting the webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			} else {
				fmt.Printf("Deleted the webhook for GitHub repo %s successfully!\n", repo.Metadata.Name)
			}
		} else {
			githubClient := github.NewGithubClient(repoPacsToken)
			err = githubClient.DeletePaCWebhook(repo.Spec.URL, webhookURL)
			if err != nil {
				fmt.Printf("WARNING: error deleting the webhook with URL %s for repository %s: %v\n",
					webhookURL, repo.Metadata.Name, err)
			} else {
				fmt.Printf("Deleted the webhook for GitHub repo %s successfully!\n", repo.Metadata.Name)
			}
		}
	}
	return nil
}
