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
> "I just made changes to [describe your changes]. Check if any visual learning guides in `docs/misc/visual-learning-guides/` need to be updated. Reference the style guide in `docs/misc/visual-learning-guides/README.md` and the existing HTML files in that directory to maintain consistency."

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
> "Create a new visual learning guide about [your topic]. Read `docs/misc/visual-learning-guides/README.md` for the style guide and color palette. Use the existing HTML files in that directory as reference for structure and patterns. Research the relevant code in this repository, then generate a new HTML file following the same patterns."

The LLM should:
1. Read `docs/misc/visual-learning-guides/README.md` for the style guide
2. Read 2-3 existing HTML files in that directory to understand patterns
3. Research the topic in the codebase
4. Generate a new HTML file with consistent styling
5. Update `docs/misc/visual-learning-guides/index.html` to add a card for the new guide

---

## 📋 Style Guide (For LLMs)

**This section is technical reference for AI assistants generating or updating guides.**

### File Structure

Each guide is a standalone HTML file with:
- Full `<!DOCTYPE html>` document
- Embedded CSS in `<style>` tags (no external stylesheets)
- Embedded JavaScript in `<script>` tags (no external scripts)
- No external dependencies

### Color Palette

```css
/* Primary gradient (headers, accents) */
background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

/* Card backgrounds */
background: white;
box-shadow: 0 4px 6px rgba(0,0,0,0.1);

/* Difficulty tags */
.beginner { background: #48bb78; }      /* Green */
.intermediate { background: #ed8936; }  /* Orange */
.advanced { background: #f56565; }      /* Red */

/* Category colors */
Success/Green: #48bb78, #10b981
Warning/Orange: #ed8936, #f59e0b
Info/Blue: #4299e1, #3b82f6
Purple accent: #9f7aea, #764ba2
Error/Red: #ef4444, #f56565
```

### Required Elements

Every guide should include:

1. **Back link** at the top:
```html
<a href="index.html" class="back-link">← Back to Visual Guide Index</a>
```

2. **Header** with gradient background:
```html
<div class="header">
    <h1>🎨 Guide Title</h1>
    <p>Brief description</p>
</div>
```

3. **Difficulty tags** on interactive elements:
```html
<span class="difficulty beginner">Beginner</span>
<span class="difficulty intermediate">Intermediate</span>
```

4. **Card-based layout** for navigation/content:
```html
<div class="card">
    <div class="card-icon">🧩</div>
    <h2>Section Title</h2>
    <p>Description text</p>
</div>
```

### Typography

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
```

### Code Blocks

**IMPORTANT:** Always include `white-space: pre-wrap` to preserve line breaks:

```css
.code-block {
    background: #2d3748;
    color: #68d391;
    padding: 20px;
    border-radius: 8px;
    font-family: 'Courier New', monospace;
    font-size: 0.9em;
    line-height: 1.6;
    white-space: pre-wrap;  /* Required for line breaks! */
    overflow-x: auto;
}
```

### File Paths & Directory Structures

**Don't use code blocks for file trees** - they render poorly. Instead, use **card layouts**:

```html
<!-- Good: Card layout for file locations -->
<div style="background: #f0fff4; border-radius: 10px; padding: 20px; border-left: 4px solid #48bb78;">
    <h3 style="color: #2f855a;">📁 Directory Name</h3>
    <code style="background: #2d3748; color: #68d391; padding: 8px 12px; border-radius: 5px;">
        path/to/file.yaml
    </code>
    <span style="color: #48bb78; font-weight: bold;">← Description</span>
</div>
```

Use **colored boxes** with:
- Blue (#4299e1) for base/shared resources
- Green (#48bb78) for development  
- Orange (#ed8936) for staging
- Red (#f56565) for production

### Responsive Design

All guides should work on mobile. Use:
```css
@media (max-width: 768px) {
    /* Mobile adjustments */
}
```

### Index Card Template

When adding a new guide, add this to `index.html`:
```html
<a href="visual-NEW-GUIDE.html" class="card">
    <div class="card-icon">🆕</div>
    <h2>Guide Title</h2>
    <p>Description of what this guide covers and why it's useful.</p>
    <span class="difficulty intermediate">Intermediate</span>
    <span class="card-tag">Tag</span>
</a>
```

---

