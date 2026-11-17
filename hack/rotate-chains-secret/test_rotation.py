"""Offline integration tests: real Ansible, fake Vault/Cosign/Kustomize CLIs."""

import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

PLAYBOOK = Path(__file__).with_name("playbook.yaml")
BOOTSTRAP_TASKS = PLAYBOOK.resolve().parents[1] / "new-cluster/tasks/vault/chains.yaml"
FAKE_CLI = r"""#!/usr/bin/python3
import json
import os
from pathlib import Path
import sys
import yaml

state_file = Path(os.environ["ROTATION_TEST_STATE"])
state = json.loads(state_file.read_text())
tool = Path(sys.argv[0]).name
args = sys.argv[1:]

def finish(output="", code=0):
    state_file.write_text(json.dumps(state))
    print(output)
    sys.exit(code)

if tool == "kustomize":
    overlay = Path(args[-1])
    docs = []
    for name, field in [("tekton-chains-signing-secret", "/spec/dataFrom/0/extract/key"),
                        ("tekton-chains-public-key", "/spec/data/0/remoteRef/key")]:
        patch = yaml.safe_load((overlay / (name + ".yaml")).read_text())
        key = patch[0]["value"]
        spec = {"secretStoreRef": {"kind": "ClusterSecretStore", "name": "appsre-stonesoup-vault"}}
        if "signing" in name:
            spec["dataFrom"] = [{"extract": {"key": key}}]
        else:
            spec["data"] = [{"secretKey": "cosign.pub", "remoteRef": {"key": key, "property": "cosign.pub"}}]
        if state.get("pinned"):
            source = spec["dataFrom"][0]["extract"] if "signing" in name else spec["data"][0]["remoteRef"]
            source["version"] = "1"
        docs.append({"kind": "ExternalSecret", "metadata": {"name": name, "namespace": "openshift-pipelines"}, "spec": spec})
    finish(yaml.safe_dump_all(docs))

if tool == "openssl":
    state["events"].append({"operation": "password"})
    finish("PASSWORD-SENTINEL")

if tool == "cosign":
    prefix = Path(args[args.index("--output-key-prefix") + 1]) if "--output-key-prefix" in args else Path.cwd() / "cosign"
    state["temporary_dirs"].append(str(prefix.parent))
    if state.get("fail_generation"):
        finish("PRIVATE-SENTINEL PASSWORD-SENTINEL", 1)
    assert os.environ["COSIGN_PASSWORD"] == "PASSWORD-SENTINEL"
    prefix.with_suffix(".key").write_text("PRIVATE-SENTINEL")
    prefix.with_suffix(".pub").write_text("PUBLIC-" + prefix.parent.name)
    finish("Generated keypair")

assert tool == "vault", tool
token = os.environ.get("VAULT_TOKEN", "")
session_valid = not state.get("invalid_session") or (not token and state.get("cached_session_valid"))
if args[:2] == ["token", "lookup"]:
    assert args == ["token", "lookup", "-field=ttl"], args
    state.setdefault("auth_events", []).append("lookup")
    if not session_valid:
        finish("permission denied: invalid token", 2)
    finish("3600")
if args[0] == "login":
    assert "-method=oidc" in args and "-format=json" in args, args
    assert not token, "An expired environment token must not override OIDC login"
    state.setdefault("auth_events", []).append("login")
    if state.get("fail_login"):
        finish("TOKEN-SENTINEL OIDC login failed", 2)
    state["cached_session_valid"] = True
    finish(json.dumps({"auth": {"client_token": "REFRESHED-TOKEN-SENTINEL"}}))
if not session_valid:
    finish("permission denied: invalid token", 2)
if state.get("expected_token"):
    assert token == state["expected_token"]
# Bootstrap and rotation must work with secret-scoped permissions only.
assert args[0] == "kv", args
if args[1:3] == ["metadata", "get"]:
    path = args[-1].removeprefix("stonesoup/")
    state["events"].append({"operation": "metadata", "path": path})
    if state.get("deny_metadata") == path:
        finish("permission denied", 2)
    secret = state["secrets"].get(path)
    if secret is None:
        finish(f"No value found at stonesoup/metadata/{path}", 2)
    version = secret["version"]
    finish(json.dumps({"data": {"current_version": version, "max_versions": secret.get("max_versions", 0), "versions": {str(version): {
        "destroyed": secret.get("destroyed", False), "deletion_time": secret.get("deletion_time", "")}}}}))
if args[1] == "put":
    path = next(arg for arg in args[2:] if not arg.startswith("-") and "=" not in arg).removeprefix("stonesoup/")
    cas = int(next((arg.split("=", 1)[1] for arg in args if arg.startswith("-cas=")), "-1"))
    current = state["secrets"].get(path, {}).get("version", 0)
    if path == state.get("concurrent_write"):
        current += 1
        state["secrets"][path] = {"version": current, "public": "CONCURRENT-PUBLIC"}
    state["events"].append({"operation": "put", "path": path, "cas": cas})
    if path == state.get("fail_write") or (cas >= 0 and current != cas):
        finish("PRIVATE-SENTINEL PASSWORD-SENTINEL write rejected", 2)
    fields = dict(arg.split("=", 1) for arg in args if arg.startswith("cosign."))
    assert Path(fields["cosign.key"][1:]).read_text() == "PRIVATE-SENTINEL"
    password = fields["cosign.password"]
    assert (Path(password[1:]).read_text().strip() if password.startswith("@") else password) == "PASSWORD-SENTINEL"
    public = Path(fields["cosign.pub"][1:]).read_text()
    state["secrets"][path] = {"version": current + 1, "public": public}
    finish(json.dumps({"data": {"version": current + 1}}))
if args[1] == "get":
    if state.get("deny_data_read"):
        finish("permission denied", 2)
    finish(state["secrets"][args[-1]]["public"])
finish("Unexpected command: " + repr(args), 1)
"""


