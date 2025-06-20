package sync

// Options contains settings that modify the sync's operation.
type Options struct {
	// K6Version is the version of k6 to use, overriding the version in go.mod.
	K6Version string
	// DryRun is a flag that indicates whether the sync should omit changes.
	DryRun bool
}
