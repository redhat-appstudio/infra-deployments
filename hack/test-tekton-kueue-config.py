#!/usr/bin/env python3
"""
Tekton-Kueue Configuration Test

A comprehensive test suite that validates the CEL expressions in the tekton-kueue configuration by:

1. **Reading configuration dynamically** from specified config files
2. **Getting the image** from specified kustomization files
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
    Each test can now specify its own config file and kustomization file,
    allowing testing of multiple configurations and images.

Prerequisites:
    - Python 3 with PyYAML
    - Podman (for running the tekton-kueue container)
    - Access to the tekton-kueue images specified in the kustomizations

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
from typing import Dict, Any, TypedDict
from dataclasses import dataclass
import sys


@dataclass
class TestConfig:
    config_file: Path
    kustomization_file: Path
    image: str


class ConfigCombination(TypedDict):
    config_file: str
    kustomization_file: str


class TestCombination(TypedDict):
    pipelinerun_key: str
    config_key: str


class PipelineRunMetadata(TypedDict, total=False):
    name: str
    namespace: str
    labels: Dict[str, str]
    annotations: Dict[str, str]


class PipelineRunDefinition(TypedDict):
    apiVersion: str
    kind: str
    metadata: PipelineRunMetadata
    spec: Dict[str, Any]  # More flexible since PipelineRun specs can vary


class ExpectedResults(TypedDict):
    annotations: Dict[str, str]
    labels: Dict[str, str]


class PipelineRunTestData(TypedDict):
    pipelinerun: PipelineRunDefinition
    expected: ExpectedResults


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

def resolve_path(path_str: str, repo_root: Path) -> Path:
    """Resolve a path string to an absolute Path, handling both relative and absolute paths."""
    if Path(path_str).is_absolute():
        return Path(path_str)
    return repo_root / path_str


def validate_config_combination(config_key: str, repo_root: Path) -> TestConfig:
    """Validate and resolve config and kustomization files for a config combination."""
    config_data = CONFIG_COMBINATIONS[config_key]

    config_file = resolve_path(config_data["config_file"], repo_root)
    kustomization_file = resolve_path(config_data["kustomization_file"], repo_root)

    # Validate files exist
    if not config_file.exists():
        raise FileNotFoundError(f"Config file not found for config '{config_key}': {config_file}")

    if not kustomization_file.exists():
        raise FileNotFoundError(f"Kustomization file not found for config '{config_key}': {kustomization_file}")

    # Get image from kustomization
    image = get_tekton_kueue_image(kustomization_file)

    return TestConfig(
        config_file=config_file,
        kustomization_file=kustomization_file,
        image=image
    )


def check_prerequisites(should_print: bool = True) -> Dict[str, TestConfig]:
    """Check that all prerequisites are available and pre-process config combinations."""
    messages = ["Checking prerequisites..."]
    repo_root = Path(__file__).parent.parent

    # Check podman availability
    try:
        result = subprocess.run(["podman", "--version"], capture_output=True, check=True, text=True)
        podman_version = result.stdout.strip()
        messages.append(f"✓ Podman available: {podman_version}")
    except (subprocess.CalledProcessError, FileNotFoundError):
        raise RuntimeError("Podman not available")

    # Pre-process all unique config combinations
    processed_configs: Dict[str, TestConfig] = {}

    for _, test_combination in TEST_COMBINATIONS.items():
        config_key = test_combination["config_key"]

        # Only process each config combination once
        if config_key not in processed_configs:
            try:
                config = validate_config_combination(config_key, repo_root)
                processed_configs[config_key] = config
                messages.append(f"✓ Config '{config_key}': {config.config_file}, image={config.image}")
            except Exception as e:
                raise RuntimeError(f"Config '{config_key}' validation failed: {e}")

    if should_print:
        for message in messages:
            print(message)

    return processed_configs

# Test PipelineRun definitions (reusable across different configs)
PIPELINERUN_DEFINITIONS: Dict[str, PipelineRunTestData] = {
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
    "multiplatform_old_no_pipelineSpecTasks": {
        "name": "Multi-platform pipeline (old style with PLATFORM parameters): no tasks in pipelineSpec",
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
                    "tasks": None,
                },
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-pre-merge-build",
                "pipelinesascode.tekton.dev/event-type": "pull_request"
            }
        }
    },
    "multiplatform_old_empty_pipelineSpecTasks": {
        "name": "Multi-platform pipeline (old style with PLATFORM parameters): empty tasks in pipelineSpec",
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
                    "tasks": [],
                },
                "workspaces": [{"name": "shared-workspace", "emptyDir": {}}]
            }
        },
        "expected": {
            "annotations": {},
            "labels": {
                "kueue.x-k8s.io/queue-name": "pipelines-queue",
                "kueue.x-k8s.io/priority-class": "konflux-pre-merge-build",
                "pipelinesascode.tekton.dev/event-type": "pull_request"
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
            "annotations": {
                "kueue.konflux-ci.dev/requests-mintmaker": "1",
            },
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
    },

    "user-specific-priority": {
        "name": "Multi-platform pipeline with user-specific priority (new style)",
        "pipelinerun": {
            "apiVersion": "tekton.dev/v1",
            "kind": "PipelineRun",
            "metadata": {
                "name": "test-user-specific-priority",
                "namespace": "default",
                "labels": {
                    "pipelinesascode.tekton.dev/event-type": "push",
                    "kueue.x-k8s.io/priority-class": "konflux-user-specific"
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
                "kueue.x-k8s.io/priority-class": "konflux-user-specific"
            }
        }
    },

}

# Configuration combinations that can be applied to any PipelineRun
CONFIG_COMBINATIONS: Dict[str, ConfigCombination] = {
    "development": {
        "name": "Development config",
        "config_file": "components/kueue/development/tekton-kueue/config.yaml",
        "kustomization_file": "components/kueue/development/tekton-kueue/kustomization.yaml"
    },
    "staging": {
        "name": "Staging config",
        "config_file": "components/kueue/staging/base/tekton-kueue/config.yaml",
        "kustomization_file": "components/kueue/staging/base/tekton-kueue/kustomization.yaml"
    },
    "production": {
        "name": "Production config",
        "config_file": "components/kueue/production/base/tekton-kueue/config.yaml",
        "kustomization_file": "components/kueue/production/base/tekton-kueue/kustomization.yaml"
    },
    "production-kflux-ocp-p01": {
        "name": "Production config",
        "config_file": "components/kueue/production/kflux-ocp-p01/config.yaml",
        "kustomization_file": "components/kueue/production/base/tekton-kueue/kustomization.yaml"
    }
}

# Test combinations: which PipelineRuns to test with which configs
# This creates a cartesian product of PipelineRuns and configs
TEST_COMBINATIONS: Dict[str, TestCombination] = {
    # Test all PipelineRuns with development config (default)
    "multiplatform_new_dev": {
        "pipelinerun_key": "multiplatform_new",
        "config_key": "development"
    },
    "multiplatform_old_dev": {
        "pipelinerun_key": "multiplatform_old",
        "config_key": "development"
    },
    "release_managed_dev": {
        "pipelinerun_key": "release_managed",
        "config_key": "development"
    },
    "release_tenant_dev": {
        "pipelinerun_key": "release_tenant",
        "config_key": "development"
    },
    "mintmaker_dev": {
        "pipelinerun_key": "mintmaker",
        "config_key": "development"
    },
    "integration_test_push_dev": {
        "pipelinerun_key": "integration_test_push",
        "config_key": "development"
    },
    "integration_test_pr_dev": {
        "pipelinerun_key": "integration_test_pr",
        "config_key": "development"
    },
    "default_priority_dev": {
        "pipelinerun_key": "default_priority",
        "config_key": "development"
    },
    "aws_platforms_only_dev": {
        "pipelinerun_key": "aws_platforms_only",
        "config_key": "development"
    },
    "mixed_platforms_excluded_included_dev": {
        "pipelinerun_key": "mixed_platforms_excluded_included",
        "config_key": "development"
    },

    # multiplatform_old edge cases
    "multiplatform_old_no_pipelineSpecTasks": {
        "pipelinerun_key": "multiplatform_old_no_pipelineSpecTasks",
        "config_key": "development"
    },
    "multiplatform_old_empty_pipelineSpecTasks": {
        "pipelinerun_key": "multiplatform_old_empty_pipelineSpecTasks",
        "config_key": "development"
    },

    # Test key PipelineRuns with staging config
    "multiplatform_new_staging": {
        "pipelinerun_key": "multiplatform_new",
        "config_key": "staging"
    },
    "release_managed_staging": {
        "pipelinerun_key": "release_managed",
        "config_key": "staging"
    },
    "integration_test_push_staging": {
        "pipelinerun_key": "integration_test_push",
        "config_key": "staging"
    },
    "mintmaker_staging": {
        "pipelinerun_key": "mintmaker",
        "config_key": "staging"
    },

    # Test key PipelineRuns with production config
    "multiplatform_new_production": {
        "pipelinerun_key": "multiplatform_new",
        "config_key": "production"
    },
    "release_managed_production": {
        "pipelinerun_key": "release_managed",
        "config_key": "production"
    },

    # Example: Test the same PipelineRun with different configs to show reusability
    "user-specific-priority_and_mixed_platforms_production-kflux-ocp-p01": {
        "pipelinerun_key": "user-specific-priority",
        "config_key": "production-kflux-ocp-p01"
    }
}


class TektonKueueMutationTest(unittest.TestCase):
    """Test suite for tekton-kueue CEL expression mutations."""

    @classmethod
    def setUpClass(cls):
        """Set up test class - check prerequisites and pre-process configs."""
        cls.processed_configs = check_prerequisites(should_print=False)
        cls.repo_root = Path(__file__).parent.parent
        print("Prerequisites validated for all tests.")

    def run_mutation_test(self, test_combination: TestCombination) -> Dict[str, Any]:
        """Run a single mutation test and return results."""
        # Get pre-processed configuration
        config_key = test_combination["config_key"]
        test_config = self.processed_configs[config_key]

        # Get the PipelineRun definition
        pipelinerun_key = test_combination["pipelinerun_key"]
        pipelinerun_data = PIPELINERUN_DEFINITIONS[pipelinerun_key]
        pipelinerun = pipelinerun_data["pipelinerun"]

        with tempfile.TemporaryDirectory() as temp_dir:
            # Write the config file
            config_path = Path(temp_dir) / "config.yaml"
            pipelinerun_path = Path(temp_dir) / "pipelinerun.yaml"

            # Copy the test-specific config file
            import shutil
            shutil.copy2(test_config.config_file, config_path)

            # Write the PipelineRun
            with open(pipelinerun_path, 'w') as f:
                yaml.dump(pipelinerun, f, default_flow_style=False)

            # Set proper permissions
            os.chmod(config_path, 0o644)
            os.chmod(pipelinerun_path, 0o644)
            os.chmod(temp_dir, 0o755)

            # Run the mutation with test-specific image
            cmd = [
                "podman", "run", "--rm",
                "-v", f"{temp_dir}:/workspace:z",
                test_config.image,
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

    def validate_mutation_result(self, test_key: str, test_combination: TestCombination) -> None:
        """Helper method to validate mutation results."""
        with self.subTest(test=test_key):
            # Get pre-processed configuration for logging
            config_key = test_combination["config_key"]
            test_config = self.processed_configs[config_key]
            print(f"Running test '{test_key}' with config: {test_config.config_file}, image: {test_config.image}")

            mutated = self.run_mutation_test(test_combination)

            # Get expected results from the PipelineRun definition
            pipelinerun_key = test_combination["pipelinerun_key"]
            pipelinerun_data = PIPELINERUN_DEFINITIONS[pipelinerun_key]
            expected = pipelinerun_data["expected"]

            original_metadata = pipelinerun_data["pipelinerun"].get("metadata", {})
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
        for test_key, test_combination in TEST_COMBINATIONS.items():
            self.validate_mutation_result(test_key, test_combination)


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
            processed_configs = check_prerequisites(should_print=True)
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
