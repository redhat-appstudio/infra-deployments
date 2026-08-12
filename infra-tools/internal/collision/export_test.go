package collision

// Test-only exports so the external collision_test package can exercise
// unexported helpers directly without duplicating logic or requiring a real
// kustomize build on disk for every test case.
var (
	ExtractAppSets = extractAppSets
	FindCollisions = findCollisions
	StringField    = stringField
)
