---
title: About Visual Learning Guides
---

# About These Visual Guides

Interactive visual documentation to help understand the Konflux CI/CD platform and infra-deployments repository.

**→ [Browse All Visual Guides](index.html)**

## What Are These Guides?

Self-contained interactive HTML pages with:
- 📊 Animated diagrams and flowcharts
- 🔍 Searchable component directories  
- 📱 Mobile-responsive design
- 🎨 Color-coded categories and difficulty levels

They complement the text documentation by providing visual explanations of complex workflows.

### How Were These Guides Created?

These guides were generated using **AI assistance in a Cursor workspace** that had multiple Konflux-related repositories cloned together:

- `redhat-appstudio/infra-deployments` - The main deployment manifests
- `konflux-ci/e2e-tests` - End-to-end test suites and rules engine
- `openshift/release` - OpenShift CI configuration and job definitions
- `konflux-ci/architecture` - System architecture documentation
- Various component repositories (build-service, integration-service, etc.)

**Why does this matter?** Many workflows involving infra-deployments span multiple repositories. For example:
- The test selection rules that run on infra-deployments PRs are defined in `e2e-tests`
- The CI job configuration lives in `openshift/release`
- Component behavior that drives manifest changes is in the component repos

Having all these repos in the same workspace allowed the AI to research across the entire ecosystem and accurately describe how things work end-to-end, not just what's visible in infra-deployments alone.

**To update or create guides with full context**, clone the related repos into your workspace before asking your LLM to make changes.

---

## 🤖 Contributing with AI Assistance

These guides are designed to be **maintained and extended with LLM assistance** (Cursor, Copilot, Claude, etc.).

### Updating Existing Guides

When you make changes to infra-deployments that might affect a visual guide:

**Tell your LLM:**
> "I just made changes to [describe your changes]. Check if any visual learning guides in `docs/misc/visual-learning-guides/` need to be updated. Reference the style guide in `docs/misc/visual-learning-guides/style-guide-for-llms.md` and the existing HTML files in that directory to maintain consistency."

**What changes trigger updates:**

| Your Change | Guide to Update |
|-------------|-----------------|
| New component in `components/` | `visual-components-map.html` |
| New Kustomize overlay | `visual-kustomize-overlays.html` |
| CI/testing config changes | `visual-testing-flow.html`, `visual-e2e-infra-tests.html` |
| PR pairing syntax changes | `visual-pr-pairing.html` |
| Renovate/MintMaker changes | `visual-renovate-workflow.html` |
| Deployment pipeline/ArgoCD changes | `visual-pr-workflow.html` |

**Current guides:**
- `index.html` - Landing page with cards linking to all guides
- `visual-components-map.html` - Searchable component directory
- `visual-kustomize-overlays.html` - How Kustomize overlays work
- `visual-testing-flow.html` - Complete E2E testing flow
- `visual-e2e-infra-tests.html` - Test selection rules for infra-deployments PRs
- `visual-pr-pairing.html` - How to pair PRs across repos (Prow-specific)
- `visual-pr-workflow.html` - Animated PR-to-production timeline
- `visual-renovate-workflow.html` - MintMaker/Renovate automation

### Creating New Guides

To add a new visual learning guide:

**Tell your LLM:**
> "Create a new visual learning guide about [your topic]. Read `docs/misc/visual-learning-guides/style-guide-for-llms.md` for the style guide and color palette. Use the existing HTML files in that directory as reference for structure and patterns. Research the relevant code in this repository, then generate a new HTML file following the same patterns."

The LLM should:
1. Read `docs/misc/visual-learning-guides/style-guide-for-llms.md` for the style guide
2. Read 2-3 existing HTML files in that directory to understand patterns
3. Research the topic in the codebase
4. Generate a new HTML file with consistent styling
5. Update `docs/misc/visual-learning-guides/index.html` to add a card for the new guide

---

## 📋 Style Guide (For LLMs)

For detailed technical reference on colors, typography, code blocks, and patterns to use when generating or updating guides, see:

**→ [style-guide-for-llms.md](style-guide-for-llms.md)**

