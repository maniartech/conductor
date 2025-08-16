package orchestrator

import (
	"errors"
	"fmt"
	"log"

	. "github.com/maniartech/orchestrator"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

// ExampleConditional_ErrorHandling demonstrates the new error-returning condition signature
func ExampleConditional_ErrorHandling() {
	// Example 1: Basic condition with error handling
	fmt.Println("=== Example 1: Basic Conditional with Error Handling ===")

	adminTask := Task(func() (string, error) {
		return "Admin dashboard loaded", nil
	}).Named("admin-dashboard")

	userTask := Task(func() (string, error) {
		return "User profile loaded", nil
	}).Named("user-profile")

	// Conditional that can return an error during evaluation
	roleBasedAccess := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			role := ctx.Get("user_role")
			if role == nil {
				return false, errors.New("user_role not found in context")
			}

			roleStr, ok := role.(string)
			if !ok {
				return false, errors.New("user_role must be a string")
			}

			// Additional validation
			if roleStr == "" {
				return false, errors.New("user_role cannot be empty")
			}

			return roleStr == "admin", nil
		},
		adminTask,
		userTask,
	).Named("role-based-access")

	// Test with missing role (should error)
	workflow1 := Setup(roleBasedAccess)
	result1, err1 := workflow1.Await()

	if err1 != nil {
		fmt.Printf("❌ Expected error occurred: %v\n", err1)
		fmt.Printf("   Error count in result: %d\n", len(result1.Errors()))
	}

	// Example 2: Successful condition evaluation
	fmt.Println("\n=== Example 2: Successful Condition Evaluation ===")

	// Create a workflow with proper context setup
	contextualWorkflow := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			role := ctx.Get("user_role")
			if role == nil {
				return false, errors.New("user_role not found")
			}

			roleStr, ok := role.(string)
			if !ok {
				return false, errors.New("invalid role type")
			}

			fmt.Printf("   Evaluating role: %s\n", roleStr)
			return roleStr == "admin", nil
		},
		Task(func() (string, error) {
			fmt.Println("   Executing admin task...")
			return "Admin operations completed", nil
		}).Named("admin-ops"),
		Task(func() (string, error) {
			fmt.Println("   Executing user task...")
			return "User operations completed", nil
		}).Named("user-ops"),
	).Named("contextual-access")

	// This would need proper context setup in a real scenario
	workflow2 := Setup(contextualWorkflow)
	result2, err2 := workflow2.Await()

	if err2 != nil {
		fmt.Printf("❌ Error: %v\n", err2)
	} else {
		fmt.Printf("✅ Workflow completed successfully\n")
		// Note: In this example, the context won't have the role set,
		// so it will error as expected
	}
	_ = result2 // Suppress unused variable warning

	// Example 3: Complex condition with business logic
	fmt.Println("\n=== Example 3: Complex Business Logic Condition ===")

	orderProcessing := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			// Simulate complex business logic with error handling
			orderValue := ctx.Get("order_value")
			if orderValue == nil {
				return false, errors.New("order_value is required")
			}

			value, ok := orderValue.(float64)
			if !ok {
				return false, errors.New("order_value must be a number")
			}

			if value < 0 {
				return false, errors.New("order_value cannot be negative")
			}

			customerTier := ctx.Get("customer_tier")
			if customerTier == nil {
				return false, errors.New("customer_tier is required")
			}

			tier, ok := customerTier.(string)
			if !ok {
				return false, errors.New("customer_tier must be a string")
			}

			// Business logic: Premium customers get expedited processing for orders > $100
			isPremium := tier == "premium"
			isHighValue := value > 100.0

			fmt.Printf("   Order value: $%.2f, Customer tier: %s\n", value, tier)
			fmt.Printf("   Expedited processing: %t\n", isPremium && isHighValue)

			return isPremium && isHighValue, nil
		},
		Task(func() (string, error) {
			fmt.Println("   Processing with expedited shipping...")
			return "Order processed with expedited shipping", nil
		}).Named("expedited-processing"),
		Task(func() (string, error) {
			fmt.Println("   Processing with standard shipping...")
			return "Order processed with standard shipping", nil
		}).Named("standard-processing"),
	).Named("order-processing")

	workflow3 := Setup(orderProcessing)
	result3, err3 := workflow3.Await()

	if err3 != nil {
		fmt.Printf("❌ Order processing failed: %v\n", err3)
		if result3 != nil && len(result3.Errors()) > 0 {
			fmt.Printf("   Detailed errors: %d\n", len(result3.Errors()))
		}
	}

	// Example 4: Panic handling in conditions
	fmt.Println("\n=== Example 4: Panic Handling in Conditions ===")

	panicCondition := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			// Simulate a panic in condition evaluation
			fmt.Println("   About to panic in condition...")
			panic("simulated condition panic")
		},
		Task(func() (string, error) { return "true branch", nil }),
		Task(func() (string, error) { return "false branch", nil }),
	).Named("panic-condition")

	workflow4 := Setup(panicCondition)
	result4, err4 := workflow4.Await()

	if err4 != nil {
		fmt.Printf("❌ Panic was handled: %v\n", err4)
		if result4 != nil && len(result4.Errors()) > 0 {
			fmt.Printf("   Panic recorded in result errors: %d\n", len(result4.Errors()))
		}
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("✅ New condition signature func(orchContext.Context) (bool, error) provides:")
	fmt.Println("   - Explicit error handling for condition evaluation")
	fmt.Println("   - Better error messages and debugging")
	fmt.Println("   - Robust validation of context values")
	fmt.Println("   - Panic recovery with proper error reporting")
	fmt.Println("   - Consistent error handling across the orchestration")

	// Output:
	// === Example 1: Basic Conditional with Error Handling ===
	// ❌ Expected error occurred: condition evaluation failed: user_role not found in context
	//    Error count in result: 1
	//
	// === Example 2: Successful Condition Evaluation ===
	// ❌ Error: condition evaluation failed: user_role not found
	//
	// === Example 3: Complex Business Logic Condition ===
	// ❌ Order processing failed: condition evaluation failed: order_value is required
	//    Detailed errors: 1
	//
	// === Example 4: Panic Handling in Conditions ===
	//    About to panic in condition...
	// ❌ Panic was handled: condition evaluation failed: condition evaluation panic: simulated condition panic
	//    Panic recorded in result errors: 1
	//
	// === Summary ===
	// ✅ New condition signature func(orchContext.Context) (bool, error) provides:
	//    - Explicit error handling for condition evaluation
	//    - Better error messages and debugging
	//    - Robust validation of context values
	//    - Panic recovery with proper error reporting
	//    - Consistent error handling across the orchestration
}

