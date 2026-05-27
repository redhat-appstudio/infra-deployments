# perf-team-prometheus-reader

## Validation

```
kustomize build components/perf-team-prometheus-reader/development/
kustomize build components/perf-team-prometheus-reader/staging/base/
kustomize build components/perf-team-prometheus-reader/production/base/
```

## Production Isolation

`production/base/` has its own copies of all resource files. It does NOT reference the shared `base/` directory. When making changes:

- Modifying `base/core/` or `base/tenants-rbac/` only affects development and staging.
- Production requires a separate PR that copies the validated files into `production/base/`.
- Never refactor production to reference `../../base` — the isolation is intentional.

## General Conventions

See the [README.md](README.md) for general info about this component and workflow for changes and the repo-level [AGENTS.md](/AGENTS.md) for commit format, PR requirements, and other conventions.
