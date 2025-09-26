package main

import (
	"flag"
	"fmt"
	"os"

	gitlab_webhooks "gitlab-webhooks"
)

var (
	create, delete, dryRun bool
	webhookURL, namespace  string
)

func main() {
	flag.BoolVar(&create, "create", false, "Flag to create the webhook")
	flag.BoolVar(&delete, "delete", false, "Flag to delete the webhook")
	flag.StringVar(&webhookURL, "webhook-url", "", "Required: The webhook URL to reference")
	flag.BoolVar(&dryRun, "dry-run", false, "Optional: Flag to dry run the webhook creation")
	flag.StringVar(&namespace, "namespace", "", "Optional: The namespace to use for the webhook")
	flag.Parse()

	// Check if webhook URL is provided
	if webhookURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --webhook-url is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check that exactly one action is specified
	if create && delete {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --create and --delete\n")
		flag.Usage()
		os.Exit(1)
	}

	if create {
		err := gitlab_webhooks.CreateGitWebhooks(webhookURL, dryRun, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if delete {
		err := gitlab_webhooks.DeleteGitWebhooks(webhookURL, dryRun, namespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: must specify either --create or --delete\n")
		flag.Usage()
		os.Exit(1)
	}
}
