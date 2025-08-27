package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// IoTData represents sensor data from IoT devices
type IoTData struct {
	DeviceID       string                 `json:"device_id"`
	DeviceType     string                 `json:"device_type"`
	Location       string                 `json:"location"`
	Timestamp      time.Time              `json:"timestamp"`
	Metrics        map[string]interface{} `json:"metrics"`
	BatteryLevel   float64                `json:"battery_level"`
	SignalStrength int                    `json:"signal_strength"`
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
	fmt.Println("🌐 IoT Data Processing Pipeline")
	fmt.Println(strings.Repeat("=", 50))

	// Sample IoT sensor data
	iotData := IoTData{
		DeviceID:   "sensor-temp-001",
		DeviceType: "temperature",
		Location:   "Building A - Floor 3",
		Timestamp:  time.Now(),
		Metrics: map[string]interface{}{
			"temperature": 23.5,
			"humidity":    65.2,
			"pressure":    1013.25,
		},
		BatteryLevel:   85.5,
		SignalStrength: -45,
	}

	fmt.Printf("Processing data from: %s (%s)\n", iotData.DeviceID, iotData.DeviceType)
	fmt.Printf("Location: %s\n", iotData.Location)
	fmt.Printf("Battery: %.1f%% | Signal: %d dBm\n\n", iotData.BatteryLevel, iotData.SignalStrength)

	// Create IoT data processing workflow
	workflow := orchestrator.Setup(
		orchestrator.Sequential(
			// Data validation and preprocessing
			orchestrator.Task(func() (ProcessingResult, error) {
				return validateIoTData(iotData)
			}).Named("validation"),

			// Parallel processing pipeline
			orchestrator.Concurrent(
				// Data transformation and enrichment
				orchestrator.Sequential(
					orchestrator.Task(func() (ProcessingResult, error) {
						return transformData(iotData)
					}).Named("data-transformation"),

					orchestrator.Task(func() (ProcessingResult, error) {
						return enrichWithMetadata(iotData)
					}).Named("metadata-enrichment"),
				).Named("data-processing"),

				// Real-time analytics
				orchestrator.Task(func() (ProcessingResult, error) {
					return performRealTimeAnalytics(iotData)
				}).Named("real-time-analytics"),

				// Anomaly detection
				orchestrator.Task(func() (ProcessingResult, error) {
					return detectAnomalies(iotData)
				}).Named("anomaly-detection"),

				// Device health monitoring
				orchestrator.Task(func() (ProcessingResult, error) {
					return monitorDeviceHealth(iotData)
				}).Named("device-health"),
			).Named("parallel-processing"),

			// Conditional alerting based on anomalies
			orchestrator.Conditional(
				func(ctx orchestrator.Context) (bool, error) {
					// Check if anomalies were detected
					if anomalyResult := ctx.Get("anomaly-detection"); anomalyResult != nil {
						if result, ok := anomalyResult.(ProcessingResult); ok {
							if data, ok := result.Data.(map[string]interface{}); ok {
								return data["anomaly_detected"].(bool), nil
							}
						}
					}
					return false, nil
				},
				// Alert path
				orchestrator.Concurrent(
					orchestrator.Task(func() (ProcessingResult, error) {
						return sendAlert(iotData)
					}).Named("alert-notification"),

					orchestrator.Task(func() (ProcessingResult, error) {
						return logIncident(iotData)
					}).Named("incident-logging"),
				).Named("alert-processing"),
				// Normal path
				orchestrator.Task(func() (ProcessingResult, error) {
					return routineLogging(iotData)
				}).Named("routine-logging"),
			).Named("conditional-alerting"),

			// Data storage
			orchestrator.Task(func() (ProcessingResult, error) {
				return storeProcessedData(iotData)
			}).Named("data-storage"),
		).Named("iot-processing-pipeline"),
	)

	// Execute workflow
	result, err := workflow.Await()
	if err != nil {
		handleProcessingFailure(iotData, err)
		return
	}

	// Display results
	displayIoTResults(result, iotData)
}

// IoT processing implementations
func validateIoTData(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate data validation
	time.Sleep(time.Duration(rand.IntN(100)+50) * time.Millisecond)

	success := true
	message := "IoT data validated successfully"

	// Basic validation checks
	if data.DeviceID == "" || data.DeviceType == "" {
		success = false
		message = "Missing required device information"
	} else if data.BatteryLevel < 0 || data.BatteryLevel > 100 {
		success = false
		message = "Invalid battery level"
	} else if len(data.Metrics) == 0 {
		success = false
		message = "No sensor metrics provided"
	}

	return ProcessingResult{
		Step:     "validation",
		Success:  success,
		Duration: time.Since(start),
		Message:  message,
	}, nil
}

func transformData(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate data transformation
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	// Transform temperature to different units
	transformedMetrics := make(map[string]interface{})
	for key, value := range data.Metrics {
		if key == "temperature" && value != nil {
			if temp, ok := value.(float64); ok {
				transformedMetrics["temperature_celsius"] = temp
				transformedMetrics["temperature_fahrenheit"] = temp*9/5 + 32
				transformedMetrics["temperature_kelvin"] = temp + 273.15
			}
		} else {
			transformedMetrics[key] = value
		}
	}

	return ProcessingResult{
		Step:     "data-transformation",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Data transformed with unit conversions",
		Data:     map[string]interface{}{"transformed_metrics": transformedMetrics},
	}, nil
}