@unittest.skipUnless(shutil.which("ansible-playbook"), "requires ansible-core")
class RotationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="chains-rotation-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.state_file = self.root / "state.json"
        self.state = {"secrets": {}, "events": [], "temporary_dirs": []}
        self.bin_dir = self.root / "bin"
        self.bin_dir.mkdir()
        (self.root / "ansible.cfg").write_text("[defaults]\n")
        for tool in ("vault", "kustomize", "cosign", "openssl"):
            executable = self.bin_dir / tool
            executable.write_text(FAKE_CLI)
            executable.chmod(0o755)

    def cluster(self, ring, name, legacy=False, version=3):
        env = "staging" if ring == "ring-1" else "production"
        canonical = f"{env}/platform/ansible/generated/{name}/chains-signing-secret"
        source = (
            f"{env}/pipeline-service/{name}/chains-signing-secret"
            if legacy
            else canonical
        )
        overlay = self.root / "components/pipeline-service/rings" / ring / name
        overlay.mkdir(parents=True)
        patches = []
        for secret_name, pointer in (
            ("tekton-chains-signing-secret", "/spec/dataFrom/0/extract/key"),
            ("tekton-chains-public-key", "/spec/data/0/remoteRef/key"),
        ):
            filename = secret_name + ".yaml"
            (overlay / filename).write_text(
                yaml.safe_dump([{"op": "replace", "path": pointer, "value": source}])
            )
            patches.append(
                {
                    "path": filename,
                    "target": {"kind": "ExternalSecret", "name": secret_name},
                }
            )
        (overlay / "kustomization.yaml").write_text(
            yaml.safe_dump({"patches": patches})
        )
        self.state["secrets"][source] = {"version": version, "public": "OLD-PUBLIC"}
        return source, canonical, overlay

    def run_playbook(
        self,
        ring="ring-1",
        check=False,
        clusters=None,
        default_journal=False,
        repo_root=None,
        playbook=None,
        tags=None,
        vault_token="TOKEN-SENTINEL",
    ):
        self.state_file.write_text(json.dumps(self.state))
        variables = {
            "repo_root": str(repo_root or self.root),
            "rotation_log": str(self.root / "rotation.jsonl"),
        }
        if default_journal:
            del variables["rotation_log"]
        if ring is not None:
            variables["ring"] = ring
        if clusters is not None:
            variables["clusters"] = clusters
        command = [
            "ansible-playbook",
            "-i",
            "localhost,",
            str(playbook or PLAYBOOK),
            "-e",
            json.dumps(variables),
        ]
        if check:
            command.append("--check")
        if tags:
            command.extend(["--tags", tags])
        result = subprocess.run(
            command,
            check=False,
            cwd=self.root,
            capture_output=True,
            text=True,
            timeout=90,
            env={
                **os.environ,
                "PATH": f"{self.bin_dir}:{os.environ['PATH']}",
                "ROTATION_TEST_STATE": str(self.state_file),
                "ANSIBLE_LOCAL_TEMP": str(self.root / "ansible-local"),
                "ANSIBLE_REMOTE_TEMP": str(self.root / "ansible-remote"),
                "ANSIBLE_NOCOLOR": "1",
                "ANSIBLE_FORKS": "1",
                "ANSIBLE_CONFIG": str(self.root / "ansible.cfg"),
                "TMPDIR": str(self.root),
                "VAULT_TOKEN": vault_token,
            },
        )
        self.state = json.loads(self.state_file.read_text())
        self.assertNotIn("PRIVATE-SENTINEL", result.stdout + result.stderr)
        self.assertNotIn("PASSWORD-SENTINEL", result.stdout + result.stderr)
        self.assertNotIn("TOKEN-SENTINEL", result.stdout + result.stderr)
        return result

    def bootstrap_playbook(self):
        playbook = self.root / "bootstrap.yaml"
        playbook.write_text(
            yaml.safe_dump(
                [
                    {
                        "name": "Test the real new-cluster Chains entry point",
                        "hosts": "localhost",
                        "connection": "local",
                        "gather_facts": False,
                        "vars": {
                            "env": "staging",
                            "shortname": "stage-a",
                            "vault_addr": "https://vault.invalid",
                            "vault_token": "BOOTSTRAP-TOKEN-SENTINEL",
                        },
                        "tasks": [
                            {"ansible.builtin.import_tasks": str(BOOTSTRAP_TASKS)}
                        ],
                    }
                ]
            )
        )
        self.state["expected_token"] = "BOOTSTRAP-TOKEN-SENTINEL"
        return playbook

    def assert_ok(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def writes(self):
        return [event for event in self.state["events"] if event["operation"] == "put"]

    def assert_cleaned(self):
        self.assertTrue(self.state["temporary_dirs"])
        for path in self.state["temporary_dirs"]:
            self.assertFalse(Path(path).exists(), path)

    def test_requires_one_supported_ring(self):
        for ring in (None, "ring-0", "../ring-2", "all"):
            with self.subTest(ring=ring):
                result = self.run_playbook(ring=ring, check=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.state["events"])

    def test_check_preflights_only_selected_ring_without_writes(self):
        source, _, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        self.cluster("ring-2", "prod-a")
        before = (overlay / "tekton-chains-public-key.yaml").read_text()
        self.assert_ok(self.run_playbook(check=True))
        self.assertTrue(self.state["events"])
        self.assertTrue(
            all(event["operation"] == "metadata" for event in self.state["events"])
        )
        self.assertEqual(self.state["secrets"][source]["version"], 3)
        self.assertEqual(
            (overlay / "tekton-chains-public-key.yaml").read_text(), before
        )
        self.assertFalse((self.root / "rotation.jsonl").exists())

    def test_valid_environment_token_does_not_start_oidc(self):
        self.cluster("ring-1", "stage-a")
        self.state["expected_token"] = "TOKEN-SENTINEL"
        self.assert_ok(self.run_playbook(check=True))
        self.assertEqual(self.state["auth_events"], ["lookup"])

    def test_valid_cached_session_does_not_start_oidc(self):
        self.cluster("ring-1", "stage-a")
        self.assert_ok(self.run_playbook(check=True, vault_token=""))
        self.assertEqual(self.state["auth_events"], ["lookup"])

    def test_expired_environment_token_is_replaced_before_rotation(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        self.state["invalid_session"] = True
        self.assert_ok(self.run_playbook())
        self.assertEqual(self.state["auth_events"], ["lookup", "login"])
        self.assertEqual(self.state["secrets"][source]["version"], 4)
        self.assert_cleaned()

    def test_missing_session_starts_oidc_even_in_check_mode(self):
        source, _, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        before = (overlay / "tekton-chains-public-key.yaml").read_text()
        self.state["invalid_session"] = True
        self.assert_ok(self.run_playbook(check=True, vault_token=""))
        self.assertEqual(self.state["auth_events"], ["lookup", "login"])
        self.assertEqual(self.state["secrets"][source]["version"], 3)
        self.assertEqual(
            (overlay / "tekton-chains-public-key.yaml").read_text(), before
        )
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])
        self.assertFalse((self.root / "rotation.jsonl").exists())

    def test_oidc_failure_stops_before_preflight_without_printing_tokens(self):
        self.cluster("ring-1", "stage-a")
        self.state.update(invalid_session=True, fail_login=True)
        result = self.run_playbook()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state["auth_events"], ["lookup", "login"])
        self.assertFalse(self.state["events"])
        self.assertFalse(self.state["temporary_dirs"])
        self.assertFalse((self.root / "rotation.jsonl").exists())

    def test_secret_permission_failure_does_not_start_oidc(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        self.state["deny_metadata"] = source
        self.assertNotEqual(self.run_playbook(check=True).returncode, 0)
        self.assertEqual(self.state["auth_events"], ["lookup"])
        self.assertFalse(self.writes())

    def test_bootstrap_creates_with_cas_zero_and_preserves_local_files(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        del self.state["secrets"][source]
        self.state["deny_data_read"] = True
        local_key = self.root / "cosign.key"
        local_key.write_text("Existing unrelated local key")
        self.assert_ok(
            self.run_playbook(playbook=self.bootstrap_playbook(), tags="chains")
        )
        self.assertEqual(
            self.writes(), [{"operation": "put", "path": source, "cas": 0}]
        )
        self.assertEqual(local_key.read_text(), "Existing unrelated local key")
        self.assertFalse((self.root / "rotation.jsonl").exists())
        self.assert_cleaned()

    def test_bootstrap_still_skips_an_existing_secret(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        self.assert_ok(
            self.run_playbook(playbook=self.bootstrap_playbook(), tags="chains")
        )
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])
        self.assertEqual(self.state["secrets"][source]["version"], 3)

    def test_rotates_each_selected_cluster_with_cas_and_journal(self):
        first, _, _ = self.cluster("ring-2", "prod-a", version=4)
        second, _, _ = self.cluster("ring-2", "prod-b", version=7)
        other, _, _ = self.cluster("ring-3", "prod-c")
        self.assert_ok(self.run_playbook(ring="ring-2"))
        self.assertEqual(
            self.writes(),
            [
                {"operation": "put", "path": first, "cas": 4},
                {"operation": "put", "path": second, "cas": 7},
            ],
        )
        self.assertEqual(self.state["secrets"][other]["version"], 3)
        self.assertNotEqual(
            self.state["secrets"][first]["public"],
            self.state["secrets"][second]["public"],
        )
        journal = [
            json.loads(line)
            for line in (self.root / "rotation.jsonl").read_text().splitlines()
        ]
        self.assertEqual(
            [
                entry["previous_version"]
                for entry in journal
                if entry["status"] == "written"
            ],
            [4, 7],
        )
        for entry in journal:
            expected_fingerprint = hashlib.sha256(
                self.state["secrets"][entry["path"]]["public"].strip().encode()
            ).hexdigest()
            self.assertEqual(entry["public_key_sha256"], expected_fingerprint)
        self.assert_cleaned()

    def test_missing_secret_aborts_whole_ring_before_generating(self):
        self.cluster("ring-1", "stage-a")
        missing, _, _ = self.cluster("ring-1", "stage-b")
        del self.state["secrets"][missing]
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])

    def test_pinned_secret_is_rejected_before_vault_access(self):
        self.cluster("ring-1", "stage-a")
        self.state["pinned"] = True
        self.assertNotEqual(self.run_playbook(check=True).returncode, 0)
        self.assertFalse(self.state["events"])

    def test_failure_stops_remaining_clusters_and_cleans_keys(self):
        first, _, _ = self.cluster("ring-1", "stage-a")
        second, _, _ = self.cluster("ring-1", "stage-b")
        third, _, _ = self.cluster("ring-1", "stage-c")
        self.state["fail_write"] = second
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertEqual([event["path"] for event in self.writes()], [first, second])
        self.assertEqual(self.state["secrets"][third]["version"], 3)
        self.assert_cleaned()

    def test_cluster_subset_is_explicit_and_validated(self):
        first, _, _ = self.cluster("ring-1", "stage-a")
        self.cluster("ring-1", "stage-b")
        self.assert_ok(self.run_playbook(clusters=["stage-a"]))
        self.assertEqual([event["path"] for event in self.writes()], [first])
        self.state["events"] = []
        self.assertNotEqual(self.run_playbook(clusters=["not-in-ring"]).returncode, 0)
        self.assertFalse(self.state["events"])

    def test_migration_creates_canonical_key_then_updates_both_patches(self):
        source, canonical, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        _, _, other_overlay = self.cluster("ring-2", "prod-a", legacy=True)
        other_before = (other_overlay / "tekton-chains-public-key.yaml").read_text()
        self.assert_ok(self.run_playbook())
        self.assertEqual(
            self.writes(), [{"operation": "put", "path": canonical, "cas": 0}]
        )
        self.assertEqual(self.state["secrets"][source]["version"], 3)
        self.assertEqual(self.state["secrets"][canonical]["version"], 1)
        for patch in overlay.glob("tekton-chains-*.yaml"):
            self.assertEqual(yaml.safe_load(patch.read_text())[0]["value"], canonical)
        self.assertEqual(
            (other_overlay / "tekton-chains-public-key.yaml").read_text(), other_before
        )
        self.assert_cleaned()

    def test_existing_migration_destination_is_rotated_with_its_current_version(self):
        source, canonical, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        self.state["secrets"][canonical] = {"version": 7, "public": "PREPARED-PUBLIC"}
        self.assert_ok(self.run_playbook())
        self.assertEqual(
            self.writes(), [{"operation": "put", "path": canonical, "cas": 7}]
        )
        self.assertEqual(self.state["secrets"][source]["version"], 3)
        self.assertEqual(self.state["secrets"][canonical]["version"], 8)
        for patch in overlay.glob("tekton-chains-*.yaml"):
            self.assertEqual(yaml.safe_load(patch.read_text())[0]["value"], canonical)
        journal = [
            json.loads(line)
            for line in (self.root / "rotation.jsonl").read_text().splitlines()
        ]
        self.assertTrue(
            all(
                entry["previous_version"] == 7
                and entry["new_version"] == 8
                and entry["source_version"] == 3
                for entry in journal
            )
        )
        self.assert_cleaned()

    def test_existing_migration_destination_dry_run_makes_no_changes(self):
        source, canonical, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        self.state["secrets"][canonical] = {"version": 7, "public": "PREPARED-PUBLIC"}
        result = self.run_playbook(check=True)
        self.assert_ok(result)
        self.assertIn('"previous_version": 7', result.stdout)
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])
        self.assertEqual(self.state["secrets"][canonical]["version"], 7)
        for patch in overlay.glob("tekton-chains-*.yaml"):
            self.assertEqual(yaml.safe_load(patch.read_text())[0]["value"], source)

    def test_concurrent_destination_update_stops_migration_without_patching(self):
        source, canonical, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        self.state["secrets"][canonical] = {"version": 7, "public": "PREPARED-PUBLIC"}
        self.state["concurrent_write"] = canonical
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertEqual(
            self.writes(), [{"operation": "put", "path": canonical, "cas": 7}]
        )
        self.assertEqual(self.state["secrets"][source]["version"], 3)
        self.assertEqual(
            self.state["secrets"][canonical],
            {"version": 8, "public": "CONCURRENT-PUBLIC"},
        )
        for patch in overlay.glob("tekton-chains-*.yaml"):
            self.assertEqual(yaml.safe_load(patch.read_text())[0]["value"], source)
        self.assert_cleaned()

    def test_permission_denied_is_not_treated_as_an_absent_destination(self):
        _, canonical, _ = self.cluster("ring-1", "stage-a", legacy=True)
        self.state["deny_metadata"] = canonical
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())

    def test_generation_failure_cleans_up_without_modifying_manifests(self):
        source, _, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        self.state["fail_generation"] = True
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())
        self.assertEqual(
            yaml.safe_load((overlay / "tekton-chains-public-key.yaml").read_text())[0][
                "value"
            ],
            source,
        )
        self.assert_cleaned()

    def test_deleted_current_secret_is_rejected(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        self.state["secrets"][source]["deletion_time"] = "2026-11-16T00:00:00Z"
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())

    def test_mismatched_public_key_source_is_rejected(self):
        _, _, overlay = self.cluster("ring-1", "stage-a")
        patch = overlay / "tekton-chains-public-key.yaml"
        data = yaml.safe_load(patch.read_text())
        data[0]["value"] = (
            "staging/pipeline-service/another-cluster/chains-signing-secret"
        )
        patch.write_text(yaml.safe_dump(data))
        self.assertNotEqual(self.run_playbook(check=True).returncode, 0)
        self.assertFalse(self.state["events"])

    def test_explicit_one_version_retention_is_rejected(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        self.state["secrets"][source]["max_versions"] = 1
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())

    def test_retention_checks_use_only_secret_metadata(self):
        source, _, _ = self.cluster("ring-1", "stage-a")
        for per_key_limit in (0, 2):
            with self.subTest(per_key_limit=per_key_limit):
                self.state["secrets"][source]["max_versions"] = per_key_limit
                self.assert_ok(self.run_playbook(check=True))
                self.assertFalse(self.writes())

    def test_migration_rejects_a_patch_that_does_not_set_the_source(self):
        _, _, overlay = self.cluster("ring-1", "stage-a", legacy=True)
        patch = overlay / "tekton-chains-public-key.yaml"
        data = yaml.safe_load(patch.read_text())
        data[0]["path"] = "/metadata/annotations/legacy-path"
        patch.write_text(yaml.safe_dump(data))
        self.assertNotEqual(self.run_playbook(check=True).returncode, 0)
        self.assertFalse(self.state["events"])

    def test_shared_legacy_source_is_rejected_before_generating(self):
        shared, _, _ = self.cluster("ring-1", "stage-a", legacy=True)
        _, _, overlay = self.cluster("ring-1", "stage-b", legacy=True)
        for patch in overlay.glob("tekton-chains-*.yaml"):
            data = yaml.safe_load(patch.read_text())
            data[0]["value"] = shared
            patch.write_text(yaml.safe_dump(data))
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])

    def test_destination_used_by_another_cluster_is_rejected_before_generating(self):
        _, first_destination, _ = self.cluster("ring-1", "stage-a", legacy=True)
        _, _, second_overlay = self.cluster("ring-1", "stage-b", legacy=True)
        self.state["secrets"][first_destination] = {
            "version": 7,
            "public": "SECOND-CLUSTER-ACTIVE-PUBLIC",
        }
        for patch in second_overlay.glob("tekton-chains-*.yaml"):
            data = yaml.safe_load(patch.read_text())
            data[0]["value"] = first_destination
            patch.write_text(yaml.safe_dump(data))
        self.assertNotEqual(self.run_playbook().returncode, 0)
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])

    def test_default_journal_is_created_and_reported(self):
        self.cluster("ring-1", "stage-a")
        result = self.run_playbook(default_journal=True)
        self.assert_ok(result)
        journals = list(self.root.glob("chains-rotation-ring-1-*.jsonl"))
        self.assertEqual(len(journals), 1)
        self.assertIn(str(journals[0]), result.stdout)
        self.assertEqual(journals[0].stat().st_mode & 0o777, 0o600)

    @unittest.skipUnless(shutil.which("kustomize"), "requires kustomize")
    def test_real_ring_overlays_preflight_with_fake_vault(self):
        # Exercise real rendering and discovery, without allowing live Vault calls.
        repo_root = PLAYBOOK.resolve().parents[2]
        (self.bin_dir / "kustomize").unlink()
        rings = repo_root / "components/pipeline-service/rings"
        for ring in ("ring-1", "ring-2", "ring-3", "ring-4"):
            for overlay in sorted((rings / ring).glob("*/kustomization.yaml")):
                if overlay.parent.name == "base":
                    continue
                rendered = subprocess.run(
                    ["kustomize", "build", "--enable-helm", str(overlay.parent)],
                    capture_output=True,
                    text=True,
                    check=True,
                )
                for resource in yaml.safe_load_all(rendered.stdout):
                    if (
                        resource
                        and resource.get("kind") == "ExternalSecret"
                        and resource["metadata"]["name"]
                        == "tekton-chains-signing-secret"
                    ):
                        path = resource["spec"]["dataFrom"][0]["extract"]["key"]
                        self.state["secrets"][path] = {
                            "version": 3,
                            "public": "OLD-PUBLIC",
                        }
            self.assert_ok(
                self.run_playbook(ring=ring, check=True, repo_root=repo_root)
            )
        self.assertFalse(self.writes())
        self.assertFalse(self.state["temporary_dirs"])


if __name__ == "__main__":
    unittest.main()
