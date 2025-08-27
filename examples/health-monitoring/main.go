package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	ServiceName  string        `json:"service_name"`
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// HealthCheckResult aggregates all service health checks
type HealthCheckResult struct {
	OverallStatus string          `json:"overall_status"`
	Services      []ServiceHealth `json:"services"`
	CheckTime     time.Time       `json:"check_time"`
	TotalServices int             `json:"total_services"`
	HealthyCount  int             `json:"healthy_count"`
}

func main() {
	fmt.Println("🏥 Health Monitoring System - Service Mesh Health Checks")
	fmt.Println("Real-world example: Netflix/Uber service monitoring")
	fmt.Println()

	start := time.Now()

	// Execute health checks for all microservices concurrently
	result, err := orchestrator.Setup(
		orchestrator.Concurrent(
			orchestrator.Task(checkUserServiceHealth).Named("user-service"),
			orchestrator.Task(checkPaymentServiceHealth).Named("payment-service"),
			orchestrator.Task(checkInventoryServiceHealth).Named("inventory-service"),
			orchestrator.Task(checkNotificationServiceHealth).Named("notification-service"),
			orchestrator.Task(checkAnalyticsServiceHealth).Named("analytics-service"),
		).Named("health-checks"),
	).With(orchestrator.Config{
		Timeout:       5 * time.Second,
		ErrorStrategy: orchestrator.CollectAll, // Continue checking all services even if some fail
	}).Await()

	duration := time.Since(start)

	// Process results even if some services failed
	healthReport := generateHealthReport(result, err)

	fmt.Printf("✅ Health check completed in %v\n\n", duration)
	displayHealthReport(healthReport)

	// Demonstrate alerting based on health status
	if healthReport.OverallStatus == "DEGRADED" || healthReport.OverallStatus == "DOWN" {
		triggerAlert(healthReport)
	}
}

// Service health check functions
func checkUserServiceHealth() (ServiceHealth, error) {
	return performHealthCheck("user-service", "http://user-service:8080/health")
}

func checkPaymentServiceHealth() (ServiceHealth, error) {
	return performHealthCheck("payment-service", "http://payment-service:8081/health")
}

func checkInventoryServiceHealth() (ServiceHealth, error) {
	return performHealthCheck("inventory-service", "http://inventory-service:8082/health")
}

func checkNotificationServiceHealth() (ServiceHealth, error) {
	return performHealthCheck("notification-service", "http://notification-service:8083/health")
}

func checkAnalyticsServiceHealth() (ServiceHealth, error) {
	return performHealthCheck("analytics-service", "http://analytics-service:8084/health")
}

// performHealthCheck simulates HTTP health check to a service endpoint
func performHealthCheck(serviceName, endpoint string) (ServiceHealth, error) {
	start := time.Now()

	fmt.Printf("   🔍 Checking %s health...\n", serviceName)

	// Simulate network call with random latency
	latency := time.Duration(rand.Intn(200)+50) * time.Millisecond
	time.Sleep(latency)

	// Simulate occasional service failures (20% failure rate)
	if rand.Float32() < 0.2 {
		errorMsg := fmt.Sprintf("Connection timeout to %s", endpoint)
		fmt.Printf("   ❌ %s: %s\n", serviceName, errorMsg)
		return ServiceHealth{
			ServiceName:  serviceName,
			Status:       "DOWN",
			ResponseTime: time.Since(start),
			ErrorMessage: errorMsg,
		}, fmt.Errorf("health check failed for %s: %s", serviceName, errorMsg)
	}

	// Simulate different health statuses
	status := "HEALTHY"
	if rand.Float32() < 0.1 {
		status = "DEGRADED"
	}

	responseTime := time.Since(start)
	fmt.Printf("   ✅ %s: %s (%v)\n", serviceName, status, responseTime)

	return ServiceHealth{
		ServiceName:  serviceName,
		Status:       status,
		ResponseTime: responseTime,
	}, nil
}

// generateHealthReport creates a comprehensive health report
func generateHealthReport(result *orchestrator.Result, err error) HealthCheckResult {
	services := []ServiceHealth{}
	healthyCount := 0
	totalServices := 5

	// Collect results from all services
	serviceNames := []string{"user-service", "payment-service", "inventory-service", "notification-service", "analytics-service"}

	for _, serviceName := range serviceNames {
		if serviceResult := result.Get(serviceName); serviceResult != nil {
			if health, ok := serviceResult.(ServiceHealth); ok {
				services = append(services, health)
				if health.Status == "HEALTHY" {
					healthyCount++
				}
			}
		}
	}

	// Determine overall status
	overallStatus := "HEALTHY"
	if healthyCount == 0 {
		overallStatus = "DOWN"
	} else if healthyCount < totalServices {
		overallStatus = "DEGRADED"
	}

	return HealthCheckResult{
		OverallStatus: overallStatus,
		Services:      services,
		CheckTime:     time.Now(),
		TotalServices: totalServices,
		HealthyCount:  healthyCount,
	}
}

// displayHealthReport shows the health check results
func displayHealthReport(report HealthCheckResult) {
	fmt.Println("📊 Health Check Report")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Overall Status: %s\n", getStatusEmoji(report.OverallStatus))
	fmt.Printf("Healthy Services: %d/%d\n", report.HealthyCount, report.TotalServices)
	fmt.Printf("Check Time: %s\n\n", report.CheckTime.Format("2006-01-02 15:04:05"))

	fmt.Println("Service Details:")
	for _, service := range report.Services {
		emoji := getStatusEmoji(service.Status)
		fmt.Printf("  %s %-20s | %-8s | %8v", emoji, service.ServiceName, service.Status, service.ResponseTime)
		if service.ErrorMessage != "" {
			fmt.Printf(" | %s", service.ErrorMessage)
		}
		fmt.Println()
	}
	fmt.Println()
}

// getStatusEmoji returns appropriate emoji for status
func getStatusEmoji(status string) string {
	switch status {
	case "HEALTHY":
		return "✅ HEALTHY"
	case "DEGRADED":
		return "⚠️  DEGRADED"
	case "DOWN":
		return "❌ DOWN"
	default:
		return "❓ UNKNOWN"
	}
}

// triggerAlert simulates alerting system integration
func triggerAlert(report HealthCheckResult) {
	fmt.Println("🚨 ALERT TRIGGERED")
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("System Status: %s\n", report.OverallStatus)
	fmt.Printf("Affected Services: %d\n", report.TotalServices-report.HealthyCount)

	// In a real system, this would:
	// - Send notifications to Slack/PagerDuty
	// - Update monitoring dashboards
	// - Trigger auto-scaling or failover
	// - Log to centralized logging system

	fmt.Println("📧 Notifications sent to:")
	fmt.Println("  - DevOps team via Slack")
	fmt.Println("  - On-call engineer via PagerDuty")
	fmt.Println("  - Monitoring dashboard updated")
	fmt.Println()
}
