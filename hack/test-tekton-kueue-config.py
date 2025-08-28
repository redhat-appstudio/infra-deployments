#!/usr/bin/env python3
"""
Tekton-Kueue Configuration Test

A comprehensive test suite that validates the CEL expressions in the tekton-kueue configuration by:

1. **Reading configuration dynamically** from `components/kueue/development/tekton-kueue/config.yaml`
2. **Getting the image** from `components/kueue/staging/base/tekton-kueue/kustomization.yaml`
3. **Running mutations** using the actual tekton-kueue container via podman
4. **Validating results** against expected annotations, labels, and priority classes

Usage:
    # Check if all prerequisites are met
    python hack/test-tekton-kueue-config.py --check-setup

    # Run all tests
    python hack/test-tekton-kueue-config.py

    # Run tests with verbose output
    python hack/test-tekton-kueue-config.py --verbose

Test Scenarios:
    The test covers all CEL expressions in the configuration:

    1. **Multi-platform Resource Requests**:
       - New style: `build-platforms` parameter → `kueue.konflux-ci.dev/requests-*` annotations
       - Old style: `PLATFORM` parameters in tasks → `kueue.konflux-ci.dev/requests-*` annotations

    2. **AWS IP Resource Requests**:
       - New style: `build-platforms` parameter → `kueue.konflux-ci.dev/requests-aws-ip` annotations
         for platforms NOT in the excluded list (linux/ppc64le, linux/s390x, linux-x86-64, local, localhost, linux/amd64)
       - Old style: `PLATFORM` parameters in tasks → `kueue.konflux-ci.dev/requests-aws-ip` annotations
         for platforms NOT in the excluded list

    3. **Priority Assignment Logic**:
       - Push events → `konflux-post-merge-build`
       - Pull requests → `konflux-pre-merge-build`
       - Integration test push → `konflux-post-merge-test`
       - Integration test PR → `konflux-pre-merge-test`
       - Release managed → `konflux-release`
       - Release tenant → `konflux-tenant-release`
       - Mintmaker namespace → `konflux-dependency-update`
       - Default → `konflux-default`

    4. **Queue Assignment**: All PipelineRuns get `kueue.x-k8s.io/queue-name: pipelines-queue`

Prerequisites:
    - Python 3 with PyYAML
    - Podman (for running the tekton-kueue container)
    - Access to the tekton-kueue image specified in the kustomization

CI/CD Integration:
    The test runs automatically on pull requests via the GitHub action
    `.github/workflows/test-tekton-kueue-config.yaml` when:
    - Changes are made to `components/kueue/**`
    - The test script itself is modified
    - The workflow file is modified

    The test will **FAIL** (not skip) if any prerequisites are missing, ensuring
    issues are caught early in CI/CD pipelines.
"""

import subprocess
import tempfile
import os
import yaml
import unittest
from pathlib import Path
from typing import Dict
from dataclasses import dataclass
import sys


@dataclass
class Prerequisites:
    image: str
    podman_version: str
    config_file: Path
    kustomization_file: Path


def get_tekton_kueue_image(kustomization_file: Path) -> str:
    """Read the tekton-kueue image from the given kustomization file."""
    try:
        with open(kustomization_file, 'r') as f:
            kustomization = yaml.safe_load(f)

        # Look for the tekton-kueue image in the images section
        images = kustomization.get('images', [])
        for image in images:
            if image.get('name') == 'konflux-ci/tekton-kueue':
                new_name = image.get('newName', '')
                new_tag = image.get('newTag', '')
                if new_name and new_tag:
                    return f"{new_name}:{new_tag}"

        raise ValueError("tekton-kueue image not found in kustomization")

    except Exception as e:
        raise RuntimeError(f"Failed to read tekton-kueue image from {kustomization_file}: {e}")

def check_prerequisites(should_print: bool = True) -> Prerequisites:
    """Check that all prerequisites are available.

    Returns a Prerequisites object with discovered info (image, podman_version)
    on success. Raises an exception on failure.
    """
    messages = ["Checking prerequisites..."]

    # Compute repo paths locally
    repo_root = Path(__file__).parent.parent
    config_file = repo_root / "components/kueue/development/tekton-kueue/config.yaml"
    kustomization_file = repo_root / "components/kueue/development/tekton-kueue/kustomization.yaml"

    # Config file
    if not config_file.exists():
        raise FileNotFoundError(f"Config file not found: {config_file}")
    messages.append(f"\u2713 Config file found: {config_file}")

    # Kustomization file
    if not kustomization_file.exists():
        raise FileNotFoundError(f"Kustomization file not found: {kustomization_file}")
    messages.append(f"\u2713 Kustomization file found: {kustomization_file}")

    # Image from kustomization
    image = get_tekton_kueue_image(kustomization_file)
    messages.append(f"\u2713 Tekton-kueue image: {image}")

    # Podman availability
    result = subprocess.run(["podman", "--version"], capture_output=True, check=True, text=True)
    podman_version = result.stdout.strip()
    messages.append(f"\u2713 Podman available: {podman_version}")

    if should_print:
        for line in messages:
            print(line)

    return Prerequisites(
        image=image,
        podman_version=podman_version,
        config_file=config_file,
        kustomization_file=kustomization_file,
    )

