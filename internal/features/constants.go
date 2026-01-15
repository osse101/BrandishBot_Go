package features

// ==================== File Processing ====================

// File format constants
const (
	// FeatureFileExtension is the expected file extension for feature files
	FeatureFileExtension = ".txt"

	// FeatureFileDelimiter is the delimiter that separates description from commands in feature files
	FeatureFileDelimiter = "---"
)

// ==================== Error Messages ====================

// Error messages for file operations
const (
	// ErrMsgReadDirectoryFailed is the error message when reading the feature directory fails
	ErrMsgReadDirectoryFailed = "failed to read feature directory: %w"

	// ErrMsgParseFileFailed is the error message template when parsing a feature file fails
	ErrMsgParseFileFailed = "failed to parse feature file %s: %w"
)
