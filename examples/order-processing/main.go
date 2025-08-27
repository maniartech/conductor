package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// Order represents an e-commerce order
type Order struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	Items       []Item  `json:"items"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
}

// Item represents an order item
type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// ProcessingResult represents the result of each processing step
type ProcessingResult struct {
	Step      string        `json:"step"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

func main() {
	fmt.Println("🛒 E-commerce Order Processing Pipeline")
	fmt.Println("Real-world example: Amazon/Shopify order fulfillment")
	fmt.Println()

	// Create sample order
	order := Order{
		ID:         "ORD-2025-001",
		CustomerID: "CUST-12345",
		Items: []Item{
			{ProductID: "PROD-001", Quantity: 2, Price: 29.99},
			{ProductID: "PROD-002", Quantity: 1, Price: 49.99},
		},
		TotalAmount: 109.97,
		Status:      "PENDING",
	}

	fmt.Printf("Processing Order: %s (Total: $%.2f)\n\n", order.ID, order.TotalAmount)

	start := time.Now()

	// Execute order processing pipeline sequentially
	result, err := orchestrator.Setup(
		orchestrator.Sequential(
			orchestrator.Task(func() (ProcessingResult, error) {
				return validateOrderData(order)
			}).Named("validation"),
			orchestrator.Task(func() (ProcessingResult, error) {
				return checkInventoryAvailability(order)
			}).Named("inventory-check"),
			orchestrator.Task(func() (ProcessingResult, error) {
				return processPayment(order)
			}).Named("payment"),
			orchestrator.Task(func() (ProcessingResult, error) {
				return reserveInventory(order)
			}).Named("reservation"),
			orchestrator.Task(func() (ProcessingResult, error) {
				return createShippingLabel(order)
			}).Named("shipping"),
			orchestrator.Task(func() (ProcessingResult, error) {
				return sendConfirmationEmail(order)
			}).Named("confirmation"),
		).Named("order-pipeline"),
	).With(orchestrator.Config{
		ErrorStrategy: orchestrator.FailFast, // Stop on any failure
		Timeout:       30 * time.Second,
	}).Await()

	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Order processing failed: %v", err)
		handleOrderFailure(order, err)
		return
	}

	fmt.Printf("✅ Order processed successfully in %v\n\n", duration)
	displayProcessingResults(result)

	fmt.Println("🎉 Order fulfillment completed!")
}

// Order processing pipeline functions

func validateOrderData(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Println("   📋 Validating order data...")

	// Simulate validation time
	time.Sleep(100 * time.Millisecond)

	// Simulate validation failure (5% chance)
	if rand.Float32() < 0.05 {
		return ProcessingResult{
			Step:      "validation",
			Success:   false,
			Message:   "Invalid customer data",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}, fmt.Errorf("order validation failed: invalid customer data")
	}

	fmt.Println("   ✅ Order data validated")
	return ProcessingResult{
		Step:      "validation",
		Success:   true,
		Message:   "Order data is valid",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}

func checkInventoryAvailability(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Println("   📦 Checking inventory availability...")

	// Simulate inventory check time
	time.Sleep(150 * time.Millisecond)

	// Simulate out of stock (10% chance)
	if rand.Float32() < 0.1 {
		return ProcessingResult{
			Step:      "inventory-check",
			Success:   false,
			Message:   "Insufficient inventory for requested items",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}, fmt.Errorf("inventory check failed: insufficient stock")
	}

	fmt.Println("   ✅ Inventory available")
	return ProcessingResult{
		Step:      "inventory-check",
		Success:   true,
		Message:   "All items available in inventory",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}

func processPayment(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Printf("   💳 Processing payment of $%.2f...\n", order.TotalAmount)

	// Simulate payment processing time
	time.Sleep(200 * time.Millisecond)

	// Simulate payment failure (8% chance)
	if rand.Float32() < 0.08 {
		return ProcessingResult{
			Step:      "payment",
			Success:   false,
			Message:   "Payment declined by bank",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}, fmt.Errorf("payment processing failed: card declined")
	}

	fmt.Println("   ✅ Payment processed successfully")
	return ProcessingResult{
		Step:      "payment",
		Success:   true,
		Message:   fmt.Sprintf("Payment of $%.2f processed", order.TotalAmount),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}
func reserveInventory(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Println("   🔒 Reserving inventory...")

	// Simulate inventory reservation time
	time.Sleep(120 * time.Millisecond)

	fmt.Println("   ✅ Inventory reserved")
	return ProcessingResult{
		Step:      "reservation",
		Success:   true,
		Message:   "Inventory reserved for order",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}

func createShippingLabel(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Println("   📮 Creating shipping label...")

	// Simulate shipping label creation time
	time.Sleep(80 * time.Millisecond)

	fmt.Println("   ✅ Shipping label created")
	return ProcessingResult{
		Step:      "shipping",
		Success:   true,
		Message:   "Shipping label generated",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}

func sendConfirmationEmail(order Order) (ProcessingResult, error) {
	start := time.Now()
	fmt.Println("   📧 Sending confirmation email...")

	// Simulate email sending time
	time.Sleep(50 * time.Millisecond)

	fmt.Println("   ✅ Confirmation email sent")
	return ProcessingResult{
		Step:      "confirmation",
		Success:   true,
		Message:   "Order confirmation email sent to customer",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}, nil
}

// Helper functions

func displayProcessingResults(result *orchestrator.Result) {
	fmt.Println("📊 Processing Results")
	fmt.Println(strings.Repeat("=", 50))

	steps := []string{"validation", "inventory-check", "payment", "reservation", "shipping", "confirmation"}

	for _, step := range steps {
		if stepResult := result.Get(step); stepResult != nil {
			if procResult, ok := stepResult.(ProcessingResult); ok {
				status := "✅"
				if !procResult.Success {
					status = "❌"
				}
				fmt.Printf("%s %-15s | %8v | %s\n",
					status, procResult.Step, procResult.Duration, procResult.Message)
			}
		}
	}
	fmt.Println()
}

func handleOrderFailure(order Order, err error) {
	fmt.Println("🚨 Order Processing Failed")
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("Order ID: %s\n", order.ID)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Order status updated to FAILED")
	fmt.Println("  - Customer notified of failure")
	fmt.Println("  - Inventory holds released")
	fmt.Println("  - Payment authorization voided")
	fmt.Println("  - Support ticket created")
}
