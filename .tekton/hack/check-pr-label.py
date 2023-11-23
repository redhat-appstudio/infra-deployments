#!/usr/bin/env python3
import json
import os
import requests
import argparse

def get_labels_name(args):
    github_api_url = os.environ.get('GITHUB_API_URL', "https://api.github.com")
    req = requests.get(
      f"{github_api_url}/repos/{args.git_owner}/{args.git_repo}/issues/{args.pr_number}",
      headers={
        "Authorization": f"Bearer {args.token}"
      },
      data=json.dumps({}))
    response = req.json()
    label_names = [label["name"] for label in response["labels"]]
    return label_names

def check_label_exist(args, label_names):
    if not label_names or args.label not in label_names:
      print("Label does not exists")
      return False
    else:
      print("Label matched")
      return True


def parse_args():
    parser = argparse.ArgumentParser(description="check pull request labels")
    parser.add_argument("--filepath", "-f", required=True)
    parser.add_argument("--label", "-l", required=True)
    parser.add_argument("--token", "-t", required=True)
    parser.add_argument("--git-owner", "-o", required=True)
    parser.add_argument("--git-repo", "-r", required=True)
    parser.add_argument("--pr-number", "-n", required=True)
    return parser.parse_args()


def main(args):
    label_names = get_labels_name(args)
    matched_label_status = check_label_exist(args, label_names)
    with open(args.filepath, "w") as f:
      f.write(str(matched_label_status))


if __name__ == "__main__":
    main(parse_args())
