package main

import (
	"fmt"
	"log"
	"time"

	"github.com/maniartech/orchestrator"
)

func main() {
	fmt.Println(" Orchestrator - Concurrent Tasks Example")
	fmt.Println()

	start := time.Now()

	// Execute multiple tasks concurrently
	workflow := orchestrator.Setup(
		orchestrator.Concurrent(
			orchestrator.Task(fetchUserData).Named("fetch-user"),
			orchestrator.Task(processAnalytics).Named("process-analytics"),
			orchestrator.Task(sendNotifications).Named("send-notifications"),
		).Named("concurrent-tasks"),
	)

	result, err := workflow.Await() // Executes and awaits
	duration := time.Since(start)

	if err != nil {
		log.Printf(" Error: %v", err)
		return
	}

	fmt.Printf(" All tasks completed concurrently in %v\n\n", duration)

	// Display results
	fmt.Println(" Results:")
	fmt.Printf("   User Data: %s\n", result.Get("fetch-user").(string))
	fmt.Printf("   Analytics: %s\n", result.Get("process-analytics").(string))
	fmt.Printf("   Notifications: %s\n", result.Get("send-notifications").(string))

	fmt.Println("\n Concurrent execution completed!")
}

// fetchUserData simulates fetching user data from an API
func fetchUserData() (string, error) {
	fmt.Println("    Starting user data fetch...")
	time.Sleep(200 * time.Millisecond) // Simulate API call
	fmt.Println("    User data fetched")
	return "User profile loaded", nil
}

// processAnalytics simulates processing analytics data
func processAnalytics() (string, error) {
	fmt.Println("    Starting analytics processing...")
	time.Sleep(150 * time.Millisecond) // Simulate processing
	fmt.Println("    Analytics processed")
	return "Analytics report generated", nil
}

// sendNotifications simulates sending notifications
func sendNotifications() (string, error) {
	fmt.Println("    Starting notification send...")
	time.Sleep(100 * time.Millisecond) // Simulate notification
	fmt.Println("    Notifications sent")
	return "Email and SMS notifications delivered", nil
}
