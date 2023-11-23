#!/usr/bin/env python3
import os
import requests
import argparse

github_graphql_api_url = os.environ.get('GITHUB_GRAPHQL_API_URL', "https://api.github.com/graphql")

def run_query(query, variables, headers={}):
    url = f"{github_graphql_api_url}"
    request = requests.post(url, json={'query': query, 'variables': variables}, headers=headers)
    if request.status_code == 200:
        return request.json()
    else:
        raise Exception("Query failed to run by returning code of {}. {}".format(request.status_code, query))

def get_pr_id_branch_oid(args):
    get_pr_id_branch_oid_query_variables = {
      "owner": f"{args.git_owner}",
      "repo": f"{args.git_repo}",
      "pr_number": int(f"{args.pr_number}"),
      "source_branch_name": f"{args.branch}",
    }
    get_pr_id_branch_oid_query = """
    query GetPullRequestIDs($owner:String!, $repo:String!, $pr_number:Int!, $source_branch_name:String!) {
      repository(owner:$owner, name:$repo) {
        pullRequest(number: $pr_number) {
          id
        }
        ref(qualifiedName: $source_branch_name) {
          target {
            oid
          }
        }
      }
    }
    """
    result = run_query(
        get_pr_id_branch_oid_query,
        get_pr_id_branch_oid_query_variables,
        headers={
            "Authorization": f"Bearer {args.token}"
        })
    pr_id = result["data"]["repository"]["pullRequest"]["id"]
    source_branch_oid = result["data"]["repository"]["ref"]["target"]["oid"]
    return pr_id, source_branch_oid

def merge_pr(args, pr_id, source_branch_oid):
    merge_pr_varaibles = {
      "pullrequest_id": str(pr_id),
      "expectedHeadOid": source_branch_oid,
      "mergeMethod" : 'MERGE',
    }

    merge_pr_mutation = """
    mutation MergePullRequest($pullrequest_id:ID!, $expectedHeadOid:GitObjectID, $mergeMethod:PullRequestMergeMethod) {
      mergePullRequest(input:{pullRequestId: $pullrequest_id, expectedHeadOid: $expectedHeadOid, mergeMethod: $mergeMethod}) {
        clientMutationId
        pullRequest {
          id
        }
      }
    }
    """

    result = run_query(
        merge_pr_mutation,
        merge_pr_varaibles,
        headers={
            "Authorization": f"Bearer {args.token}"
        })
    if "errors" in result.keys():
        for error in result["errors"]:
            print("ERROR: {0}".format(error["message"]))
    else:
        print("SUCCESS: merged PR id {0}".format(result["data"]["mergePullRequest"]["pullRequest"]["id"]))

def parse_args():
    parser = argparse.ArgumentParser(description="merge pull request")
    parser.add_argument("--branch", "-b", required=True)
    parser.add_argument("--token", "-t", required=True)
    parser.add_argument("--git-owner", "-o", required=True)
    parser.add_argument("--git-repo", "-r", required=True)
    parser.add_argument("--pr-number", "-n", required=True)
    return parser.parse_args()


def main(args):
    pr_id, source_branch_oid = get_pr_id_branch_oid(args)
    merge_pr(args, pr_id, source_branch_oid)


if __name__ == "__main__":
    main(parse_args())
