package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// BuildInfo represents build information
type BuildInfo struct {
	CommitHash  string    `json:"commit_hash"`
	Branch      string    `json:"branch"`
	BuildNumber int       `json:"build_number"`
	Timestamp   time.Time `json:"timestamp"`
}

// StepResult represents the result of a CI/CD step
type StepResult struct {
	Step      string        `json:"step"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Message   string        `json:"message"`
	Artifacts []string      `json:"artifacts,omitempty"`
}

func main() {
	fmt.Println("🚀 CI/CD Deployment Pipeline")
	fmt.Println("Real-world example: GitHub Actions, GitLab CI, Jenkins")
	fmt.Println()

	// Sample build info
	build := BuildInfo{
		CommitHash:  "a1b2c3d4",
		Branch:      "main",
		BuildNumber: 1234,
		Timestamp:   time.Now(),
	}

	fmt.Printf("Build #%d - Commit: %s (%s)\n", build.BuildNumber, build.CommitHash, build.Branch)
	fmt.Printf("Started: %s\n\n", build.Timestamp.Format("2006-01-02 15:04:05"))

	start := time.Now()

	// Complex CI/CD pipeline with conditional deployment
	result, err := orchestrator.Setup(
		orchestrator.Sequential(
			// Store build data in context
			orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
				ctx.Set("build", build)
				return "Build data stored in context", nil
			}).Named("setup"),

			// Parallel quality checks
			orchestrator.Concurrent(
				orchestrator.Task(runUnitTests).Named("unit-tests"),
				orchestrator.Task(runLintingChecks).Named("linting"),
				orchestrator.Task(performSecurityScan).Named("security-scan"),
			).Named("quality-checks"),

			// Build artifacts
			orchestrator.Task(buildDockerImage).Named("build"),

			// Deployment pipeline
			orchestrator.Sequential(
				orchestrator.Task(deployToStaging).Named("staging-deploy"),
				orchestrator.Task(runIntegrationTests).Named("integration-tests"),

				// Conditional production deployment
				orchestrator.Conditional(
					func(ctx orchestrator.Context) (bool, error) {
						// Check if tests passed and it's main branch
						buildData := ctx.Get("build").(BuildInfo)
						return buildData.Branch == "main" && rand.Float32() > 0.1, nil // 90% success rate
					},
					// Production deployment
					orchestrator.Sequential(
						orchestrator.Task(deployToProduction).Named("prod-deploy"),
						orchestrator.Task(runSmokeTests).Named("smoke-tests"),
					).Named("production-deployment"),
					// Rollback scenario
					orchestrator.Task(rollbackDeployment).Named("rollback"),
				).Named("deployment-decision"),
			).Named("deployment-pipeline"),
		).Named("cicd-pipeline"),
	).With(orchestrator.Config{
		ErrorStrategy: orchestrator.FailFast,
		Timeout:       20 * time.Minute,
	}).Await()

	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ CI/CD pipeline failed: %v", err)
		handlePipelineFailure(build, err)
		return
	}

	fmt.Printf("✅ CI/CD pipeline completed in %v\n\n", duration)
	displayPipelineResults(result, build)
}

// CI/CD pipeline functions

func runUnitTests(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🧪 Running unit tests...")

	// Simulate test execution
	time.Sleep(800 * time.Millisecond)

	// Simulate test failure (5% chance)
	if rand.Float32() < 0.05 {
		return StepResult{
			Step:     "unit-tests",
			Success:  false,
			Duration: time.Since(start),
			Message:  "Unit tests failed: 3 tests failing",
		}, fmt.Errorf("unit tests failed")
	}

	fmt.Println("   ✅ Unit tests passed (127 tests)")
	return StepResult{
		Step:     "unit-tests",
		Success:  true,
		Duration: time.Since(start),
		Message:  "All 127 unit tests passed",
	}, nil
}

func runLintingChecks(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   📝 Running linting checks...")

	time.Sleep(300 * time.Millisecond)

	fmt.Println("   ✅ Linting checks passed")
	return StepResult{
		Step:     "linting",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Code style and linting checks passed",
	}, nil
}

func performSecurityScan(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🔒 Performing security scan...")

	time.Sleep(600 * time.Millisecond)

	// Simulate security issue (3% chance)
	if rand.Float32() < 0.03 {
		return StepResult{
			Step:     "security-scan",
			Success:  false,
			Duration: time.Since(start),
			Message:  "Security vulnerabilities detected",
		}, fmt.Errorf("security scan failed")
	}

	fmt.Println("   ✅ Security scan completed")
	return StepResult{
		Step:     "security-scan",
		Success:  true,
		Duration: time.Since(start),
		Message:  "No security vulnerabilities found",
	}, nil
}

func buildDockerImage(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🐳 Building Docker image...")

	time.Sleep(1200 * time.Millisecond)

	imageTag := fmt.Sprintf("myapp:%s", build.CommitHash)
	fmt.Printf("   ✅ Docker image built: %s\n", imageTag)

	return StepResult{
		Step:      "build",
		Success:   true,
		Duration:  time.Since(start),
		Message:   fmt.Sprintf("Docker image built successfully: %s", imageTag),
		Artifacts: []string{imageTag, "build-artifacts.tar.gz"},
	}, nil
}

func deployToStaging(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🚀 Deploying to staging environment...")

	time.Sleep(400 * time.Millisecond)

	fmt.Println("   ✅ Deployed to staging")
	return StepResult{
		Step:     "staging-deploy",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Application deployed to staging environment",
	}, nil
}

func runIntegrationTests(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🔗 Running integration tests...")

	time.Sleep(1000 * time.Millisecond)

	// Simulate integration test failure (8% chance)
	if rand.Float32() < 0.08 {
		return StepResult{
			Step:     "integration-tests",
			Success:  false,
			Duration: time.Since(start),
			Message:  "Integration tests failed",
		}, fmt.Errorf("integration tests failed")
	}

	fmt.Println("   ✅ Integration tests passed")
	return StepResult{
		Step:     "integration-tests",
		Success:  true,
		Duration: time.Since(start),
		Message:  "All integration tests passed",
	}, nil
}

func deployToProduction(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   🌟 Deploying to production...")

	time.Sleep(800 * time.Millisecond)

	fmt.Println("   ✅ Deployed to production")
	return StepResult{
		Step:     "prod-deploy",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Application deployed to production",
	}, nil
}

func runSmokeTests(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   💨 Running smoke tests...")

	time.Sleep(300 * time.Millisecond)

	fmt.Println("   ✅ Smoke tests passed")
	return StepResult{
		Step:     "smoke-tests",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Production smoke tests passed",
	}, nil
}

func rollbackDeployment(ctx orchestrator.Context) (StepResult, error) {
	build := ctx.Get("build").(BuildInfo)
	start := time.Now()
	fmt.Println("   ⏪ Rolling back deployment...")

	time.Sleep(200 * time.Millisecond)

	fmt.Println("   ✅ Rollback completed")
	return StepResult{
		Step:     "rollback",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Deployment rolled back to previous version",
	}, nil
}

func displayPipelineResults(result *orchestrator.Result, build BuildInfo) {
	fmt.Println("📊 CI/CD Pipeline Results")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Build: #%d (%s)\n", build.BuildNumber, build.CommitHash)
	fmt.Printf("Branch: %s\n\n", build.Branch)

	steps := []string{"unit-tests", "linting", "security-scan", "build",
		"staging-deploy", "integration-tests", "prod-deploy", "smoke-tests", "rollback"}

	for _, step := range steps {
		if stepResult := result.Get(step); stepResult != nil {
			if pipeResult, ok := stepResult.(StepResult); ok {
				status := "✅ SUCCESS"
				if !pipeResult.Success {
					status = "❌ FAILED"
				}
				fmt.Printf("%-20s | %s | %8v | %s\n",
					pipeResult.Step, status, pipeResult.Duration, pipeResult.Message)

				if len(pipeResult.Artifacts) > 0 {
					fmt.Printf("   📦 Artifacts: %v\n", pipeResult.Artifacts)
				}
			}
		}
	}
	fmt.Println()
}

func handlePipelineFailure(build BuildInfo, err error) {
	fmt.Println("🚨 CI/CD Pipeline Failed")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Build: #%d (%s)\n", build.BuildNumber, build.CommitHash)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Failure actions:")
	fmt.Println("  - Build marked as failed")
	fmt.Println("  - Development team notified")
	fmt.Println("  - Deployment blocked")
	fmt.Println("  - Logs archived for analysis")
}
