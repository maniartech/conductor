package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// Content represents user-generated content
type Content struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Type     string `json:"type"` // text, image, video
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

// ModerationResult represents the result of content moderation
type ModerationResult struct {
	Check     string    `json:"check"`
	Passed    bool      `json:"passed"`
	Score     float64   `json:"score"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("🛡️  Social Media Content Moderation System")
	fmt.Println("Real-world example: Facebook/Twitter content safety")
	fmt.Println()

	// Sample content to moderate
	content := Content{
		ID:     "POST-2025-001",
		UserID: "USER-12345",
		Type:   "text",
		Text:   "This is a sample social media post with some content to moderate!",
	}

	fmt.Printf("Moderating Content: %s\n", content.ID)
	fmt.Printf("Content Type: %s\n", content.Type)
	fmt.Printf("Text: %s\n\n", content.Text)

	start := time.Now()

	// Mixed sequential + concurrent moderation pipeline
	result, err := orchestrator.Setup(
		orchestrator.Sequential(
			// Store content data in context
			orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
				ctx.Set("content", content)
				return "Content data stored in context", nil
			}).Named("setup"),

			// First extract metadata
			orchestrator.Task(extractContentMetadata).Named("metadata"),

			// Then run parallel moderation checks
			orchestrator.Concurrent(
				orchestrator.Task(scanForExplicitContent).Named("explicit-scan"),
				orchestrator.Task(detectHateSpeech).Named("hate-speech"),
				orchestrator.Task(checkCopyrightViolation).Named("copyright"),
				orchestrator.Task(analyzeSpamIndicators).Named("spam-detection"),
			).Named("parallel-moderation"),

			// Finally make approval decision
			orchestrator.Task(makeApprovalDecision).Named("approval-decision"),
		).Named("moderation-pipeline"),
	).With(orchestrator.Config{
		ErrorStrategy: orchestrator.CollectAll, // Continue even if some checks fail
		Timeout:       10 * time.Second,
	}).Await()

	duration := time.Since(start)

	if err != nil {
		log.Printf("⚠️  Some moderation checks failed: %v", err)
	}

	fmt.Printf("✅ Content moderation completed in %v\n\n", duration)
	displayModerationResults(result)
}

// Moderation functions

func extractContentMetadata(ctx orchestrator.Context) (ModerationResult, error) {
	fmt.Println("   📊 Extracting content metadata...")
	time.Sleep(50 * time.Millisecond)

	fmt.Println("   ✅ Metadata extracted")
	return ModerationResult{
		Check:     "metadata",
		Passed:    true,
		Score:     1.0,
		Reason:    "Content metadata successfully extracted",
		Timestamp: time.Now(),
	}, nil
}

func scanForExplicitContent(ctx orchestrator.Context) (ModerationResult, error) {
	fmt.Println("   🔍 Scanning for explicit content...")
	time.Sleep(120 * time.Millisecond)

	// Simulate explicit content detection
	explicitScore := rand.Float64()
	passed := explicitScore < 0.8

	reason := "Content appears safe"
	if !passed {
		reason = "Potentially explicit content detected"
	}

	status := "✅"
	if !passed {
		status = "⚠️"
	}
	fmt.Printf("   %s Explicit content scan completed (score: %.2f)\n", status, explicitScore)

	return ModerationResult{
		Check:     "explicit-scan",
		Passed:    passed,
		Score:     explicitScore,
		Reason:    reason,
		Timestamp: time.Now(),
	}, nil
}

func detectHateSpeech(ctx orchestrator.Context) (ModerationResult, error) {
	content := ctx.Get("content").(Content)
	fmt.Println("   🗣️  Detecting hate speech...")
	time.Sleep(100 * time.Millisecond)

	// Simple hate speech detection based on keywords
	hatefulWords := []string{"hate", "terrible", "awful"}
	contentLower := strings.ToLower(content.Text)

	hateScore := 0.0
	for _, word := range hatefulWords {
		if strings.Contains(contentLower, word) {
			hateScore += 0.3
		}
	}

	passed := hateScore < 0.5
	reason := "No hate speech detected"
	if !passed {
		reason = "Potential hate speech indicators found"
	}

	status := "✅"
	if !passed {
		status = "⚠️"
	}
	fmt.Printf("   %s Hate speech detection completed (score: %.2f)\n", status, hateScore)

	return ModerationResult{
		Check:     "hate-speech",
		Passed:    passed,
		Score:     hateScore,
		Reason:    reason,
		Timestamp: time.Now(),
	}, nil
}

func checkCopyrightViolation(ctx orchestrator.Context) (ModerationResult, error) {
	fmt.Println("   ©️  Checking copyright violations...")
	time.Sleep(80 * time.Millisecond)

	// Simulate copyright check
	copyrightScore := rand.Float64() * 0.3 // Usually low
	passed := copyrightScore < 0.2

	reason := "No copyright violations detected"
	if !passed {
		reason = "Potential copyright violation detected"
	}

	status := "✅"
	if !passed {
		status = "⚠️"
	}
	fmt.Printf("   %s Copyright check completed (score: %.2f)\n", status, copyrightScore)

	return ModerationResult{
		Check:     "copyright",
		Passed:    passed,
		Score:     copyrightScore,
		Reason:    reason,
		Timestamp: time.Now(),
	}, nil
}
func analyzeSpamIndicators(ctx orchestrator.Context) (ModerationResult, error) {
	content := ctx.Get("content").(Content)
	fmt.Println("   🚫 Analyzing spam indicators...")
	time.Sleep(90 * time.Millisecond)

	// Simple spam detection
	spamIndicators := []string{"buy now", "click here", "free money", "urgent"}
	contentLower := strings.ToLower(content.Text)

	spamScore := 0.0
	for _, indicator := range spamIndicators {
		if strings.Contains(contentLower, indicator) {
			spamScore += 0.4
		}
	}

	passed := spamScore < 0.6
	reason := "No spam indicators detected"
	if !passed {
		reason = "Potential spam content detected"
	}

	status := "✅"
	if !passed {
		status = "⚠️"
	}
	fmt.Printf("   %s Spam analysis completed (score: %.2f)\n", status, spamScore)

	return ModerationResult{
		Check:     "spam-detection",
		Passed:    passed,
		Score:     spamScore,
		Reason:    reason,
		Timestamp: time.Now(),
	}, nil
}

func makeApprovalDecision(ctx orchestrator.Context) (ModerationResult, error) {
	fmt.Println("   ⚖️  Making approval decision...")
	time.Sleep(30 * time.Millisecond)

	// Simulate decision making based on content analysis
	// In a real system, this would analyze results from previous steps
	avgScore := 0.3   // Simulated average score
	failedChecks := 0 // Simulated failed checks
	approved := failedChecks == 0 && avgScore < 0.5

	decision := "APPROVED"
	reason := "Content passed all moderation checks"

	if !approved {
		if failedChecks > 2 {
			decision = "REJECTED"
			reason = "Content failed multiple moderation checks"
		} else {
			decision = "REQUIRES_HUMAN_REVIEW"
			reason = "Content requires human moderator review"
		}
	}

	status := "✅"
	if decision == "REJECTED" {
		status = "❌"
	} else if decision == "REQUIRES_HUMAN_REVIEW" {
		status = "👤"
	}

	fmt.Printf("   %s Final decision: %s\n", status, decision)

	return ModerationResult{
		Check:     "approval-decision",
		Passed:    approved,
		Score:     avgScore,
		Reason:    fmt.Sprintf("%s - %s", decision, reason),
		Timestamp: time.Now(),
	}, nil
}

func displayModerationResults(result *orchestrator.Result) {
	fmt.Println("📊 Moderation Results")
	fmt.Println(strings.Repeat("=", 60))

	checks := []string{"metadata", "explicit-scan", "hate-speech", "copyright", "spam-detection", "approval-decision"}

	for _, check := range checks {
		if checkResult := result.Get(check); checkResult != nil {
			if modResult, ok := checkResult.(ModerationResult); ok {
				status := "✅ PASS"
				if !modResult.Passed {
					status = "❌ FAIL"
				}
				if check == "approval-decision" {
					if strings.Contains(modResult.Reason, "REQUIRES_HUMAN_REVIEW") {
						status = "👤 REVIEW"
					} else if strings.Contains(modResult.Reason, "REJECTED") {
						status = "❌ REJECT"
					} else {
						status = "✅ APPROVE"
					}
				}

				fmt.Printf("%-20s | %s | Score: %.2f | %s\n",
					modResult.Check, status, modResult.Score, modResult.Reason)
			}
		}
	}
	fmt.Println()
}
