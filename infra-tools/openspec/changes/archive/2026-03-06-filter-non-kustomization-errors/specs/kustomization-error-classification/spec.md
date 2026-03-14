## ADDED Requirements

### Requirement: Non-kustomization errors are classified
The engine SHALL classify kustomize build errors as "non-kustomization" when the error message indicates the directory does not contain a kustomization file. All other build errors SHALL be classified as genuine build errors.

#### Scenario: Directory without kustomization.yaml
- **WHEN** `kustomize build` fails because the directory has no `kustomization.yaml`
- **THEN** the error SHALL be classified as a non-kustomization error

#### Scenario: Genuine kustomize build failure
- **WHEN** `kustomize build` fails for any other reason (e.g., malformed YAML, missing referenced resources)
- **THEN** the error SHALL be classified as a genuine build error

### Requirement: Non-kustomization errors are excluded from output
Components with non-kustomization errors SHALL NOT appear in any user-facing output, including local stdout, CI summary, CI PR comment, and diff file output.

#### Scenario: Local output excludes non-kustomization errors
- **WHEN** render-diff runs in local mode and a component has a non-kustomization error
- **THEN** no BUILD ERROR header or error message SHALL be printed for that component
- **THEN** the component SHALL NOT appear in the summary

#### Scenario: CI summary excludes non-kustomization errors
- **WHEN** render-diff runs in CI summary mode and a component has a non-kustomization error
- **THEN** no details/summary block SHALL be written for that component

#### Scenario: CI PR comment excludes non-kustomization errors
- **WHEN** render-diff runs in CI comment mode and a component has a non-kustomization error
- **THEN** no table row SHALL be written for that component

#### Scenario: Component counts exclude non-kustomization errors
- **WHEN** the summary reports the number of components with differences
- **THEN** components with non-kustomization errors SHALL NOT be included in the count

### Requirement: Non-kustomization errors are logged
Non-kustomization errors SHALL be logged for diagnostic purposes so operators can verify which directories were skipped.

#### Scenario: Error appears in log output
- **WHEN** a component has a non-kustomization error
- **THEN** a log message SHALL be emitted at WARN level or higher with the component path and error details

### Requirement: Genuine build errors remain visible
Genuine kustomize build errors SHALL continue to be displayed in all output modes exactly as they are today.

#### Scenario: Genuine error in local output
- **WHEN** render-diff runs in local mode and a component has a genuine build error
- **THEN** a BUILD ERROR header and error message SHALL be printed for that component

#### Scenario: Genuine error in CI output
- **WHEN** render-diff runs in any CI output mode and a component has a genuine build error
- **THEN** the error SHALL appear in the summary, comment, or artifact output
