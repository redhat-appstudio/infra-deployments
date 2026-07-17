# Component Promotion

> **Warning**
> Ring deployments are still under active development. This documentation may change as the implementation evolves.

## Introduction

This document serves as a guide on how to utilize ring-deployments to promote changes to Rings 0 & 1 and how to manually promote changes to later rings.

## Pre-requisites

* The target component follows the standard component structure outlined in the [Directory Layout](./directory-layout.md) document
* The target component has been onboarded to Kargo

## Image-based Promotions

Image-based promotions include things like updating image tag(s)/digest(s) in Kustomize files and updating external repository references in Kustomize files. The following sections detail steps for promoting these kinds of changes. Step 1 & 2 should be automated with Kargo, with step 1 being completely optional.

### Step 1: Promotion to Ring 0 (development) *--Optional*

*If your component needs to be deployed to a development environment*, the component should be configured in Kargo to auto-promote changes to its Ring 0 overlay. Kargo should create and auto-merge a PR for this promotion.

### Step 2: Promotion to Ring 1 (staging)

The component should be configured, in Kargo, to auto-promote changes to its Ring 1 overlay. Kargo should create and auto-merge a PR for this promotion.

### Step 3: Promotion to Ring <N> (production)

This process should be repeated for all rings after ring 1. Kargo currently promotes to Ring 0 and Ring 1 clusters, so this step is manual.

1. Fork and clone the infra-deployments repository, if needed, and create a new branch.
2. Take any changes made to the `components/<COMPONENT_NAME>/rings/ring-<N-1>/base/kustomization.yaml` file in the PR created for the N-1 ring and apply them to the  `components/<COMPONENT_NAME>/rings/ring-<N>/base/kustomization.yaml` file.
3. Save, commit, and push those changes.
4. Create a PR titled 'chore(ring-<N>): promote <COMPONENT_NAME>' with the following description:
    ```markdown
    ### Ring <N>

    | | |
    |---|---|
    | **Component** | `<COMPONENT_NAME>` |
    | **Project** | `kargo-infra-deployments` |
    | **Promotion** | `<LINK_TO_PREV_RING_PR>` |

    ---

    Promotes `<COMPONENT_NAME>` to Ring <N>

    ---
    ```

*Be sure to replace all templated fields (surrounded with <>)*
5. Wait for the PR to be approved by the code owners and merged.
6. Let the change soak for at least 24 hours and perform any tests needed.

## Manifest-based Promotions

Manifest-based promotions include any changes that need to be made to the resources defined in the `components/<COMPONENT_NAME>/base/` folder. The following sections detail steps for promoting these kinds of changes. Steps 2 & 3 should be automated with Kargo, with Step 2 being optional.


### Step 1: Create the Manifest Changes

1. Fork and clone the infra-deployments repository, if needed, and create a new branch.
2. Make any manifest changes (additions, deletions, modifications) to the resources in the `components/<COMPONENT_NAME>/base` folder.
3. Save, commit, and push those changes.
4. Create a PR called 'chore: update <COMPONENT_NAME> manifests' and describe the changes made.
5. Wait for the PR to be approved by the code owners and merged.

### Step 2: Promotion to Ring 0 (development) *--Optional*

*If your component needs to be deployed to a development environment*, the component should be configured in Kargo to auto-promote changes to its Ring 0 overlay. Kargo should create and auto-merge a PR for this promotion.

### Step 3: Promotion to Ring 1 (staging)

The component should be configured, in Kargo, to auto-promote changes to its Ring 1 overlay. Kargo should create and auto-merge a PR for this promotion.

### Step 4: Promotion to Ring <N> (production)

This step can be applied to all rings after Ring 1. Kargo currently promotes to Ring 0 and Ring 1 clusters, so this step is manual.

1. Fork and clone the infra-deployments repository, if needed, and create a new branch.
2. Copy the `components/<COMPONENT_NAME>/rings/ring-<N-1>/base/tier-1-ref/` folder into the `components/<COMPONENT_NAME>/rings/ring-<N>/base/` folder. *This should overwrite an existig folder.*
3. Save, commit, and push those changes.
4. Create a PR titled 'chore(ring-<N>): promote <COMPONENT_NAME>' with the following description:
    ```markdown
    ### Ring <N>

    | | |
    |---|---|
    | **Component** | `<COMPONENT_NAME>` |
    | **Project** | `kargo-infra-deployments` |
    | **Promotion** | `<ORIG_PR_LINK>` |

    ---

    Promotes `<COMPONENT_NAME>` to Ring <N>

    ---
    ```
6. Wait for the PR to be approved by the code owners and merged.
7. Let the change soak for at least 24 hours and perform any tests needed.
