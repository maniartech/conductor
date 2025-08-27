package orchestrator_test

import (
	"fmt"
	"time"

	"github.com/maniartech/orchestrator"
)

// Example functions for demonstration
func keepInfraReady() error {
	time.Sleep(10 * time.Millisecond)
	return nil
}

func fetchResource(resourceId int) error {
	time.Sleep(20 * time.Millisecond)
	return nil
}

func processResource() error {
	time.Sleep(30 * time.Millisecond)
	return nil
}

func submitResource() error {
	time.Sleep(15 * time.Millisecond)
	return nil
}

func prepareDependencyA() error {
	time.Sleep(25 * time.Millisecond)
	return nil
}

func prepareDependencyB() error {
	time.Sleep(20 * time.Millisecond)
	return nil
}

func prepareDependencyC() error {
	time.Sleep(35 * time.Millisecond)
	return nil
}

func postToSocialMedia() error {
	time.Sleep(40 * time.Millisecond)
	return nil
}

func sendNotifications() error {
	time.Sleep(30 * time.Millisecond)
	return nil
}

func submitReport() error {
	time.Sleep(25 * time.Millisecond)
	return nil
}

// Example_simpleTask demonstrates basic task execution
func Example_simpleTask() {
	result, err := orchestrator.Setup(
		orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
			return "Hello, World!", nil
		}).Named("greeting"),
	).Await()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	greeting := result.Get("greeting").(string)
	fmt.Println(greeting)

	// Output:
	// Hello, World!
}

// Example_taskWithConfiguration demonstrates task configuration
func Example_taskWithConfiguration() {
	result, err := orchestrator.Setup(
		orchestrator.Task(func(ctx orchestrator.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 42, nil
		}).Named("answer").
			With(orchestrator.Config{
				Timeout: 30 * time.Second,
			}),
	).Await()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	answer := result.Get("answer").(int)
	fmt.Printf("The answer is: %d\n", answer)

	// Output:
	// The answer is: 42
}

// Example_statusMonitoring demonstrates real-time status monitoring
func Example_statusMonitoring() {
	workflow := orchestrator.Setup(
		orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "completed", nil
		}).Named("slow-task"),
	)

	// Check initial status
	fmt.Printf("Initial status: %s\n", workflow.GetStatus())

	// Start execution in background
	go func() {
		workflow.Await()
	}()

	// Monitor status changes
	time.Sleep(10 * time.Millisecond)
	fmt.Printf("Status during execution: %s\n", workflow.GetStatus())

	time.Sleep(60 * time.Millisecond)
	fmt.Printf("Final status: %s\n", workflow.GetStatus())

	// Output:
	// Initial status: NotStarted
	// Status during execution: Running
	// Final status: Completed
}

// NOTE: The following examples will work once Sequential and Concurrent are implemented in tasks 4.1 and 5.1

// Example_complexOrchestration demonstrates the complex orchestration from the README
// This will be uncommented once Sequential and Concurrent are implemented
/*
func Example_complexOrchestration() {
	resourceId := 123

	result, err := orchestrator.Setup(
		orchestrator.Sequential(
			// Infrastructure preparation
			orchestrator.Task(keepInfraReady).Named("infra-ready"),

			// Concurrent resource processing
			orchestrator.Concurrent(
				// Main resource processing pipeline
				orchestrator.Sequential(
					orchestrator.Task(func(ctx orchestrator.Context) error { return fetchResource(resourceId) }).Named("fetch-resource"),
					orchestrator.Task(processResource).Named("process-resource"),
					orchestrator.Task(submitResource).Named("submit-resource"),
				).Named("resource-pipeline"),

				// Dependency preparation (concurrent)
				orchestrator.Concurrent(
					orchestrator.Task(prepareDependencyA).Named("prep-dep-a"),
					orchestrator.Task(prepareDependencyB).Named("prep-dep-b"),
					orchestrator.Task(prepareDependencyC).Named("prep-dep-c"),
				).Named("dependency-prep"),
			).Named("main-processing"),

			// Final notifications (concurrent)
			orchestrator.Concurrent(
				orchestrator.Task(postToSocialMedia).Named("social-media"),
				orchestrator.Task(sendNotifications).Named("notifications"),
				orchestrator.Task(submitReport).Named("report"),
			).Named("notifications"),
		).Named("resource-handler"),
	).Await()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Resource processing completed successfully!")

	// Output:
	// Resource processing completed successfully!
}
*/
