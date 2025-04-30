#!/usr/bin/env python
""" A small program to handle the GitHub App Manifest flow.

https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest
"""

import base64
import http.server
import json
import random
import string
import sys
import urllib.parse
import webbrowser

import requests

expected_state = "".join(
    random.choice(string.ascii_letters + string.digits) for _ in range(16)
)

docs = "https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest"


def parse_input(argv):
    try:
        _, organization, manifest = argv
    except ValueError:
        print("Usage:\n\tgithub-app-flow.py <organization> <manifest>\n")
        print(docs)
        sys.exit(1)
    manifest = base64.b64decode(manifest).decode("utf-8")
    return organization, manifest


class BaseHandler(http.server.BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.organization, self.manifest = parse_input(sys.argv)
        super(BaseHandler, self).__init__(*args, **kwargs)

    def log_message(self, format, *args):
        return  # Suppress logging


class Handler(BaseHandler):
    def do_GET(self):
        # Parse query string
        query = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        code = query.get("code", [None])[0]
        state = query.get("state", [None])[0]
        if not code:
            self.handle_redirect_to_github()
        else:
            if expected_state != state:
                self.send_response(400)
                self.send_header("Content-type", "text/plain")
                self.end_headers()
                message = f"Unexpected state value received: '{state}'; expected '{expected_state}'"
                self.wfile.write(message.encode("utf-8"))
                raise ValueError(
                    message
                )  # Crash out so that expected_state can't be used again.
            else:
                self.handle_redirect_from_github(code, state)

    def handle_redirect_to_github(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        url = f"https://github.com/organizations/{self.organization}/settings/apps/new?state={expected_state}"
        redirect_page = f"""
          <!DOCTYPE html>
          <html><body>
            <form id="form" action="{url}" method="post">
             <input type="text" name="manifest" id="manifest">
             <input type="submit" value="Submit">
            </form>
            <script>
             document.getElementById("manifest").value = JSON.stringify({self.manifest})
             document.getElementById("form").submit()  // send them to github immediately
            </script>
          </body></html>
        """.strip()
        self.wfile.write(redirect_page.encode("utf-8"))

    def handle_redirect_from_github(self, code, state):
        self.send_response(200)
        self.send_header("Content-type", "text/plain")
        self.end_headers()
        update = f"got code {code}.\nrequesting app creation...\n"
        self.wfile.write(update.encode("utf-8"))
        url = f"https://api.github.com/app-manifests/{code}/conversions"
        try:
            response = requests.post(url)

            update = f"{response}\n"
            self.wfile.write(update.encode("utf-8"))

            if response:
                update = "Success!\n"
                self.wfile.write(update.encode("utf-8"))
            else:
                response.raise_for_status()

            details = response.json()

            output = dict(
                application_id=details["id"],
                client_id=details["client_id"],
                webhook_secret=details["webhook_secret"],
                pem=details["pem"],
            )
            print(json.dumps(output))  # To stdout!  To be consumed by ansible.
        except Exception as e:
            update = f"{e}\n"
            self.wfile.write(update.encode("utf-8"))
        self.wfile.write("Playbook will proceed... see your terminal.".encode("utf-8"))


if __name__ == "__main__":
    port = 8089
    server = http.server.HTTPServer(("", port), Handler)
    localurl = f"http://localhost:{port}"
    print(f"Opening {localurl} ...", file=sys.stderr)
    webbrowser.open(localurl)
    print(f"Serving form submission request on {localurl} ...", file=sys.stderr)
    server.handle_request()  # Handle the POST to github
    print(f"Awaiting redirect from github request on {localurl} ...", file=sys.stderr)
    server.handle_request()  # Handle the return from github
    print("Shutting down.", file=sys.stderr)
