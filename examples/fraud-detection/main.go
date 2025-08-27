package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// FraudCheck represents a fraud detection check
type FraudCheck struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Amount     float64   `json:"amount"`
	Location   string    `json:"location"`
	DeviceID   string    `json:"device_id"`
	Timestamp  time.Time `json:"timestamp"`
	RiskScore  float64   `json:"risk_score"`
	Suspicious bool      `json:"suspicious"`
}

// CheckResult represents the result of a fraud check
type CheckResult struct {
	CheckType string        `json:"check_type"`
	Passed    bool          `json:"passed"`
	RiskScore float64       `json:"risk_score"`
	Duration  time.Duration `json:"duration"`
	Details   string        `json:"details"`
}

func main() {
	fmt.Println("🔍 Fraud Detection System")
	fmt.Println(strings.Repeat("=", 50))

	// Sample fraud check
	fraudCheck := FraudCheck{
		ID:        "fraud-001",
		UserID:    "user-12345",
		Amount:    2500.00,
		Location:  "New York, NY",
		DeviceID:  "device-abc123",
		Timestamp: time.Now(),
	}

	fmt.Printf("Processing fraud check: %s\n", fraudCheck.ID)
	fmt.Printf("User: %s | Amount: $%.2f\n", fraudCheck.UserID, fraudCheck.Amount)
	fmt.Printf("Location: %s | Device: %s\n\n", fraudCheck.Location, fraudCheck.DeviceID)

	// Create fraud detection workflow
	workflow := orchestrator.Setup(
		orchestrator.Sequential(
			// Store fraud check data in context
			orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
				ctx.Set("fraudCheck", fraudCheck)
				return "Fraud check data stored in context", nil
			}).Named("setup"),

			// Parallel fraud checks
			orchestrator.Concurrent(
				orchestrator.Task(performVelocityCheck).Named("velocity-check"),
				orchestrator.Task(performLocationCheck).Named("location-check"),
				orchestrator.Task(performDeviceCheck).Named("device-check"),
				orchestrator.Task(performBehaviorCheck).Named("behavior-check"),
			).Named("fraud-checks"),
		).Named("fraud-detection-pipeline"),
	)

	// Execute workflow
	result, err := workflow.Await()
	if err != nil {
		handleFraudCheckFailure(fraudCheck, err)
		return
	}

	// Display results
	displayFraudResults(result, fraudCheck)
}

// Fraud check implementations
func performVelocityCheck(ctx orchestrator.Context) (CheckResult, error) {
	check := ctx.Get("fraudCheck").(FraudCheck)
	start := time.Now()

	// Simulate velocity analysis
	time.Sleep(time.Duration(rand.IntN(300)+100) * time.Millisecond)

	// Check transaction velocity (simplified)
	riskScore := 0.0
	passed := true
	details := "Normal transaction velocity"

	if check.Amount > 5000 {
		riskScore = 0.7
		passed = false
		details = "High amount transaction detected"
	} else if check.Amount > 1000 {
		riskScore = 0.3
		details = "Medium amount transaction"
	}

	return CheckResult{
		CheckType: "velocity",
		Passed:    passed,
		RiskScore: riskScore,
		Duration:  time.Since(start),
		Details:   details,
	}, nil
}

func performLocationCheck(ctx orchestrator.Context) (CheckResult, error) {
	check := ctx.Get("fraudCheck").(FraudCheck)
	start := time.Now()

	// Simulate location analysis
	time.Sleep(time.Duration(rand.IntN(200)+50) * time.Millisecond)

	// Check location patterns (simplified)
	riskScore := 0.0
	passed := true
	details := "Location verified"

	suspiciousLocations := []string{"Unknown", "VPN", "Proxy"}
	for _, suspicious := range suspiciousLocations {
		if strings.Contains(check.Location, suspicious) {
			riskScore = 0.8
			passed = false
			details = "Suspicious location detected"
			break
		}
	}

	return CheckResult{
		CheckType: "location",
		Passed:    passed,
		RiskScore: riskScore,
		Duration:  time.Since(start),
		Details:   details,
	}, nil
}

func performDeviceCheck(ctx orchestrator.Context) (CheckResult, error) {
	check := ctx.Get("fraudCheck").(FraudCheck)
	start := time.Now()

	// Simulate device fingerprinting
	time.Sleep(time.Duration(rand.IntN(250)+75) * time.Millisecond)

	// Check device patterns (simplified)
	riskScore := 0.0
	passed := true
	details := "Device recognized"

	if len(check.DeviceID) < 10 {
		riskScore = 0.5
		passed = false
		details = "Suspicious device fingerprint"
	}

	return CheckResult{
		CheckType: "device",
		Passed:    passed,
		RiskScore: riskScore,
		Duration:  time.Since(start),
		Details:   details,
	}, nil
}

func performBehaviorCheck(ctx orchestrator.Context) (CheckResult, error) {
	check := ctx.Get("fraudCheck").(FraudCheck)
	start := time.Now()

	// Simulate behavioral analysis
	time.Sleep(time.Duration(rand.IntN(400)+150) * time.Millisecond)

	// Check behavioral patterns (simplified)
	riskScore := 0.0
	passed := true
	details := "Normal behavior pattern"

	// Check for unusual timing
	hour := check.Timestamp.Hour()
	if hour < 6 || hour > 23 {
		riskScore = 0.4
		details = "Unusual transaction time"
	}

	return CheckResult{
		CheckType: "behavior",
		Passed:    passed,
		RiskScore: riskScore,
		Duration:  time.Since(start),
		Details:   details,
	}, nil
}

// Helper functions
func displayFraudResults(result *orchestrator.Result, check FraudCheck) {
	fmt.Println("📊 Fraud Detection Results")
	fmt.Println(strings.Repeat("=", 50))

	checks := []string{"velocity-check", "location-check", "device-check", "behavior-check"}
	totalRiskScore := 0.0
	failedChecks := 0

	for _, checkName := range checks {
		if checkResult := result.Get(checkName); checkResult != nil {
			if fraudResult, ok := checkResult.(CheckResult); ok {
				status := "✅ PASS"
				if !fraudResult.Passed {
					status = "❌ FAIL"
					failedChecks++
				}

				totalRiskScore += fraudResult.RiskScore

				fmt.Printf("%s %-15s | Risk: %.2f | %8v | %s\n",
					status, fraudResult.CheckType, fraudResult.RiskScore,
					fraudResult.Duration, fraudResult.Details)
			}
		}
	}

	fmt.Println()
	avgRiskScore := totalRiskScore / float64(len(checks))

	// Final decision
	decision := "APPROVED"
	if failedChecks > 1 || avgRiskScore > 0.6 {
		decision = "BLOCKED"
	} else if failedChecks > 0 || avgRiskScore > 0.3 {
		decision = "REVIEW"
	}

	fmt.Printf("Final Decision: %s\n", decision)
	fmt.Printf("Average Risk Score: %.2f\n", avgRiskScore)
	fmt.Printf("Failed Checks: %d/%d\n", failedChecks, len(checks))
	fmt.Println()
}

func handleFraudCheckFailure(check FraudCheck, err error) {
	fmt.Println("🚨 Fraud Detection Failed")
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("Check ID: %s\n", check.ID)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Transaction flagged for manual review")
	fmt.Println("  - User notified of security check")
	fmt.Println("  - Incident logged for analysis")
}
