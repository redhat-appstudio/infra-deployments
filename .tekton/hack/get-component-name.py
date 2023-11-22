#!/usr/bin/env python3
import argparse
import yaml
import re

def update_app_path(app_path, app_name):
    with open(app_path, "a") as f:
        f.write(app_name + "\n")

def main(args):
    try:
        APP_FILE_NAME = args.filepath

        # Load applicationset file
        with open(APP_FILE_NAME) as f:
            applications = yaml.safe_load(f)

        keys_pattern = r"\b(" + "|".join(re.escape(key) for key in applications.keys()) + r")\b"

        # Print the list of file paths
        apps = []
        for file in open(args.pr_filepath).readlines():
            file = file.strip()
            path_matches = re.findall(keys_pattern, file)
            apps.extend(path_matches)

        # Find app match in pull request files
        unique_apps = set(apps)

        if not unique_apps:
            print(f'No matching components found in pull request files. Add them to the {APP_FILE_NAME} file.')
        else:
            for app in unique_apps:
                for app_name in applications.get(app, []):
                    update_app_path(args.app_filepath, app + ":" + app_name)
    except Exception as e:
        print(f"Error while updating the applicationset name into file: {str(e)}")

def parse_args():
    parser = argparse.ArgumentParser(description="update a file with all the applicationsets for application changed in PR")
    parser.add_argument("--filepath", "-f", required=True)
    parser.add_argument("--app-filepath", "-af", required=True)
    parser.add_argument("--pr-filepath", "-pf", required=True)
    return parser.parse_args()

if __name__ == '__main__':
    main(parse_args())
