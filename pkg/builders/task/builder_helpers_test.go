package task

// Shared helper functions extracted from original monolithic test file.

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}
	if len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr) {
		return true
	}
	return containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
