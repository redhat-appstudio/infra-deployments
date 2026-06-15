---
name: brainstorming-workflow
description: >
  Use when in an interactive session and the user requests a new feature, significant
  change, rollout strategy, or migration in infra-deployments. Provides a structured
  process choice before any changes are made. Skip when dispatched with a complete task.
---

# Brainstorming Workflow

Discipline for interactive sessions involving new features, rollout planning, overlay restructuring, migrations, or other significant changes.

## Context Detection

- **Interactive session** (human in CLI/IDE): follow this workflow.
- **Dispatched with a complete task** (sub-agent, automation, explicit spec): skip entirely and execute.

## First Message

Before making any changes, ask exactly ONE question:

> I can approach this a few ways:
>
> A) Jump straight to making changes
> B) Discuss approaches first, then make changes
> C) Full design process — explore approaches, write up a plan, then execute
>
> Which works for you?

If the human says "just do it", gives a direct instruction, or otherwise signals urgency, treat as **A**.

## Path A — Jump to Changes

Proceed directly. All existing conventions still apply (pr-workflow, kustomize build validation). No additional ceremony.

## Path B — Discuss Approaches

1. **Understand the problem**: what is being changed, why, and any constraints.
2. **Propose 2-3 approaches** with trade-offs (blast radius, complexity, number of PRs, ring strategy).
3. **Lead with a recommendation** and explain why.
4. Let the human choose, then execute.

Infra-deployments examples where this helps:
- Choosing between simple vs temp-ring base approach for a production rollout
- Deciding whether a change should go through staging first or can use the hotfix bypass
- Planning how to restructure component overlays across clusters
- Evaluating whether a shared base change is safe or needs staging-first validation

Ask one question at a time. Prefer multiple choice over open-ended questions.

## Path C — Full Design Process

Everything in Path B, plus:

1. **Write up the plan** — what changes in which files, which clusters, how many PRs/rings.
2. **Break into ordered steps** with dependencies (e.g., staging PR first, then ring 1, then ring 2).
3. Get human approval before executing.

## Key Principles

- **One question at a time.** Never pile up multiple questions in one message.
- **Prefer multiple choice.** Easier for the human to decide quickly.
- **Human decides the process, not the agent.** Respect the chosen path.
- **"Just do it" means just do it.** Don't add process the human didn't ask for.
