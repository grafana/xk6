package lint

// Options contains settings that modify the linter's operation.
type Options struct {
	// Preset is the ID of the preset to use.
	Preset PresetID

	// Enable is a list of checkers to enable in addition to those in the preset.
	Enable []CheckID

	// Disable is a list of checkers to disable from those in the preset.
	Disable []CheckID

	// EnableOnly, if set, makes the linter run only the checks in this list,
	// ignoring the preset and other Enabled/Disabled options.
	EnableOnly []CheckID
}
