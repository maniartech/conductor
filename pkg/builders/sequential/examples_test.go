package sequential

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/errors"
)

// TestExample_SimpleSequential demonstrates basic sequential orchestration
func TestExample_SimpleSequential(t *testing.T) {
	t.Log("=== Example: Simple Sequential Orchestration ===")

	// Create a simple 3-step sequential process
	seq := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 1: Fetching user data...")
			return "user-123", nil
		}).Named("fetch-user"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 2: Validating user...")
			return "validated", nil
		}).Named("validate-user"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 3: Processing user...")
			return "processed", nil
		}).Named("process-user"),
	).Named("simple-user-pipeline")

	ctx := context.Background()
	cfg := config.Config{}

	result, err := seq.Execute(ctx, cfg)

	if err != nil {
		t.Fatalf("Simple example failed: %v", err)
	}

	t.Logf("✅ Simple pipeline completed successfully")
	t.Logf("   - User ID: %v", result.Get("fetch-user"))
	t.Logf("   - Validation: %v", result.Get("validate-user"))
	t.Logf("   - Processing: %v", result.Get("process-user"))
}

// TestExample_ErrorHandling demonstrates different error strategies
func TestExample_ErrorHandling(t *testing.T) {
	t.Log("=== Example: Error Handling Strategies ===")

	// Test FailFast strategy
	t.Log("--- Testing FailFast Strategy ---")
	failFastSeq := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 1: Success")
			return "success", nil
		}).Named("success-step"),

		task.Task(func(ctx orchContext.Context) (int, error) {
			t.Log("Step 2: Failure")
			return 0, stderrors.New("intentional failure")
		}).Named("failure-step"),

		task.Task(func(ctx orchContext.Context) (bool, error) {
			t.Log("Step 3: Should not execute")
			return true, nil
		}).Named("skipped-step"),
	).Named("fail-fast-pipeline")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := failFastSeq.Execute(ctx, cfg)
	if err == nil {
		t.Error("Expected error from FailFast strategy")
	} else {
		t.Logf("❌ FailFast stopped at first error")
		t.Logf("   - Success step result: %v", result.Get("success-step"))
		t.Logf("   - Skipped step result: %v", result.Get("skipped-step")) // Should be nil
	}

	// Test CollectAll strategy
	t.Log("--- Testing CollectAll Strategy ---")
	collectAllSeq := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 1: Success")
			return "success", nil
		}).Named("success-step"),

		task.Task(func(ctx orchContext.Context) (int, error) {
			t.Log("Step 2: Failure")
			return 0, stderrors.New("intentional failure")
		}).Named("failure-step"),

		task.Task(func(ctx orchContext.Context) (bool, error) {
			t.Log("Step 3: Another success")
			return true, nil
		}).Named("another-success-step"),
	).Named("collect-all-pipeline")

	cfg = config.Config{ErrorStrategy: errors.CollectAll}
	result, err = collectAllSeq.Execute(ctx, cfg)

	if err == nil {
		t.Error("Expected error from CollectAll strategy")
	} else {
		t.Logf("⚠️  CollectAll completed with errors")
		t.Logf("   - Success results: %v, %v", result.Get("success-step"), result.Get("another-success-step"))
		t.Logf("   - Total errors collected: %d", len(result.Errors()))
	}
}

