package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Helper to check if a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// Walk all *_test.go files and count patterns
func countTestPatterns(t *testing.T) (basicRace, advancedRace, fuzz, stress, leak int, filesScanned int) {
	t.Helper()

	reBasicRace := regexp.MustCompile(`(?m)^func\s+TestRaceCondition_[A-Za-z0-9_]+\s*\(t \*testing\.T\)`)
	reAdvancedRace := regexp.MustCompile(`(?m)^func\s+TestAdvancedRaceConditions_[A-Za-z0-9_]+\s*\(t \*testing\.T\)`)
	reFuzz := regexp.MustCompile(`(?m)^func\s+Fuzz[^\(]+\(`)
	reStress := regexp.MustCompile(`(?m)^func\s+Test(?:Stress|ProductionStress)_[A-Za-z0-9_]+\s*\(t \*testing\.T\)`)
	reLeak := regexp.MustCompile(`(?m)^func\s+Test\w*(GoroutineLeak|GoroutineLifecycle|LeakDetection)\w*\s*\(t \*testing\.T\)`)

	_ = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Skip common non-source folders, but don't skip the root "."
			if name != "." && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		bytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		filesScanned++
		content := string(bytes)
		basicRace += len(reBasicRace.FindAllString(content, -1))
		advancedRace += len(reAdvancedRace.FindAllString(content, -1))
		fuzz += len(reFuzz.FindAllString(content, -1))
		stress += len(reStress.FindAllString(content, -1))
		leak += len(reLeak.FindAllString(content, -1))
		return nil
	})

	return
}

func maybeValidateRaceFlag(t *testing.T) {
	t.Helper()
	if os.Getenv("VALIDATE_RACE") == "" {
		t.Log("Skipping '-race' validation. Set VALIDATE_RACE=1 to enable.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-race", "./...")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("`go test -race ./...` failed: %v\n%s", err, string(out))
	}
	t.Log("`go test -race ./...` passed")
}

func TestAdvancedTestingSuiteValidation(t *testing.T) {
	basicRace, advancedRace, fuzz, stress, leak, files := countTestPatterns(t)

	// Existence assertions (thresholds, not brittle exact counts)
	if basicRace == 0 {
		t.Error("No basic race-condition tests found (TestRaceCondition_*)")
	}
	if advancedRace == 0 {
		t.Error("No advanced race-condition tests found (TestAdvancedRaceConditions_*)")
	}
	if fuzz == 0 {
		t.Error("No fuzzing tests found (Fuzz*)")
	}
	if stress == 0 {
		t.Error("No stress tests found (TestStress_* or TestProductionStress_*)")
	}
	if leak == 0 {
		t.Error("No goroutine leak detection tests found")
	}

	// Documentation check
	docsOK := fileExists("ADVANCED_TESTING_DOCUMENTATION.md")
	if !docsOK {
		t.Error("ADVANCED_TESTING_DOCUMENTATION.md not found")
	}

	// Summary logs
	t.Log("✅ Test suite discovery summary")
	t.Logf("   - Test files scanned: %d", files)
	t.Logf("   - Basic race-condition tests: %d", basicRace)
	t.Logf("   - Advanced race-condition tests: %d", advancedRace)
	t.Logf("   - Fuzzing tests: %d", fuzz)
	t.Logf("   - Stress tests: %d", stress)
	t.Logf("   - Goroutine leak detection tests: %d", leak)
	t.Logf("   - Advanced testing docs present: %v", docsOK)

	// Optional: validate with -race if requested
	maybeValidateRaceFlag(t)

	// Guidance for maintainers (kept as logs, assertions stay threshold-based)
	t.Log("ℹ️ To update thresholds or categories, adjust regexes in advanced_testing_suite_validation_test.go")
	if fuzz > 0 {
		t.Logf("ℹ️ Fuzz tests discovered: %d. Use 'go test -fuzz=.' to run fuzzing.", fuzz)
	}
}
