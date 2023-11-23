#!/usr/bin/env python3
import json
import os
import requests
import argparse

github_api_url = os.environ.get('GITHUB_API_URL', "https://api.github.com")

def get_labels_name(args):
    req = requests.get(
      f"{github_api_url}/repos/{args.git_owner}/{args.git_repo}/issues/{args.pr_number}",
      headers={
        "Authorization": f"Bearer {args.token}"
      },
      data=json.dumps({}))
    response = req.json()
    label_names = [label["name"] for label in response["labels"]]
    return label_names

def remove_label(args, label_names):
    # Check if the label to remove exists on the pull request
    if args.label in label_names:
        # Remove the label
        label_names.remove(args.label)

        # Update the labels on the pull request
        req = requests.patch(
            f"{github_api_url}/repos/{args.git_owner}/{args.git_repo}/issues/{args.pr_number}",
            headers={
                "Authorization": f"Bearer {args.token}"
            },
            data=json.dumps({"labels": label_names}))
        if req.status_code == 200:
          print("label {args.label} has been removed successfully")
        else:
          raise Exception("Failed to remove the label {args.label} from the pull request.")

def parse_args():
    parser = argparse.ArgumentParser(description="remove pull request labels")
    parser.add_argument("--label", "-l", required=True)
    parser.add_argument("--token", "-t", required=True)
    parser.add_argument("--git-owner", "-o", required=True)
    parser.add_argument("--git-repo", "-r", required=True)
    parser.add_argument("--pr-number", "-n", required=True)
    return parser.parse_args()


def main(args):
    label_names = get_labels_name(args)
    remove_label(args, label_names)


if __name__ == "__main__":
    main(parse_args())
