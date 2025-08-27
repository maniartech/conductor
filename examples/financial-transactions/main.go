package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	FromID   string  `json:"from_id"`
	ToID     string  `json:"to_id"`
	Type     string  `json:"type"`
}

// ValidationResult represents validation step result
type ValidationResult struct {
	Step    string `json:"step"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

func main() {
	fmt.Println("💳 Financial Transaction Processing System")
	fmt.Println("Real-world example: Stripe/PayPal payment processing")
	fmt.Println()

	// Sample high-value transaction
	transaction := Transaction{
		ID:       "TXN-2025-001",
		Amount:   15000.00,
		Currency: "USD",
		FromID:   "ACC-12345",
		ToID:     "ACC-67890",
		Type:     "transfer",
	}

	fmt.Printf("Processing Transaction: %s\n", transaction.ID)
	fmt.Printf("Amount: $%.2f %s\n", transaction.Amount, transaction.Currency)
	fmt.Printf("Type: %s\n\n", transaction.Type)

	start := time.Now()

	// Conditional processing based on transaction amount
	result, err := orchestrator.Setup(
		orchestrator.Sequential(
			orchestrator.Task(func() (ValidationResult, error) {
				return validateTransactionData(transaction)
			}).Named("validation"),

			// Conditional flow based on transaction amount
			orchestrator.Conditional(
				func(ctx orchestrator.Context) (bool, error) {
					// High-value transaction threshold
					return transaction.Amount > 10000, nil
				},
				// High-value transaction flow
				orchestrator.Sequential(
					orchestrator.Task(func() (ValidationResult, error) {
						return performEnhancedKYC(transaction)
					}).Named("enhanced-kyc"),
					orchestrator.Task(func() (ValidationResult, error) {
						return requireManualApproval(transaction)
					}).Named("manual-approval"),
				).Named("high-value-flow"),
				// Standard transaction flow
				orchestrator.Concurrent(
					orchestrator.Task(func() (ValidationResult, error) {
						return performFraudCheck(transaction)
					}).Named("fraud-check"),
					orchestrator.Task(func() (ValidationResult, error) {
						return validateMerchant(transaction)
					}).Named("merchant-validation"),
					orchestrator.Task(func() (ValidationResult, error) {
						return checkRiskScore(transaction)
					}).Named("risk-assessment"),
				).Named("standard-checks"),
			).Named("risk-assessment"),

			orchestrator.Task(func() (ValidationResult, error) {
				return processPayment(transaction)
			}).Named("payment-processing"),
			orchestrator.Task(func() (ValidationResult, error) {
				return updateLedger(transaction)
			}).Named("ledger-update"),
		).Named("transaction-pipeline"),
	).With(orchestrator.Config{
		ErrorStrategy: orchestrator.FailFast,
		Timeout:       60 * time.Second,
	}).Await()

	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Transaction processing failed: %v", err)
		handleTransactionFailure(transaction, err)
		return
	}

	fmt.Printf("✅ Transaction processed successfully in %v\n\n", duration)
	displayTransactionResults(result)
}

// Transaction processing functions

func validateTransactionData(txn Transaction) (ValidationResult, error) {
	fmt.Println("   📋 Validating transaction data...")
	time.Sleep(100 * time.Millisecond)

	if txn.Amount <= 0 {
		return ValidationResult{
			Step:    "validation",
			Passed:  false,
			Message: "Invalid transaction amount",
		}, fmt.Errorf("invalid amount: %.2f", txn.Amount)
	}

	fmt.Println("   ✅ Transaction data validated")
	return ValidationResult{
		Step:    "validation",
		Passed:  true,
		Message: "Transaction data is valid",
	}, nil
}

func performEnhancedKYC(txn Transaction) (ValidationResult, error) {
	fmt.Println("   🔍 Performing enhanced KYC checks...")
	time.Sleep(300 * time.Millisecond)

	// Simulate KYC failure (5% chance)
	if rand.Float32() < 0.05 {
		return ValidationResult{
			Step:    "enhanced-kyc",
			Passed:  false,
			Message: "Enhanced KYC verification failed",
		}, fmt.Errorf("KYC verification failed for high-value transaction")
	}

	fmt.Println("   ✅ Enhanced KYC checks passed")
	return ValidationResult{
		Step:    "enhanced-kyc",
		Passed:  true,
		Message: "Enhanced KYC verification completed",
	}, nil
}

func requireManualApproval(txn Transaction) (ValidationResult, error) {
	fmt.Println("   👤 Requiring manual approval for high-value transaction...")
	time.Sleep(200 * time.Millisecond)

	// Simulate manual approval process
	fmt.Println("   ✅ Manual approval obtained")
	return ValidationResult{
		Step:    "manual-approval",
		Passed:  true,
		Message: "Manual approval granted for high-value transaction",
	}, nil
}

func performFraudCheck(txn Transaction) (ValidationResult, error) {
	fmt.Println("   🛡️  Performing fraud check...")
	time.Sleep(150 * time.Millisecond)

	// Simulate fraud detection (3% chance)
	if rand.Float32() < 0.03 {
		return ValidationResult{
			Step:    "fraud-check",
			Passed:  false,
			Message: "Potential fraud detected",
		}, fmt.Errorf("fraud check failed: suspicious activity detected")
	}

	fmt.Println("   ✅ Fraud check passed")
	return ValidationResult{
		Step:    "fraud-check",
		Passed:  true,
		Message: "No fraud indicators detected",
	}, nil
}

func validateMerchant(txn Transaction) (ValidationResult, error) {
	fmt.Println("   🏪 Validating merchant...")
	time.Sleep(100 * time.Millisecond)

	fmt.Println("   ✅ Merchant validated")
	return ValidationResult{
		Step:    "merchant-validation",
		Passed:  true,
		Message: "Merchant is verified and in good standing",
	}, nil
}

func checkRiskScore(txn Transaction) (ValidationResult, error) {
	fmt.Println("   📊 Checking risk score...")
	time.Sleep(120 * time.Millisecond)

	riskScore := rand.Float64() * 100
	passed := riskScore < 75

	message := fmt.Sprintf("Risk score: %.1f (acceptable)", riskScore)
	if !passed {
		message = fmt.Sprintf("Risk score: %.1f (too high)", riskScore)
	}

	status := "✅"
	if !passed {
		status = "⚠️"
	}
	fmt.Printf("   %s Risk assessment completed\n", status)

	return ValidationResult{
		Step:    "risk-assessment",
		Passed:  passed,
		Message: message,
	}, nil
}

func processPayment(txn Transaction) (ValidationResult, error) {
	fmt.Printf("   💰 Processing payment of $%.2f...\n", txn.Amount)
	time.Sleep(250 * time.Millisecond)

	fmt.Println("   ✅ Payment processed")
	return ValidationResult{
		Step:    "payment-processing",
		Passed:  true,
		Message: fmt.Sprintf("Payment of $%.2f processed successfully", txn.Amount),
	}, nil
}

func updateLedger(txn Transaction) (ValidationResult, error) {
	fmt.Println("   📚 Updating ledger...")
	time.Sleep(80 * time.Millisecond)

	fmt.Println("   ✅ Ledger updated")
	return ValidationResult{
		Step:    "ledger-update",
		Passed:  true,
		Message: "Transaction recorded in ledger",
	}, nil
}

func displayTransactionResults(result *orchestrator.Result) {
	fmt.Println("📊 Transaction Processing Results")
	fmt.Println(strings.Repeat("=", 50))

	// Display results from all steps
	steps := []string{"validation", "enhanced-kyc", "manual-approval", "fraud-check",
		"merchant-validation", "risk-assessment", "payment-processing", "ledger-update"}

	for _, step := range steps {
		if stepResult := result.Get(step); stepResult != nil {
			if valResult, ok := stepResult.(ValidationResult); ok {
				status := "✅ PASS"
				if !valResult.Passed {
					status = "❌ FAIL"
				}
				fmt.Printf("%-20s | %s | %s\n", valResult.Step, status, valResult.Message)
			}
		}
	}
	fmt.Println()
}

func handleTransactionFailure(txn Transaction, err error) {
	fmt.Println("🚨 Transaction Processing Failed")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Transaction ID: %s\n", txn.ID)
	fmt.Printf("Amount: $%.2f\n", txn.Amount)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Transaction marked as failed")
	fmt.Println("  - Funds authorization released")
	fmt.Println("  - Customer notified")
	fmt.Println("  - Compliance team alerted")
	fmt.Println("  - Audit trail updated")
}
