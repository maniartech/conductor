package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// SupplyChainOrder represents a supply chain order
type SupplyChainOrder struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	Quantity     int       `json:"quantity"`
	Priority     string    `json:"priority"` // LOW, MEDIUM, HIGH, URGENT
	Destination  string    `json:"destination"`
	OrderDate    time.Time `json:"order_date"`
	RequiredDate time.Time `json:"required_date"`
	Value        float64   `json:"value"`
}

// ProcessingResult represents the result of a processing step
type ProcessingResult struct {
	Step     string        `json:"step"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Message  string        `json:"message"`
	Data     interface{}   `json:"data,omitempty"`
}

func main() {
	fmt.Println("📦 Supply Chain Management System")
	fmt.Println(strings.Repeat("=", 50))

	// Sample supply chain order
	order := SupplyChainOrder{
		ID:           "order-001",
		ProductID:    "PROD-12345",
		Quantity:     100,
		Priority:     "HIGH",
		Destination:  "New York Distribution Center",
		OrderDate:    time.Now(),
		RequiredDate: time.Now().Add(72 * time.Hour),
		Value:        15000.00,
	}

	fmt.Printf("Processing order: %s\n", order.ID)
	fmt.Printf("Product: %s | Quantity: %d | Priority: %s\n",
		order.ProductID, order.Quantity, order.Priority)
	fmt.Printf("Destination: %s | Value: $%.2f\n\n", order.Destination, order.Value)

	// Create supply chain workflow
	workflow := orchestrator.Setup(
		orchestrator.Sequential(
			// Store order data in context
			orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
				ctx.Set("order", order)
				return "Order data stored in context", nil
			}).Named("setup"),

			// Order validation and planning
			orchestrator.Task(validateOrder).Named("order-validation"),

			// Parallel inventory and supplier checks
			orchestrator.Concurrent(
				orchestrator.Task(checkInventory).Named("inventory-check"),
				orchestrator.Task(verifySupplier).Named("supplier-verification"),
				orchestrator.Task(calculateShipping).Named("shipping-calculation"),
			).Named("parallel-checks"),

			// Conditional procurement based on inventory
			orchestrator.Conditional(
				func(ctx orchestrator.Context) (bool, error) {
					// Check if procurement is needed
					if inventoryResult := ctx.Get("inventory-check"); inventoryResult != nil {
						if result, ok := inventoryResult.(ProcessingResult); ok {
							if data, ok := result.Data.(map[string]interface{}); ok {
								return data["procurement_needed"].(bool), nil
							}
						}
					}
					return false, nil
				},
				// Procurement path
				orchestrator.Sequential(
					orchestrator.Task(initiateProcurement).Named("procurement-initiation"),
					orchestrator.Task(trackProcurement).Named("procurement-tracking"),
				).Named("procurement-process"),
				// Direct fulfillment path
				orchestrator.Task(prepareDirectFulfillment).Named("direct-fulfillment"),
			).Named("conditional-procurement"),

			// Fulfillment and logistics
			orchestrator.Concurrent(
				orchestrator.Task(scheduleProduction).Named("production-scheduling"),
				orchestrator.Task(arrangeLogistics).Named("logistics-arrangement"),
				orchestrator.Task(prepareDocumentation).Named("documentation-prep"),
			).Named("fulfillment-logistics"),

			// Final steps
			orchestrator.Task(updateOrderStatus).Named("status-update"),
			orchestrator.Task(notifyStakeholders).Named("stakeholder-notification"),
		).Named("supply-chain-workflow"),
	)

	// Execute workflow
	result, err := workflow.Await()
	if err != nil {
		handleSupplyChainFailure(order, err)
		return
	}

	// Display results
	displaySupplyChainResults(result, order)
}

// Supply chain processing implementations
func validateOrder(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate order validation
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	success := true
	message := "Order validated successfully"

	// Basic validation checks
	if order.ID == "" || order.ProductID == "" {
		success = false
		message = "Missing required order information"
	} else if order.Quantity <= 0 {
		success = false
		message = "Invalid order quantity"
	} else if order.RequiredDate.Before(time.Now()) {
		success = false
		message = "Required date is in the past"
	}

	return ProcessingResult{
		Step:     "order-validation",
		Success:  success,
		Duration: time.Since(start),
		Message:  message,
	}, nil
}

func checkInventory(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate inventory check
	time.Sleep(time.Duration(rand.IntN(300)+150) * time.Millisecond)

	// Simulate inventory levels
	availableStock := rand.IntN(150) + 50
	procurementNeeded := availableStock < order.Quantity

	message := fmt.Sprintf("Available stock: %d units", availableStock)
	if procurementNeeded {
		message += " - Procurement required"
	}

	return ProcessingResult{
		Step:     "inventory-check",
		Success:  true,
		Duration: time.Since(start),
		Message:  message,
		Data: map[string]interface{}{
			"available_stock":    availableStock,
			"procurement_needed": procurementNeeded,
			"shortage_quantity":  max(0, order.Quantity-availableStock),
		},
	}, nil
}

func verifySupplier(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate supplier verification
	time.Sleep(time.Duration(rand.IntN(250)+125) * time.Millisecond)

	supplierRating := 4.2 + rand.Float64()*0.8 // 4.2-5.0 rating

	return ProcessingResult{
		Step:     "supplier-verification",
		Success:  true,
		Duration: time.Since(start),
		Message:  fmt.Sprintf("Supplier verified - Rating: %.1f/5.0", supplierRating),
		Data: map[string]interface{}{
			"supplier_id":     "SUP-" + order.ProductID,
			"supplier_rating": supplierRating,
			"lead_time_days":  rand.IntN(14) + 3,
		},
	}, nil
}

func calculateShipping(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate shipping calculation
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	shippingCost := float64(order.Quantity)*2.5 + rand.Float64()*100
	estimatedDays := rand.IntN(7) + 2

	return ProcessingResult{
		Step:     "shipping-calculation",
		Success:  true,
		Duration: time.Since(start),
		Message:  fmt.Sprintf("Shipping cost: $%.2f, ETA: %d days", shippingCost, estimatedDays),
		Data: map[string]interface{}{
			"shipping_cost":   shippingCost,
			"estimated_days":  estimatedDays,
			"shipping_method": "Express Freight",
		},
	}, nil
}

func initiateProcurement(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate procurement initiation
	time.Sleep(time.Duration(rand.IntN(400)+200) * time.Millisecond)

	procurementID := "PROC-" + order.ID

	return ProcessingResult{
		Step:     "procurement-initiation",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Procurement order initiated with supplier",
		Data: map[string]interface{}{
			"procurement_id":   procurementID,
			"expected_arrival": time.Now().Add(48 * time.Hour),
		},
	}, nil
}

func trackProcurement(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate procurement tracking
	time.Sleep(time.Duration(rand.IntN(150)+75) * time.Millisecond)

	return ProcessingResult{
		Step:     "procurement-tracking",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Procurement tracking system activated",
		Data: map[string]interface{}{
			"tracking_id": "TRK-" + order.ID,
			"status":      "in_transit",
		},
	}, nil
}

func prepareDirectFulfillment(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate direct fulfillment preparation
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	return ProcessingResult{
		Step:     "direct-fulfillment",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Direct fulfillment prepared from existing inventory",
		Data: map[string]interface{}{
			"fulfillment_id": "FUL-" + order.ID,
			"ready_date":     time.Now().Add(24 * time.Hour),
		},
	}, nil
}

func scheduleProduction(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate production scheduling
	time.Sleep(time.Duration(rand.IntN(300)+150) * time.Millisecond)

	productionSlot := time.Now().Add(time.Duration(rand.IntN(48)+12) * time.Hour)

	return ProcessingResult{
		Step:     "production-scheduling",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Production scheduled successfully",
		Data: map[string]interface{}{
			"production_slot":      productionSlot,
			"line_number":          rand.IntN(5) + 1,
			"estimated_completion": productionSlot.Add(8 * time.Hour),
		},
	}, nil
}

func arrangeLogistics(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate logistics arrangement
	time.Sleep(time.Duration(rand.IntN(250)+125) * time.Millisecond)

	return ProcessingResult{
		Step:     "logistics-arrangement",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Logistics and transportation arranged",
		Data: map[string]interface{}{
			"carrier":         "FastShip Logistics",
			"tracking_number": "FS" + order.ID,
			"pickup_date":     time.Now().Add(36 * time.Hour),
		},
	}, nil
}

func prepareDocumentation(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate documentation preparation
	time.Sleep(time.Duration(rand.IntN(150)+75) * time.Millisecond)

	return ProcessingResult{
		Step:     "documentation-prep",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Shipping and customs documentation prepared",
		Data: map[string]interface{}{
			"invoice_number":  "INV-" + order.ID,
			"customs_cleared": true,
		},
	}, nil
}

func updateOrderStatus(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate status update
	time.Sleep(time.Duration(rand.IntN(100)+50) * time.Millisecond)

	return ProcessingResult{
		Step:     "status-update",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Order status updated to 'In Progress'",
		Data: map[string]interface{}{
			"status":     "in_progress",
			"updated_at": time.Now(),
		},
	}, nil
}

func notifyStakeholders(ctx orchestrator.Context) (ProcessingResult, error) {
	order := ctx.Get("order").(SupplyChainOrder)
	start := time.Now()

	// Simulate stakeholder notification
	time.Sleep(time.Duration(rand.IntN(100)+50) * time.Millisecond)

	return ProcessingResult{
		Step:     "stakeholder-notification",
		Success:  true,
		Duration: time.Since(start),
		Message:  "All stakeholders notified of order progress",
		Data: map[string]interface{}{
			"notifications_sent":    5,
			"notification_channels": []string{"email", "sms", "dashboard"},
		},
	}, nil
}

// Helper functions
func displaySupplyChainResults(result *orchestrator.Result, order SupplyChainOrder) {
	fmt.Println("📊 Supply Chain Processing Results")
	fmt.Println(strings.Repeat("=", 60))

	steps := []string{"order-validation", "inventory-check", "supplier-verification",
		"shipping-calculation", "procurement-initiation", "procurement-tracking",
		"direct-fulfillment", "production-scheduling", "logistics-arrangement",
		"documentation-prep", "status-update", "stakeholder-notification"}

	totalDuration := time.Duration(0)
	successfulSteps := 0
	procurementNeeded := false

	for _, step := range steps {
		if stepResult := result.Get(step); stepResult != nil {
			if procResult, ok := stepResult.(ProcessingResult); ok {
				status := "✅"
				if !procResult.Success {
					status = "❌"
				} else {
					successfulSteps++
				}

				totalDuration += procResult.Duration

				// Check if procurement was needed
				if step == "inventory-check" && procResult.Data != nil {
					if data, ok := procResult.Data.(map[string]interface{}); ok {
						if needed, ok := data["procurement_needed"].(bool); ok {
							procurementNeeded = needed
						}
					}
				}

				fmt.Printf("%s %-25s | %8v | %s\n",
					status, procResult.Step, procResult.Duration, procResult.Message)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Processing Summary:\n")
	fmt.Printf("  Order: %s (%s)\n", order.ID, order.ProductID)
	fmt.Printf("  Quantity: %d | Priority: %s\n", order.Quantity, order.Priority)
	fmt.Printf("  Destination: %s\n", order.Destination)
	fmt.Printf("  Total Duration: %v\n", totalDuration)
	fmt.Printf("  Successful Steps: %d/%d\n", successfulSteps, len(steps))
	fmt.Printf("  Procurement Required: %v\n", procurementNeeded)
	fmt.Printf("  Order Value: $%.2f\n", order.Value)
	fmt.Println()
}

func handleSupplyChainFailure(order SupplyChainOrder, err error) {
	fmt.Println("🚨 Supply Chain Processing Failed")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Order: %s (%s)\n", order.ID, order.ProductID)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Order flagged for manual review")
	fmt.Println("  - Alternative suppliers contacted")
	fmt.Println("  - Customer notified of potential delay")
	fmt.Println("  - Escalation to supply chain manager")
}
