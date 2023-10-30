import requests
import sys
from datetime import datetime


def get_headers(github_token: str):
    headers = {
        "authorization": "Bearer " + github_token,
        "content-type": "application/vnd.github.v3+json",
    }

    return headers


def get_branches_url(page: int):
    return f"https://api.github.com/repos/redhat-appstudio-qe/infra-deployments/branches?protected=false&per_page=20&page={page}"


def delete_branch(branch_name: str, headers: dict[str, str]):
    delete_branch_url = (
        f"https://api.github.com/repos/rsoaresd/multi-components/git/refs/heads/{branch_name}"
    )

    response = requests.request(method="DELETE", url=delete_branch_url, headers=headers)
    if response.status_code != 204:
        raise RuntimeError(f"Error deleting branch `{branch_name}`: {response.json()}")
    else:
        print(f"Branch `{branch_name}` was deleted!")


def is_branch_old(branch_commit_url: str, age: int, headers: dict[str, str]):
    response = requests.get(url=branch_commit_url, headers=headers)

    if response.status_code != 200:
        raise RuntimeError(
            f"Error making request to {branch_commit_url}: {response.json()}"
        )

    branch_commit = response.json().get("commit", {})
    committer = branch_commit.get("committer", {})
    committer_date = committer.get("date")

    last_commit_date = datetime.strptime(committer_date, "%Y-%m-%dT%H:%M:%SZ")
    current_date = datetime.now()
    diff = (current_date - last_commit_date).days

    is_branch_old = diff >= age

    return is_branch_old


def delete_branches(github_token: str, age: int):
    page = 0
    headers = get_headers(github_token)
    branches_url = get_branches_url(page)

    response = requests.get(url=branches_url, headers=headers)
    if response.status_code != 200:
        raise RuntimeError(f"Error making request to {branches_url}: {response.json()}")

    branches = response.json()

    while len(branches) > 0:
        for branch in branches:
            branch_name = branch.get("name")
            if branch_name.startswith("preview-") and is_branch_old(
                branch.get("commit", {}).get("url"), age, headers
            ):
               delete_branch(branch_name, headers)

        # request next page
        page += 1
        branches_url = get_branches_url(page)

        response = requests.get(url=branches_url, headers=headers)
        if response.status_code != 200:
            raise RuntimeError(
                f"Error making request to {branches_url}: {response.json()}"
            )

        branches = response.json()


def main():
    args = sys.argv

    if len(args) == 3:
        github_token = args[1]
        age = int(args[2])

        delete_branches(github_token, age)


main()