// TestExample_RealWorldEcommerce demonstrates a realistic e-commerce order processing pipeline
func TestExample_RealWorldEcommerce(t *testing.T) {
	t.Log("=== Example: Real-World E-commerce Order Processing ===")

	// Simulate order data
	type Order struct {
		ID     string
		UserID string
		Items  []string
		Total  float64
		Status string
	}

	order := Order{
		ID:     "order-12345",
		UserID: "user-67890",
		Items:  []string{"laptop", "mouse", "keyboard"},
		Total:  1299.99,
		Status: "pending",
	}

	orderPipeline := Sequential(
		// Step 1: Validate order
		task.Task(func(ctx orchContext.Context) (Order, error) {
			t.Log("🔍 Validating order...")
			time.Sleep(10 * time.Millisecond) // Simulate validation time
			if order.Total <= 0 {
				return Order{}, stderrors.New("invalid order total")
			}
			order.Status = "validated"
			t.Logf("   Order %s validated successfully", order.ID)
			return order, nil
		}).Named("validate-order"),

		// Step 2: Check inventory
		task.Task(func(ctx orchContext.Context) (map[string]int, error) {
			t.Log("📦 Checking inventory...")
			time.Sleep(20 * time.Millisecond) // Simulate inventory check
			inventory := map[string]int{
				"laptop":   5,
				"mouse":    20,
				"keyboard": 15,
			}
			for _, item := range order.Items {
				if inventory[item] <= 0 {
					return nil, stderrors.New("item out of stock: " + item)
				}
			}
			t.Logf("   All items in stock")
			return inventory, nil
		}).Named("check-inventory"),

		// Step 3: Process payment
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("💳 Processing payment...")
			time.Sleep(30 * time.Millisecond) // Simulate payment processing
			if order.Total > 10000 {
				return "", stderrors.New("payment amount too high")
			}
			paymentID := "payment-" + order.ID
			t.Logf("   Payment processed: %s", paymentID)
			return paymentID, nil
		}).Named("process-payment"),

		// Step 4: Create shipment
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("📮 Creating shipment...")
			time.Sleep(25 * time.Millisecond) // Simulate shipment creation
			shipmentID := "shipment-" + order.ID
			t.Logf("   Shipment created: %s", shipmentID)
			return shipmentID, nil
		}).Named("create-shipment"),

		// Step 5: Send confirmation
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("📧 Sending confirmation...")
			time.Sleep(10 * time.Millisecond) // Simulate email sending
			confirmationID := "confirmation-" + order.ID
			t.Logf("   Confirmation sent: %s", confirmationID)
			return confirmationID, nil
		}).Named("send-confirmation"),
	).Named("ecommerce-order-pipeline").
		With(config.Config{
			Timeout:       10 * time.Second,
			ErrorStrategy: errors.FailFast, // Critical business process - fail fast
		})

	ctx := context.Background()
	startTime := time.Now()

	result, err := orderPipeline.Execute(ctx, config.Config{})

	duration := time.Since(startTime)

	if err != nil {
		t.Logf("❌ Order processing failed: %v", err)
		t.Logf("   Processing time: %v", duration)
		return
	}

	t.Logf("✅ Order processing completed successfully!")
	t.Logf("   Processing time: %v", duration)
	t.Logf("   Order: %+v", result.Get("validate-order"))
	t.Logf("   Payment ID: %v", result.Get("process-payment"))
	t.Logf("   Shipment ID: %v", result.Get("create-shipment"))
	t.Logf("   Confirmation ID: %v", result.Get("send-confirmation"))
}

// TestExample_NestedOrchestrations demonstrates nested sequential orchestrations
func TestExample_NestedOrchestrations(t *testing.T) {
	t.Log("=== Example: Nested Sequential Orchestrations ===")

	// Create sub-pipelines
	authPipeline := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("  Auth Step 1: Validate credentials")
			return "credentials-valid", nil
		}).Named("validate-credentials"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("  Auth Step 2: Generate token")
			return "token-abc123", nil
		}).Named("generate-token"),
	).Named("authentication-pipeline")

	dataPipeline := Sequential(
		task.Task(func(ctx orchContext.Context) ([]string, error) {
			t.Log("  Data Step 1: Fetch user data")
			return []string{"user1", "user2", "user3"}, nil
		}).Named("fetch-data"),

		task.Task(func(ctx orchContext.Context) (map[string]interface{}, error) {
			t.Log("  Data Step 2: Transform data")
			return map[string]interface{}{
				"users":     3,
				"processed": true,
			}, nil
		}).Named("transform-data"),
	).Named("data-processing-pipeline")

	// Create main pipeline with nested orchestrations
	mainPipeline := Sequential(
		authPipeline,
		dataPipeline,
	).Named("main-business-pipeline")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := mainPipeline.Execute(ctx, cfg)

	if err != nil {
		t.Fatalf("Nested orchestration example failed: %v", err)
	}

	t.Logf("✅ Nested pipeline completed successfully")
	t.Logf("   - Auth token: %v", result.Get("generate-token"))
	t.Logf("   - Data processed: %v", result.Get("transform-data"))
}

// TestExample_ContextCancellation demonstrates context cancellation handling
func TestExample_ContextCancellation(t *testing.T) {
	t.Log("=== Example: Context Cancellation ===")

	seq := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 1: Quick task")
			time.Sleep(10 * time.Millisecond)
			return "quick-done", nil
		}).Named("quick-task"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 2: Long task (will be cancelled)")
			time.Sleep(2 * time.Second) // This will be cancelled
			return "long-done", nil
		}).Named("long-task"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("Step 3: Should not execute")
			return "never-executed", nil
		}).Named("never-executed"),
	).Named("cancellation-demo")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	startTime := time.Now()
	result, err := seq.Execute(ctx, config.Config{})
	duration := time.Since(startTime)

	if err == nil {
		t.Error("Expected cancellation error")
	} else {
		t.Logf("⏰ Pipeline cancelled after %v", duration)
		t.Logf("   - Quick task completed: %v", result.Get("quick-task") != nil)
		t.Logf("   - Long task completed: %v", result.Get("long-task") != nil)
		t.Logf("   - Never executed: %v", result.Get("never-executed") != nil)
	}
}

