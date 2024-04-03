import argparse
import itertools
import json
import logging
import os
import re
import time

from collections.abc import Iterator
from http.client import HTTPResponse
from typing import Any, Dict, List
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen
from datetime import datetime

logging.basicConfig(
    format="%(asctime)s - %(levelname)s - %(message)s", level=logging.INFO
)
LOGGER = logging.getLogger(__name__)
QUAY_API_URL = "https://quay.io/api/v1"
DAY_OLD_TS = int(datetime.now().timestamp()) - (60 * 60 * 24)
KEEP_MAX = 3

ImageRepo = Dict[str, Any]


def get_quay_tags(quay_token: str, namespace: str, name: str) -> ImageRepo:
    next_page = None
    resp: HTTPResponse

    all_tags = []
    while True:
        query_args = {"limit": 100, "onlyActiveTags": True}
        if next_page is not None:
            query_args["page"] = next_page

        api_url = f"{QUAY_API_URL}/repository/{namespace}/{name}/tag/?{urlencode(query_args)}"
        request = Request(api_url, headers={
            "Authorization": f"Bearer {quay_token}",
        })

        with urlopen(request) as resp:
            if resp.status != 200:
                raise RuntimeError(resp.reason)
            json_data = json.loads(resp.read())

        tags = json_data.get("tags", [])
        all_tags.extend(tags)

        if not tags:
            LOGGER.debug("No tags found.")
            break

        page = json_data.get("page", None)
        additional = json_data.get("has_additional", False)

        if additional:
            next_page = page + 1
        else:
            break

    return all_tags


def quay_test_token(quay_token: str, namespace: str) -> None:
    api_url = f"{QUAY_API_URL}/organization/{namespace}/applications"
    request = Request(api_url, headers={
        "Authorization": f"Bearer {quay_token}",
    })
    try:
        urlopen(request)
    except HTTPError as ex:
        # if status is 401 that means that token is wrong
        if ex.status == 401:
            raise RuntimeError("Wrong quay token")


def delete_image_tag(quay_token: str, namespace: str, name: str, tag: str) -> None:
    api_url = f"{QUAY_API_URL}/repository/{namespace}/{name}/tag/{tag}"
    request = Request(api_url, method="DELETE", headers={
        "Authorization": f"Bearer {quay_token}",
    })
    resp: HTTPResponse

    while True:
        try:
            with urlopen(request) as resp:
                if resp.status != 200 and resp.status != 204 and resp.status != 404:
                    raise RuntimeError(resp.reason)
                else:
                    break
        except HTTPError as ex:
            LOGGER.info("HTTPError exception: %s", ex)


def remove_leftover_tags(tags: List[Dict[str, Any]], quay_token: str, namespace: str, name: str,
                         dry_run: bool = False) -> None:
    tag_regex = re.compile(r"^sha256-([0-9a-f]+)(\.sbom|\.att|\.src|\.sig)$")

    # remove att/sbom/src/sig for which is missing manifest digest
    image_digests = [image["manifest_digest"] for image in tags]

    for tag in tags:
        if (match := tag_regex.match(tag["name"])) is not None:
            if f"sha256:{match.group(1)}" not in image_digests:
                if dry_run:
                    LOGGER.info("Leftover image %s from %s/%s should be removed", tag["name"], namespace, name)
                else:
                    LOGGER.info("Removing leftover image %s from %s/%s", tag["name"], namespace, name)
                    delete_image_tag(quay_token, namespace, name, tag["name"])


def remove_tags(tags: List[Dict[str, Any]], quay_token: str, namespace: str, name: str,
                days_old: int, keep_max: int, dry_run: bool = False) -> None:
    unique_names = {}
    removed_digests = []

    # first remove only named tags
    for tag in tags:
        # skip att/sbom/src/sig
        if tag["name"].startswith("sha256-") or "-" not in tag["name"]:
            continue

        tag_name, _ = tag["name"].rsplit('-', 1)

        count = unique_names.get(tag_name, 0)

        # keep at least first x per tag name
        if count < keep_max:
            unique_names[tag_name] = count + 1

        # remove older than x
        elif tag["start_ts"] < days_old:
            if dry_run:
                LOGGER.info("Image %s from %s/%s should be removed", tag["name"], namespace, name)
                removed_digests.append(tag["manifest_digest"])
            else:
                LOGGER.info("Removing image %s from %s/%s", tag["name"], namespace, name)
                delete_image_tag(quay_token, namespace, name, tag["name"])
                removed_digests.append(tag["manifest_digest"])

    tag_regex = re.compile(r"^sha256-([0-9a-f]+)(\.sbom|\.att|\.src|\.sig)$")
    # when named tags are removed, remove obsolete sbom/att/src
    for tag in tags:
        if (match := tag_regex.match(tag["name"])) is not None:
            if f"sha256:{match.group(1)}" in removed_digests:
                if dry_run:
                    LOGGER.info("Image %s from %s/%s should be removed", tag["name"], namespace, name)
                else:
                    LOGGER.info("Removing image %s from %s/%s", tag["name"], namespace, name)
                    delete_image_tag(quay_token, namespace, name, tag["name"])


def process_repository(quay_token: str, namespace: str, repo_name: str, days_old: int,
                       keep_max: int, dry_run: bool = False) -> None:
    LOGGER.info("Processing repository: %s/%s", namespace, repo_name)

    quay_test_token(quay_token, namespace)

    all_tags = get_quay_tags(quay_token, namespace, repo_name)
    LOGGER.info("Tag count in repository: %s", len(all_tags))

    if all_tags:
        remove_tags(all_tags, quay_token, namespace, repo_name, days_old, keep_max, dry_run=dry_run)

    all_tags = get_quay_tags(quay_token, namespace, repo_name)
    LOGGER.info("Tag count in repository: %s", len(all_tags))

    if all_tags:
        remove_leftover_tags(all_tags, quay_token, namespace, repo_name, dry_run=dry_run)


def main():
    token = os.getenv("QUAY_TOKEN")
    if not token:
        raise ValueError("The token required for access to Quay API is missing!")

    args = parse_args()
    process_repository(token, args.namespace, args.repo_name, days_old=args.old_days,
                       keep_max=args.keep_max, dry_run=args.dry_run)


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--namespace", required=True)
    parser.add_argument("--repo-name", required=True)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--old-days", type=int, default=DAY_OLD_TS)
    parser.add_argument("--keep-max", type=int, default=KEEP_MAX)

    args = parser.parse_args()
    return args


if __name__ == "__main__":
    main()