# Test PipelineRun definitions
TEST_PIPELINERUNS = {
    "multiplatform_new": {
        "name": "Multi-platform pipeline (new style with build-platforms parameter)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-multiplatform-new",
                "namespace": "default",
                "labels": {
                    "pipelinesascode.tekton.dev/event-type": "push"
                }
            },
            "spec": {
                "pipelineRef": {"name": "build-pipeline"},
                "params": [
                    {
                        "name": "build-platforms",
                        "value": ["linux/amd64", "linux/arm64", "linux/s390x"]
                    },
                    {"name": "other-param", "value": "test"}
                ],
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {
                "kueue.konflux-ci.dev/requests-linux-amd64": "1",
                "kueue.konflux-ci.dev/requests-linux-arm64": "1",
                "kueue.konflux-ci.dev/requests-linux-s390x": "1",
                "kueue.konflux-ci.dev/requests-aws-ip": "2"
            },
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-post-merge-build"
            }
        }
    },

    "multiplatform_old": {
        "name": "Multi-platform pipeline (old style with PLATFORM parameters)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-multiplatform-old",
                "namespace": "default",
                "labels": {
                    "pipelinesascode.tekton.dev/event-type": "pull_request"
                }
            },
            "spec": {
                "pipelineSpec": {
                    "tasks": [
                        {
                            "name": "build-task-amd64",
                            "params": [{"name": "PLATFORM", "value": "linux/amd64"}],
                            "taskRef": {"name": "build-task"}
                        },
                        {
                            "name": "build-task-arm64",
                            "params": [{"name": "PLATFORM", "value": "linux/arm64"}],
                            "taskRef": {"name": "build-task"}
                        },
                        {
                            "name": "other-task",
                            "taskRef": {"name": "other-task"}
                        }
                    ]
                },
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {
                "kueue.konflux-ci.dev/requests-linux-amd64": "1",
                "kueue.konflux-ci.dev/requests-linux-arm64": "1",
                "kueue.konflux-ci.dev/requests-aws-ip": "2"
            },
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-pre-merge-build"
            }
        }
    },

    "release_managed": {
        "name": "Release managed pipeline",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-release-managed",
                "namespace": "default",
                "labels": {
                    "appstudio.openshift.io/service": "release",
                    "pipelines.appstudio.openshift.io/type": "managed"
                }
            },
            "spec": {
                "pipelineRef": {"name": "release-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-release"
            }
        }
    },

    "release_tenant": {
        "name": "Release tenant pipeline",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-release-tenant",
                "namespace": "default",
                "labels": {
                    "appstudio.openshift.io/service": "release",
                    "pipelines.appstudio.openshift.io/type": "tenant"
                }
            },
            "spec": {
                "pipelineRef": {"name": "release-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-tenant-release"
            }
        }
    },

    "mintmaker": {
        "name": "Mintmaker dependency update",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-mintmaker",
                "namespace": "mintmaker"
            },
            "spec": {
                "pipelineRef": {"name": "dependency-update-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-dependency-update"
            }
        }
    },

    "integration_test_push": {
        "name": "Integration test pipeline (push event)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-integration-test-push",
                "namespace": "default",
                "labels": {
                    "pac.test.appstudio.openshift.io/event-type": "push"
                }
            },
            "spec": {
                "pipelineRef": {"name": "integration-test-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-post-merge-test"
            }
        }
    },

    "integration_test_pr": {
        "name": "Integration test pipeline (pull request event)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-integration-test-pr",
                "namespace": "default",
                "labels": {
                    "pac.test.appstudio.openshift.io/event-type": "pull_request"
                }
            },
            "spec": {
                "pipelineRef": {"name": "integration-test-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-pre-merge-test"
            }
        }
    },

    "default_priority": {
        "name": "Default pipeline (no special labels)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-default",
                "namespace": "default"
            },
            "spec": {
                "pipelineRef": {"name": "default-pipeline"},
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-default"
            }
        }
    },

    "aws_platforms_only": {
        "name": "Multi-platform pipeline with AWS platforms only (new style)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-aws-platforms-only",
                "namespace": "default",
                "labels": {
                    "pipelinesascode.tekton.dev/event-type": "push"
                }
            },
            "spec": {
                "pipelineRef": {"name": "build-pipeline"},
                "params": [
                    {
                        "name": "build-platforms",
                        "value": ["linux/arm64", "darwin/amd64", "windows/amd64"]
                    },
                    {"name": "other-param", "value": "test"}
                ],
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {
                "kueue.konflux-ci.dev/requests-linux-arm64": "1",
                "kueue.konflux-ci.dev/requests-darwin-amd64": "1",
                "kueue.konflux-ci.dev/requests-windows-amd64": "1",
                "kueue.konflux-ci.dev/requests-aws-ip": "3"  # All 3 platforms request aws-ip (none in excluded list)
            },
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-post-merge-build"
            }
        }
    },

    "mixed_platforms_excluded_included": {
        "name": "Multi-platform pipeline with mix of excluded and AWS platforms (new style)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-mixed-platforms",
                "namespace": "default",
                "labels": {
                    "pipelinesascode.tekton.dev/event-type": "push"
                }
            },
            "spec": {
                "pipelineRef": {"name": "build-pipeline"},
                "params": [
                    {
                        "name": "build-platforms",
                        "value": ["linux/amd64", "linux/s390x", "linux/ppc64le", "linux/arm64", "darwin/amd64"]
                    }
                ],
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {
                "kueue.konflux-ci.dev/requests-linux-amd64": "1",
                "kueue.konflux-ci.dev/requests-linux-s390x": "1",
                "kueue.konflux-ci.dev/requests-linux-ppc64le": "1",
                "kueue.konflux-ci.dev/requests-linux-arm64": "1",
                "kueue.konflux-ci.dev/requests-darwin-amd64": "1",
                "kueue.konflux-ci.dev/requests-aws-ip": "3"
            },
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-post-merge-build"
            }
        }
    }
}


