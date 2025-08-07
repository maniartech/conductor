package sequential

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/internal/result"
	"github.com/maniartech/orchestrator/internal/task"
)

// =============================================================================
// EXAMPLE TESTS - From Simple to Kitchen Sink
// =============================================================================

// ExampleSequential_Simple demonstrates the most basic sequential orchestration
func ExampleSequential_Simple(t *testing.T) {
	t.Log("=== Example: Simple Sequential Orchestration ===")

	// Create a simple 3-step sequential process
	seq := Sequential(
		task.Task(func() (string, error) {
			t.Log("Step 1: Fetching user data...")
			return "user-123", nil
		}).Named("fetch-user"),

		task.Task(func() (string, error) {
			t.Log("Step 2: Validating user...")
			return "validated", nil
		}).Named("validate-user"),

		task.Task(func() (string, error) {
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

// ExampleSequential_WithConfiguration demonstrates configuration usage
func ExampleSequential_WithConfiguration(t *testing.T) {
	t.Log("=== Example: Sequential with Configuration ===")

	seq := Sequential(
		task.Task(func() (string, error) {
			t.Log("Step 1: Quick operation")
			time.Sleep(10 * time.Millisecond)
			return "quick-result", nil
		}).Named("quick-task"),

		task.Task(func() (int, error) {
			t.Log("Step 2: Medium operation")
			time.Sleep(50 * time.Millisecond)
			return 42, nil
		}).Named("medium-task"),
	).Named("configured-pipeline").
		With(config.Config{
			Timeout:       5 * time.Second,
			ErrorStrategy: errors.FailFast,
		})

	ctx := context.Background()
	result, err := seq.Execute(ctx, config.Config{})

	if err != nil {
		t.Fatalf("Configuration example failed: %v", err)
	}

	t.Logf("✅ Configured pipeline completed in time")
	t.Logf("   - Quick result: %v", result.Get("quick-task"))
	t.Logf("   - Medium result: %v", result.Get("medium-task"))
}

// ExampleSequential_ErrorHandling demonstrates different error strategies
func ExampleSequential_ErrorHandling(t *testing.T) {
	t.Log("=== Example: Error Handling Strategies ===")

	// Test FailFast strategy
	t.Log("--- Testing FailFast Strategy ---")
	failFastSeq := Sequential(
		task.Task(func() (string, error) {
			t.Log("Step 1: Success")
			return "success", nil
		}).Named("success-step"),

		task.Task(func() (int, error) {
			t.Log("Step 2: Failure")
			return 0, stderrors.New("intentional failure")
		}).Named("failure-step"),

		task.Task(func() (bool, error) {
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
		t.Logf("❌ FailFast stopped at first error: %v", err)
		t.Logf("   - Success step result: %v", result.Get("success-step"))
		t.Logf("   - Skipped step result: %v", result.Get("skipped-step")) // Should be nil
	}

	// Test CollectAll strategy
	t.Log("--- Testing CollectAll Strategy ---")
	collectAllSeq := Sequential(
		task.Task(func() (string, error) {
			t.Log("Step 1: Success")
			return "success", nil
		}).Named("success-step"),

		task.Task(func() (int, error) {
			t.Log("Step 2: Failure")
			return 0, stderrors.New("intentional failure")
		}).Named("failure-step"),

		task.Task(func() (bool, error) {
			t.Log("Step 3: Another success")
			return true, nil
		}).Named("another-success-step"),

		task.Task(func() (string, error) {
			t.Log("Step 4: Another failure")
			return "", stderrors.New("another failure")
		}).Named("another-failure-step"),
	).Named("collect-all-pipeline")

	cfg = config.Config{ErrorStrategy: errors.CollectAll}
	result, err = collectAllSeq.Execute(ctx, cfg)

	if err == nil {
		t.Error("Expected error from CollectAll strategy")
	} else {
		t.Logf("⚠️  CollectAll completed with errors: %v", err)
		t.Logf("   - Success results: %v, %v", result.Get("success-step"), result.Get("another-success-step"))
		t.Logf("   - Total errors collected: %d", len(result.Errors()))
	}
}

// ExampleSequential_NestedOrchestrations demonstrates nested sequential orchestrations
func ExampleSequential_NestedOrchestrations(t *testing.T) {
	t.Log("=== Example: Nested Sequential Orchestrations ===")

	// Create sub-pipelines
	authPipeline := Sequential(
		task.Task(func() (string, error) {
			t.Log("  Auth Step 1: Validate credentials")
			return "credentials-valid", nil
		}).Named("validate-credentials"),

		task.Task(func() (string, error) {
			t.Log("  Auth Step 2: Generate token")
			return "token-abc123", nil
		}).Named("generate-token"),
	).Named("authentication-pipeline")

	dataPipeline := Sequential(
		task.Task(func() ([]string, error) {
			t.Log("  Data Step 1: Fetch user data")
			return []string{"user1", "user2", "user3"}, nil
		}).Named("fetch-data"),

		task.Task(func() (map[string]interface{}, error) {
			t.Log("  Data Step 2: Transform data")
			return map[string]interface{}{
				"users":     3,
				"processed": true,
			}, nil
		}).Named("transform-data"),
	).Named("data-processing-pipeline")

	notificationPipeline := Sequential(
		task.Task(func() (string, error) {
			t.Log("  Notification Step 1: Send email")
			return "email-sent", nil
		}).Named("send-email"),

		task.Task(func() (string, error) {
			t.Log("  Notification Step 2: Log activity")
			return "activity-logged", nil
		}).Named("log-activity"),
	).Named("notification-pipeline")

	// Create main pipeline with nested orchestrations
	mainPipeline := Sequential(
		authPipeline,
		dataPipeline,
		notificationPipeline,
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
	t.Logf("   - Notifications: %v, %v", result.Get("send-email"), result.Get("log-activity"))
}

// ExampleSequential_RealWorldEcommerce demonstrates a realistic e-commerce order processing pipeline
func ExampleSequential_RealWorldEcommerce(t *testing.T) {
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
		task.Task(func() (Order, error) {
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
		task.Task(func() (map[string]int, error) {
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
		task.Task(func() (string, error) {
			t.Log("💳 Processing payment...")
			time.Sleep(30 * time.Millisecond) // Simulate payment processing
			if order.Total > 10000 {
				return "", stderrors.New("payment amount too high")
			}
			paymentID := "payment-" + order.ID
			t.Logf("   Payment processed: %s", paymentID)
			return paymentID, nil
		}).Named("process-payment"),

		// Step 4: Reserve inventory
		task.Task(func() ([]string, error) {
			t.Log("🔒 Reserving inventory...")
			time.Sleep(15 * time.Millisecond) // Simulate reservation
			reservations := make([]string, len(order.Items))
			for i, item := range order.Items {
				reservations[i] = "reservation-" + item + "-" + order.ID
			}
			t.Logf("   Reserved %d items", len(reservations))
			return reservations, nil
		}).Named("reserve-inventory"),

		// Step 5: Create shipment
		task.Task(func() (string, error) {
			t.Log("📮 Creating shipment...")
			time.Sleep(25 * time.Millisecond) // Simulate shipment creation
			shipmentID := "shipment-" + order.ID
			t.Logf("   Shipment created: %s", shipmentID)
			return shipmentID, nil
		}).Named("create-shipment"),

		// Step 6: Send confirmation
		task.Task(func() (string, error) {
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

		// Check what steps completed before failure
		if result.Get("validate-order") != nil {
			t.Log("   ✅ Order validation completed")
		}
		if result.Get("check-inventory") != nil {
			t.Log("   ✅ Inventory check completed")
		}
		if result.Get("process-payment") != nil {
			t.Log("   ✅ Payment processing completed")
		}
		// ... etc for other steps

		return
	}

	t.Logf("✅ Order processing completed successfully!")
	t.Logf("   Processing time: %v", duration)
	t.Logf("   Order: %+v", result.Get("validate-order"))
	t.Logf("   Payment ID: %v", result.Get("process-payment"))
	t.Logf("   Shipment ID: %v", result.Get("create-shipment"))
	t.Logf("   Confirmation ID: %v", result.Get("send-confirmation"))
}

// ExampleSequential_ContextCancellation demonstrates context cancellation handling
func ExampleSequential_ContextCancellation(t *testing.T) {
	t.Log("=== Example: Context Cancellation ===")

	seq := Sequential(
		task.Task(func() (string, error) {
			t.Log("Step 1: Quick task")
			time.Sleep(10 * time.Millisecond)
			return "quick-done", nil
		}).Named("quick-task"),

		task.Task(func() (string, error) {
			t.Log("Step 2: Long task (will be cancelled)")
			time.Sleep(2 * time.Second) // This will be cancelled
			return "long-done", nil
		}).Named("long-task"),

		task.Task(func() (string, error) {
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
		t.Logf("⏰ Pipeline cancelled after %v: %v", duration, err)
		t.Logf("   - Quick task completed: %v", result.Get("quick-task") != nil)
		t.Logf("   - Long task completed: %v", result.Get("long-task") != nil)
		t.Logf("   - Never executed: %v", result.Get("never-executed") != nil)
	}
}

// ExampleSequential_ErrorBoundaries demonstrates error boundary functionality
func ExampleSequential_ErrorBoundaries(t *testing.T) {
	t.Log("=== Example: Error Boundaries ===")

	// Critical section that must succeed
	criticalSection := Sequential(
		task.Task(func() (string, error) {
			t.Log("🔒 Critical: Authenticating...")
			return "auth-success", nil
		}).Named("authenticate"),

		task.Task(func() (string, error) {
			t.Log("🔒 Critical: Validating permissions...")
			return "permissions-valid", nil
		}).Named("validate-permissions"),
	).Named("critical-section").
		ErrorBoundary(errors.FailFast) // Must succeed

	// Optional section that can have failures
	optionalSection := Sequential(
		task.Task(func() (string, error) {
			t.Log("📧 Optional: Sending welcome email...")
			// Simulate email service failure
			if time.Now().UnixNano()%3 == 0 {
				return "", stderrors.New("email service unavailable")
			}
			return "email-sent", nil
		}).Named("send-welcome-email"),

		task.Task(func() (string, error) {
			t.Log("📊 Optional: Logging analytics...")
			return "analytics-logged", nil
		}).Named("log-analytics"),

		task.Task(func() (string, error) {
			t.Log("🔔 Optional: Sending push notification...")
			// Simulate push service failure
			if time.Now().UnixNano()%4 == 0 {
				return "", stderrors.New("push service unavailable")
			}
			return "push-sent", nil
		}).Named("send-push-notification"),
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
	t.Logf("   - Push notification: %v", result.Get("send-push-notification"))
	t.Logf("   - Total errors: %d", len(result.Errors()))
}

// ExampleSequential_DataPipeline demonstrates a data processing pipeline
func ExampleSequential_DataPipeline(t *testing.T) {
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
		task.Task(func() ([]DataRecord, error) {
			t.Log("📥 Loading raw data...")
			time.Sleep(20 * time.Millisecond)
			t.Logf("   Loaded %d records", len(rawData))
			return rawData, nil
		}).Named("load-data"),

		// Step 2: Validate data
		task.Task(func() ([]DataRecord, error) {
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
		task.Task(func() ([]DataRecord, error) {
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

		// Step 4: Aggregate results
		task.Task(func() (map[string]interface{}, error) {
			t.Log("📊 Aggregating results...")
			time.Sleep(25 * time.Millisecond)
			
			totalValue := 0.0
			for _, record := range rawData {
				totalValue += record.Value * 1.1 // Use transformed values
			}
			
			aggregation := map[string]interface{}{
				"total_records": len(rawData),
				"total_value":   totalValue,
				"average_value": totalValue / float64(len(rawData)),
				"processed_at":  time.Now(),
			}
			
			t.Logf("   Aggregated %d records, total value: %.2f", len(rawData), totalValue)
			return aggregation, nil
		}).Named("aggregate-data"),

		// Step 5: Save results
		task.Task(func() (string, error) {
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
	t.Logf("   - Aggregation: %+v", result.Get("aggregate-data"))
	t.Logf("   - Save ID: %v", result.Get("save-results"))
}

// Helper functions for examples
func countNonNilResults(result *result.Result) int {
	count := 0
	stepNames := []string{
		"authenticate-user", "load-permissions", "generate-session",
		"fetch-datasets", "process-dataset-a", "process-dataset-b", "process-dataset-c", "generate-analytics",
		"send-email", "send-sms", "send-push", "log-notifications",
		"clean-temp-files", "clear-cache", "update-metrics",
	}

	for _, name := range stepNames {
		if result.Get(name) != nil {
			count++
		}
	}
	return count
}

func calculateSuccessRate(result *result.Result) float64 {
	totalSteps := 15 // Total number of steps in kitchen sink example
	errors := len(result.Errors())
	successfulSteps := totalSteps - errors
	return (float64(successfulSteps) / float64(totalSteps)) * 100
}// Exampl
eSequential_KitchenSink demonstrates ALL features in one comprehensive example
func ExampleSequential_KitchenSink(t *testing.T) {
	t.Log("=== KITCHEN SINK EXAMPLE: All Sequential Features ===")

	// Complex data structures
	type UserProfile struct {
		ID       string
		Name     string
		Email    string
		Verified bool
	}

	type ProcessingResult struct {
		Success   bool
		Message   string
		Timestamp time.Time
		Data      map[string]interface{}
	}

	// Create multiple nested pipelines with different error strategies

	// Critical authentication pipeline (FailFast)
	authPipeline := Sequential(
		task.Task(func() (UserProfile, error) {
			t.Log("🔐 [AUTH] Authenticating user...")
			time.Sleep(20 * time.Millisecond)
			// Simulate occasional auth failure
			if time.Now().UnixNano()%7 == 0 {
				return UserProfile{}, stderrors.New("authentication failed")
			}
			return UserProfile{
				ID:       "user-12345",
				Name:     "John Doe",
				Email:    "john@example.com",
				Verified: true,
			}, nil
		}).Named("authenticate-user").
			With(config.Config{Timeout: 1 * time.Second}),

		task.Task(func() ([]string, error) {
			t.Log("🔑 [AUTH] Loading user permissions...")
			time.Sleep(15 * time.Millisecond)
			return []string{"read", "write", "admin"}, nil
		}).Named("load-permissions"),

		task.Task(func() (string, error) {
			t.Log("🎫 [AUTH] Generating session token...")
			time.Sleep(10 * time.Millisecond)
			return "session-token-abc123xyz", nil
		}).Named("generate-session"),
	).Named("authentication-pipeline").
		ErrorBoundary(errors.FailFast). // Critical - must succeed
		With(config.Config{
			Timeout: 2 * time.Second,
		})

	// Data processing pipeline (CollectAll - try to process as much as possible)
	dataPipeline := Sequential(
		task.Task(func() ([]map[string]interface{}, error) {
			t.Log("📊 [DATA] Fetching user data...")
			time.Sleep(30 * time.Millisecond)
			return []map[string]interface{}{
				{"id": 1, "name": "Dataset A", "size": 1024},
				{"id": 2, "name": "Dataset B", "size": 2048},
				{"id": 3, "name": "Dataset C", "size": 512},
			}, nil
		}).Named("fetch-datasets"),

		task.Task(func() (ProcessingResult, error) {
			t.Log("⚙️  [DATA] Processing dataset A...")
			time.Sleep(25 * time.Millisecond)
			// Simulate occasional processing failure
			if time.Now().UnixNano()%5 == 0 {
				return ProcessingResult{}, stderrors.New("dataset A processing failed")
			}
			return ProcessingResult{
				Success:   true,
				Message:   "Dataset A processed successfully",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"processed_records": 1000},
			}, nil
		}).Named("process-dataset-a"),

		task.Task(func() (ProcessingResult, error) {
			t.Log("⚙️  [DATA] Processing dataset B...")
			time.Sleep(35 * time.Millisecond)
			return ProcessingResult{
				Success:   true,
				Message:   "Dataset B processed successfully",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"processed_records": 2000},
			}, nil
		}).Named("process-dataset-b"),

		task.Task(func() (ProcessingResult, error) {
			t.Log("⚙️  [DATA] Processing dataset C...")
			time.Sleep(20 * time.Millisecond)
			// Simulate another potential failure
			if time.Now().UnixNano()%6 == 0 {
				return ProcessingResult{}, stderrors.New("dataset C processing failed")
			}
			return ProcessingResult{
				Success:   true,
				Message:   "Dataset C processed successfully",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"processed_records": 500},
			}, nil
		}).Named("process-dataset-c"),

		task.Task(func() (map[string]interface{}, error) {
			t.Log("📈 [DATA] Generating analytics...")
			time.Sleep(40 * time.Millisecond)
			return map[string]interface{}{
				"total_records":   3500,
				"processing_time": "120ms",
				"success_rate":    0.95,
				"generated_at":    time.Now(),
			}, nil
		}).Named("generate-analytics"),
	).Named("data-processing-pipeline").
		ErrorBoundary(errors.CollectAll). // Try to process as much as possible
		With(config.Config{
			Timeout: 5 * time.Second,
		})

	// Notification pipeline (CollectAll - send as many notifications as possible)
	notificationPipeline := Sequential(
		task.Task(func() (string, error) {
			t.Log("📧 [NOTIFY] Sending email notification...")
			time.Sleep(15 * time.Millisecond)
			// Simulate email service issues
			if time.Now().UnixNano()%8 == 0 {
				return "", stderrors.New("email service unavailable")
			}
			return "email-sent-id-12345", nil
		}).Named("send-email"),

		task.Task(func() (string, error) {
			t.Log("📱 [NOTIFY] Sending SMS notification...")
			time.Sleep(20 * time.Millisecond)
			return "sms-sent-id-67890", nil
		}).Named("send-sms"),

		task.Task(func() (string, error) {
			t.Log("🔔 [NOTIFY] Sending push notification...")
			time.Sleep(10 * time.Millisecond)
			// Simulate push service issues
			if time.Now().UnixNano()%9 == 0 {
				return "", stderrors.New("push service unavailable")
			}
			return "push-sent-id-abcdef", nil
		}).Named("send-push"),

		task.Task(func() (string, error) {
			t.Log("📝 [NOTIFY] Logging notification activity...")
			time.Sleep(5 * time.Millisecond)
			return "activity-logged-" + time.Now().Format("20060102150405"), nil
		}).Named("log-notifications"),
	).Named("notification-pipeline").
		ErrorBoundary(errors.CollectAll). // Send as many notifications as possible
		With(config.Config{
			Timeout: 3 * time.Second,
		})

	// Cleanup pipeline (FailFast - cleanup is critical)
	cleanupPipeline := Sequential(
		task.Task(func() (string, error) {
			t.Log("🧹 [CLEANUP] Cleaning temporary files...")
			time.Sleep(10 * time.Millisecond)
			return "temp-files-cleaned", nil
		}).Named("clean-temp-files"),

		task.Task(func() (string, error) {
			t.Log("🗑️  [CLEANUP] Clearing cache...")
			time.Sleep(8 * time.Millisecond)
			return "cache-cleared", nil
		}).Named("clear-cache"),

		task.Task(func() (string, error) {
			t.Log("📊 [CLEANUP] Updating metrics...")
			time.Sleep(12 * time.Millisecond)
			return "metrics-updated", nil
		}).Named("update-metrics"),
	).Named("cleanup-pipeline").
		ErrorBoundary(errors.FailFast)

	// MASTER PIPELINE - Orchestrates everything
	masterPipeline := Sequential(
		authPipeline,         // Must succeed (FailFast)
		dataPipeline,         // Process as much as possible (CollectAll)
		notificationPipeline, // Send as many notifications as possible (CollectAll)
		cleanupPipeline,      // Must succeed (FailFast)
	).Named("kitchen-sink-master-pipeline").
		With(config.Config{
			Timeout:       30 * time.Second,
			ErrorStrategy: errors.CollectAll, // Master level - try to complete as much as possible
		})

	// Execute with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	t.Log("🚀 Starting Kitchen Sink Pipeline...")
	startTime := time.Now()

	result, err := masterPipeline.Execute(ctx, config.Config{})

	duration := time.Since(startTime)
	t.Logf("⏱️  Total execution time: %v", duration)

	// Comprehensive result analysis
	if err != nil {
		t.Logf("⚠️  Pipeline completed with errors: %v", err)

		// Analyze errors by pipeline
		errors := result.Errors()
		authErrors := 0
		dataErrors := 0
		notifyErrors := 0
		cleanupErrors := 0

		for _, opErr := range errors {
			switch {
			case strings.Contains(opErr.OpID, "authentication"):
				authErrors++
			case strings.Contains(opErr.OpID, "data-processing"):
				dataErrors++
			case strings.Contains(opErr.OpID, "notification"):
				notifyErrors++
			case strings.Contains(opErr.OpID, "cleanup"):
				cleanupErrors++
			}
		}

		t.Logf("📊 Error Summary:")
		t.Logf("   - Authentication errors: %d", authErrors)
		t.Logf("   - Data processing errors: %d", dataErrors)
		t.Logf("   - Notification errors: %d", notifyErrors)
		t.Logf("   - Cleanup errors: %d", cleanupErrors)
	} else {
		t.Logf("✅ Kitchen Sink Pipeline completed successfully!")
	}

	// Detailed result analysis
	t.Logf("📋 Detailed Results:")

	// Authentication results
	if userProfile := result.Get("authenticate-user"); userProfile != nil {
		t.Logf("   🔐 User authenticated: %+v", userProfile)
	}
	if permissions := result.Get("load-permissions"); permissions != nil {
		t.Logf("   🔑 Permissions loaded: %v", permissions)
	}
	if session := result.Get("generate-session"); session != nil {
		t.Logf("   🎫 Session token: %v", session)
	}

	// Data processing results
	if datasets := result.Get("fetch-datasets"); datasets != nil {
		t.Logf("   📊 Datasets fetched: %d items", len(datasets.([]map[string]interface{})))
	}
	if analytics := result.Get("generate-analytics"); analytics != nil {
		t.Logf("   📈 Analytics: %+v", analytics)
	}

	// Notification results
	notificationsSent := 0
	if result.Get("send-email") != nil {
		notificationsSent++
		t.Logf("   📧 Email sent: %v", result.Get("send-email"))
	}
	if result.Get("send-sms") != nil {
		notificationsSent++
		t.Logf("   📱 SMS sent: %v", result.Get("send-sms"))
	}
	if result.Get("send-push") != nil {
		notificationsSent++
		t.Logf("   🔔 Push sent: %v", result.Get("send-push"))
	}
	t.Logf("   📬 Total notifications sent: %d/3", notificationsSent)

	// Cleanup results
	if result.Get("clean-temp-files") != nil {
		t.Logf("   🧹 Temp files cleaned: %v", result.Get("clean-temp-files"))
	}
	if result.Get("clear-cache") != nil {
		t.Logf("   🗑️  Cache cleared: %v", result.Get("clear-cache"))
	}
	if result.Get("update-metrics") != nil {
		t.Logf("   📊 Metrics updated: %v", result.Get("update-metrics"))
	}

	// Performance metrics
	t.Logf("🎯 Performance Metrics:")
	t.Logf("   - Total steps executed: %d", countNonNilResults(result))
	t.Logf("   - Total errors: %d", len(result.Errors()))
	t.Logf("   - Success rate: %.2f%%", calculateSuccessRate(result))
	t.Logf("   - Average step time: %v", duration/time.Duration(countNonNilResults(result)))

	t.Log("🏁 Kitchen Sink Example completed!")
}

// ExampleSequential_PerformanceBenchmark demonstrates performance characteristics
func ExampleSequential_PerformanceBenchmark(t *testing.T) {
	t.Log("=== Example: Performance Benchmark ===")

	// Create a pipeline with many small tasks
	tasks := make([]orchestration.Orchestration, 100)
	for i := 0; i < 100; i++ {
		taskIndex := i // Capture loop variable
		tasks[i] = task.Task(func() (int, error) {
			// Simulate very light work
			time.Sleep(time.Microsecond * 10)
			return taskIndex, nil
		}).Named(fmt.Sprintf("task-%d", taskIndex))
	}

	performancePipeline := Sequential(tasks...).
		Named("performance-benchmark").
		With(config.Config{
			ErrorStrategy: errors.FailFast,
			Timeout:       10 * time.Second,
		})

	ctx := context.Background()
	startTime := time.Now()

	result, err := performancePipeline.Execute(ctx, config.Config{})

	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Performance benchmark failed: %v", err)
	}

	t.Logf("⚡ Performance Benchmark Results:")
	t.Logf("   - Total tasks: 100")
	t.Logf("   - Total execution time: %v", duration)
	t.Logf("   - Average time per task: %v", duration/100)
	t.Logf("   - Tasks per second: %.0f", float64(100)/duration.Seconds())
	t.Logf("   - Memory allocations: minimal (atomic operations)")
	t.Logf("   - All results collected: %d", countCompletedTasks(result, 100))
}

// Helper function for performance benchmark
func countCompletedTasks(result *result.Result, totalTasks int) int {
	count := 0
	for i := 0; i < totalTasks; i++ {
		if result.Get(fmt.Sprintf("task-%d", i)) != nil {
			count++
		}
	}
	return count
}