// TestExample_DataPipeline demonstrates a data processing pipeline
func TestExample_DataPipeline(t *testing.T) {
	t.Log("=== Example: Data Processing Pipeline ===")

	type DataRecord struct {
		ID        int
		Name      string
		Value     float64
		Processed bool
	}

	// Sample data
	rawData := []DataRecord{
		{ID: 1, Name: "Record A", Value: 100.5, Processed: false},
		{ID: 2, Name: "Record B", Value: 200.7, Processed: false},
		{ID: 3, Name: "Record C", Value: 150.3, Processed: false},
	}

	dataPipeline := Sequential(
		// Step 1: Load data
		task.Task(func(ctx orchContext.Context) ([]DataRecord, error) {
			t.Log("📥 Loading raw data...")
			time.Sleep(20 * time.Millisecond)
			t.Logf("   Loaded %d records", len(rawData))
			return rawData, nil
		}).Named("load-data"),

		// Step 2: Validate data
		task.Task(func(ctx orchContext.Context) ([]DataRecord, error) {
			t.Log("✅ Validating data...")
			time.Sleep(30 * time.Millisecond)
			validRecords := make([]DataRecord, 0)
			for _, record := range rawData {
				if record.Value > 0 && record.Name != "" {
					validRecords = append(validRecords, record)
				}
			}
			t.Logf("   Validated %d/%d records", len(validRecords), len(rawData))
			return validRecords, nil
		}).Named("validate-data"),

		// Step 3: Transform data
		task.Task(func(ctx orchContext.Context) ([]DataRecord, error) {
			t.Log("🔄 Transforming data...")
			time.Sleep(40 * time.Millisecond)
			transformedRecords := make([]DataRecord, len(rawData))
			for i, record := range rawData {
				transformedRecords[i] = DataRecord{
					ID:        record.ID,
					Name:      strings.ToUpper(record.Name),
					Value:     record.Value * 1.1, // Apply 10% increase
					Processed: true,
				}
			}
			t.Logf("   Transformed %d records", len(transformedRecords))
			return transformedRecords, nil
		}).Named("transform-data"),

		// Step 4: Save results
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("💾 Saving results...")
			time.Sleep(15 * time.Millisecond)
			saveID := "save-" + time.Now().Format("20060102150405")
			t.Logf("   Results saved with ID: %s", saveID)
			return saveID, nil
		}).Named("save-results"),
	).Named("data-processing-pipeline").
		With(config.Config{
			Timeout:       10 * time.Second,
			ErrorStrategy: errors.FailFast,
		})

	ctx := context.Background()
	startTime := time.Now()

	result, err := dataPipeline.Execute(ctx, config.Config{})

	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Data pipeline failed: %v", err)
	}

	t.Logf("✅ Data pipeline completed successfully in %v", duration)
	t.Logf("   - Raw data: %d records", len(result.Get("load-data").([]DataRecord)))
	t.Logf("   - Validated data: %d records", len(result.Get("validate-data").([]DataRecord)))
	t.Logf("   - Transformed data: %d records", len(result.Get("transform-data").([]DataRecord)))
	t.Logf("   - Save ID: %v", result.Get("save-results"))
}

// TestExample_ErrorBoundaries demonstrates error boundary functionality
func TestExample_ErrorBoundaries(t *testing.T) {
	t.Log("=== Example: Error Boundaries ===")

	// Critical section that must succeed
	criticalSection := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("🔒 Critical: Authenticating...")
			return "auth-success", nil
		}).Named("authenticate"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("🔒 Critical: Validating permissions...")
			return "permissions-valid", nil
		}).Named("validate-permissions"),
	).Named("critical-section").
		ErrorBoundary(errors.FailFast) // Must succeed

	// Optional section that can have failures
	optionalSection := Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("📧 Optional: Sending welcome email...")
			// Simulate email service failure
			if time.Now().UnixNano()%3 == 0 {
				return "", stderrors.New("email service unavailable")
			}
			return "email-sent", nil
		}).Named("send-welcome-email"),

		task.Task(func(ctx orchContext.Context) (string, error) {
			t.Log("📊 Optional: Logging analytics...")
			return "analytics-logged", nil
		}).Named("log-analytics"),
	).Named("optional-section").
		ErrorBoundary(errors.CollectAll) // Continue despite failures

	// Main pipeline
	mainPipeline := Sequential(
		criticalSection,
		optionalSection,
	).Named("error-boundary-demo")

	ctx := context.Background()
	cfg := config.Config{ErrorStrategy: errors.FailFast}

	result, err := mainPipeline.Execute(ctx, cfg)

	if err != nil {
		t.Logf("⚠️  Pipeline completed with errors: %v", err)
	} else {
		t.Logf("✅ Pipeline completed successfully")
	}

	// Analyze results
	t.Logf("📊 Results Analysis:")
	t.Logf("   - Authentication: %v", result.Get("authenticate"))
	t.Logf("   - Permissions: %v", result.Get("validate-permissions"))
	t.Logf("   - Welcome email: %v", result.Get("send-welcome-email"))
	t.Logf("   - Analytics: %v", result.Get("log-analytics"))
	t.Logf("   - Total errors: %d", len(result.Errors()))
}