func enrichWithMetadata(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate metadata enrichment
	time.Sleep(time.Duration(rand.IntN(150)+75) * time.Millisecond)

	metadata := map[string]interface{}{
		"processing_timestamp": time.Now(),
		"data_quality_score":   0.95,
		"location_zone":        "Zone-A3",
		"device_firmware":      "v2.1.3",
		"network_provider":     "IoT-Network-1",
	}

	return ProcessingResult{
		Step:     "metadata-enrichment",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Data enriched with contextual metadata",
		Data:     map[string]interface{}{"metadata": metadata},
	}, nil
}

func performRealTimeAnalytics(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate real-time analytics
	time.Sleep(time.Duration(rand.IntN(300)+150) * time.Millisecond)

	analytics := map[string]interface{}{
		"trend_analysis":    "stable",
		"prediction_score":  0.87,
		"efficiency_rating": "good",
		"performance_index": 8.2,
	}

	return ProcessingResult{
		Step:     "real-time-analytics",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Real-time analytics completed",
		Data:     map[string]interface{}{"analytics": analytics},
	}, nil
}

func detectAnomalies(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate anomaly detection
	time.Sleep(time.Duration(rand.IntN(250)+125) * time.Millisecond)

	// Simple anomaly detection logic
	anomalyDetected := false
	anomalyScore := 0.1

	if temp, ok := data.Metrics["temperature"].(float64); ok {
		if temp > 30 || temp < 15 {
			anomalyDetected = true
			anomalyScore = 0.8
		}
	}

	if data.BatteryLevel < 20 {
		anomalyDetected = true
		anomalyScore = 0.6
	}

	message := "No anomalies detected"
	if anomalyDetected {
		message = "Anomalies detected - requires attention"
	}

	return ProcessingResult{
		Step:     "anomaly-detection",
		Success:  true,
		Duration: time.Since(start),
		Message:  message,
		Data: map[string]interface{}{
			"anomaly_detected": anomalyDetected,
			"anomaly_score":    anomalyScore,
		},
	}, nil
}

func monitorDeviceHealth(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate device health monitoring
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	healthScore := 0.9
	if data.BatteryLevel < 30 {
		healthScore -= 0.3
	}
	if data.SignalStrength < -60 {
		healthScore -= 0.2
	}

	status := "healthy"
	if healthScore < 0.5 {
		status = "critical"
	} else if healthScore < 0.7 {
		status = "warning"
	}

	return ProcessingResult{
		Step:     "device-health",
		Success:  true,
		Duration: time.Since(start),
		Message:  fmt.Sprintf("Device health: %s (score: %.2f)", status, healthScore),
		Data: map[string]interface{}{
			"health_score": healthScore,
			"status":       status,
		},
	}, nil
}

func sendAlert(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate alert sending
	time.Sleep(time.Duration(rand.IntN(100)+50) * time.Millisecond)

	return ProcessingResult{
		Step:     "alert-notification",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Alert sent to monitoring team",
		Data:     map[string]interface{}{"alert_id": "ALERT-" + data.DeviceID},
	}, nil
}

func logIncident(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate incident logging
	time.Sleep(time.Duration(rand.IntN(75)+25) * time.Millisecond)

	return ProcessingResult{
		Step:     "incident-logging",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Incident logged in system",
		Data:     map[string]interface{}{"incident_id": "INC-" + data.DeviceID},
	}, nil
}

func routineLogging(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate routine logging
	time.Sleep(time.Duration(rand.IntN(50)+25) * time.Millisecond)

	return ProcessingResult{
		Step:     "routine-logging",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Data logged for routine monitoring",
	}, nil
}

func storeProcessedData(data IoTData) (ProcessingResult, error) {
	start := time.Now()

	// Simulate data storage
	time.Sleep(time.Duration(rand.IntN(150)+75) * time.Millisecond)

	return ProcessingResult{
		Step:     "data-storage",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Processed data stored in time-series database",
		Data:     map[string]interface{}{"storage_id": "TS-" + data.DeviceID},
	}, nil
}

// Helper functions
func displayIoTResults(result *orchestrator.Result, data IoTData) {
	fmt.Println("📊 IoT Data Processing Results")
	fmt.Println(strings.Repeat("=", 60))

	steps := []string{"validation", "data-transformation", "metadata-enrichment",
		"real-time-analytics", "anomaly-detection", "device-health",
		"alert-notification", "incident-logging", "routine-logging", "data-storage"}

	totalDuration := time.Duration(0)
	successfulSteps := 0
	anomalyDetected := false

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

				// Check for anomaly detection
				if step == "anomaly-detection" && procResult.Data != nil {
					if data, ok := procResult.Data.(map[string]interface{}); ok {
						if detected, ok := data["anomaly_detected"].(bool); ok {
							anomalyDetected = detected
						}
					}
				}

				fmt.Printf("%s %-20s | %8v | %s\n",
					status, procResult.Step, procResult.Duration, procResult.Message)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Processing Summary:\n")
	fmt.Printf("  Device: %s (%s)\n", data.DeviceID, data.DeviceType)
	fmt.Printf("  Location: %s\n", data.Location)
	fmt.Printf("  Total Duration: %v\n", totalDuration)
	fmt.Printf("  Successful Steps: %d/%d\n", successfulSteps, len(steps))
	fmt.Printf("  Anomaly Detected: %v\n", anomalyDetected)
	fmt.Printf("  Battery Level: %.1f%%\n", data.BatteryLevel)
	fmt.Println()
}

func handleProcessingFailure(data IoTData, err error) {
	fmt.Println("🚨 IoT Data Processing Failed")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Device: %s (%s)\n", data.DeviceID, data.DeviceType)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Data queued for retry processing")
	fmt.Println("  - Device marked for health check")
	fmt.Println("  - Incident logged in monitoring system")
	fmt.Println("  - Fallback processing initiated")
}