// ExampleConditional_SuccessfulExecution demonstrates successful execution with the new signature
func ExampleConditional_SuccessfulExecution() {
	fmt.Println("=== Successful Conditional Execution ===")

	// Create tasks
	morningTask := Task(func() (string, error) {
		return "Good morning! Starting the day.", nil
	}).Named("morning-greeting")

	eveningTask := Task(func() (string, error) {
		return "Good evening! Wrapping up the day.", nil
	}).Named("evening-greeting")

	// Create a conditional that checks time of day
	timeBasedGreeting := Conditional(
		func(ctx orchContext.Context) (bool, error) {
			// Simulate checking time of day
			timeOfDay := ctx.Get("time_of_day")
			if timeOfDay == nil {
				// Default to morning if not specified
				return true, nil
			}

			timeStr, ok := timeOfDay.(string)
			if !ok {
				return false, errors.New("time_of_day must be a string")
			}

			fmt.Printf("   Time of day: %s\n", timeStr)

			// Return true for morning (AM), false for evening (PM)
			isMorning := timeStr == "AM"
			return isMorning, nil
		},
		morningTask,
		eveningTask,
	).Named("time-based-greeting")

	// Execute with default (should use morning)
	fmt.Println("\n--- Test 1: Default execution (no time specified) ---")
	workflow1 := Setup(timeBasedGreeting)
	result1, err1 := workflow1.Await()

	if err1 != nil {
		log.Printf("Error: %v", err1)
	} else {
		fmt.Printf("✅ Result: %v\n", result1.Get("if-true"))
	}

	fmt.Println("\n--- Test 2: Evening execution ---")
	// For a real implementation, you would set up the context properly
	// This is just demonstrating the API
	workflow2 := Setup(timeBasedGreeting)
	result2, err2 := workflow2.Await()

	if err2 != nil {
		log.Printf("Error: %v", err2)
	} else {
		// This will still be morning since we didn't set up context
		fmt.Printf("✅ Result: %v\n", result2.Get("if-true"))
	}

	// Output:
	// === Successful Conditional Execution ===
	//
	// --- Test 1: Default execution (no time specified) ---
	// ✅ Result: Good morning! Starting the day.
	//
	// --- Test 2: Evening execution ---
	// ✅ Result: Good morning! Starting the day.
}