class TektonKueueMutationTest(unittest.TestCase):
    """Test suite for tekton-kueue CEL expression mutations."""

    @classmethod
    def setUpClass(cls):
        """Set up test class - check prerequisites."""
        info = check_prerequisites(should_print=False)
        cls.tekton_kueue_image = info.image
        cls.config_file = info.config_file
        print(f"Using tekton-kueue image: {cls.tekton_kueue_image}")

    def run_mutation_test(self, test_data: Dict) -> Dict:
        """Run a single mutation test and return results."""
        pipelinerun = test_data["pipelinerun"]

        with tempfile.TemporaryDirectory() as temp_dir:
            # Write the config file
            config_path = Path(temp_dir) / "config.yaml"
            pipelinerun_path = Path(temp_dir) / "pipelinerun.yaml"

            # Copy the config file
            import shutil
            shutil.copy2(self.config_file, config_path)

            # Write the PipelineRun
            with open(pipelinerun_path, 'w') as f:
                yaml.dump(pipelinerun, f, default_flow_style=False)

            # Set proper permissions
            os.chmod(config_path, 0o644)
            os.chmod(pipelinerun_path, 0o644)
            os.chmod(temp_dir, 0o755)

            # Run the mutation
            cmd = [
                "podman", "run", "--rm",
                "-v", f"{temp_dir}:/workspace:z",
                self.tekton_kueue_image,
                "mutate",
                "--pipelinerun-file", "/workspace/pipelinerun.yaml",
                "--config-dir", "/workspace"
            ]

            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                self.fail(f"Mutation failed: {result.stderr}")

            # Parse the mutated PipelineRun
            try:
                mutated = yaml.safe_load(result.stdout)
            except yaml.YAMLError as e:
                self.fail(f"Failed to parse mutated YAML: {e}")

            return mutated

    def validate_mutation_result(self, test_key: str, test_data: Dict):
        """Helper method to validate mutation results."""
        with self.subTest(test=test_key):
            mutated = self.run_mutation_test(test_data)
            expected = test_data["expected"]

            original_metadata = test_data["pipelinerun"].get("metadata", {})
            original_annotations = original_metadata.get("annotations", {}) or {}
            original_labels = original_metadata.get("labels", {}) or {}

            # Check annotations (full equality vs original + expected)
            annotations = mutated.get("metadata", {}).get("annotations", {})
            expected_annotations = expected["annotations"]
            expected_annotations_full = {**original_annotations, **expected_annotations}
            self.assertDictEqual(
                annotations,
                expected_annotations_full,
                f"Annotations mismatch; expected {expected_annotations_full}, got {annotations}"
            )

            # Check labels (full equality vs original + expected)
            labels = mutated.get("metadata", {}).get("labels", {})
            expected_labels = expected["labels"]
            expected_labels_full = {**original_labels, **expected_labels}
            self.assertDictEqual(
                labels,
                expected_labels_full,
                f"Labels mismatch; expected {expected_labels_full}, got {labels}"
            )

    def test_all_mutations(self):
        """Test all tekton-kueue mutation scenarios."""
        for test_key, test_data in TEST_PIPELINERUNS.items():
            self.validate_mutation_result(test_key, test_data)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Test tekton-kueue CEL expressions")
    parser.add_argument("--check-setup", action="store_true",
                       help="Check if prerequisites are met and show configuration")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Run tests with verbose output")

    # Parse known args to allow unittest args to pass through
    args, unknown = parser.parse_known_args()

    if args.check_setup:
        try:
            info = check_prerequisites(should_print=True)
        except Exception as e:
            print(f"✗ {e}")
            sys.exit(1)

        print("\n✅ All prerequisites met! Ready to run tests.")
        print("Run: python hack/test-tekton-kueue-config.py")
        print("\nNote: Tests will FAIL (not skip) if any prerequisites are missing.")

    else:
        # Run unittest with remaining args
        verbosity = 2 if args.verbose else 1
        sys.argv = [sys.argv[0]] + unknown
        unittest.main(verbosity=verbosity)
