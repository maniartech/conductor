package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/maniartech/orchestrator"
)

// PatientRecord represents a healthcare patient record
type PatientRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Condition string    `json:"condition"`
	Priority  string    `json:"priority"` // LOW, MEDIUM, HIGH, CRITICAL
	Timestamp time.Time `json:"timestamp"`
	Processed bool      `json:"processed"`
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
	fmt.Println("🏥 Healthcare Data Pipeline")
	fmt.Println(strings.Repeat("=", 50))

	// Sample patient record
	patient := PatientRecord{
		ID:        "patient-001",
		Name:      "John Doe",
		Age:       45,
		Condition: "Hypertension",
		Priority:  "HIGH",
		Timestamp: time.Now(),
	}

	fmt.Printf("Processing patient: %s (%s)\n", patient.Name, patient.ID)
	fmt.Printf("Condition: %s | Priority: %s | Age: %d\n\n",
		patient.Condition, patient.Priority, patient.Age)

	// Create healthcare processing workflow
	workflow := orchestrator.Setup(
		orchestrator.Sequential(
			// Store patient data in context
			orchestrator.Task(func(ctx orchestrator.Context) (string, error) {
				ctx.Set("patient", patient)
				return "Patient data stored in context", nil
			}).Named("setup"),

			// Data validation and intake
			orchestrator.Task(validatePatientData).Named("validation"),

			// Parallel processing based on priority
			orchestrator.Conditional(
				func(ctx orchestrator.Context) (bool, error) {
					patientData := ctx.Get("patient").(PatientRecord)
					return patientData.Priority == "CRITICAL" || patientData.Priority == "HIGH", nil
				},
				// High priority path
				orchestrator.Concurrent(
					orchestrator.Task(performUrgentScreening).Named("urgent-screening"),
					orchestrator.Task(notifyMedicalTeam).Named("team-notification"),
					orchestrator.Task(checkInsurance).Named("insurance-check"),
				).Named("high-priority-processing"),
				// Standard priority path
				orchestrator.Sequential(
					orchestrator.Task(performStandardScreening).Named("standard-screening"),
					orchestrator.Task(scheduleAppointment).Named("appointment-scheduling"),
				).Named("standard-processing"),
			).Named("priority-routing"),

			// Final processing steps
			orchestrator.Task(updateMedicalRecord).Named("record-update"),
			orchestrator.Task(generateReport).Named("report-generation"),
		).Named("healthcare-pipeline"),
	)

	// Execute workflow
	result, err := workflow.Await()
	if err != nil {
		handleProcessingFailure(patient, err)
		return
	}

	// Display results
	displayHealthcareResults(result, patient)
}

// Healthcare processing implementations
func validatePatientData(ctx orchestrator.Context) (ProcessingResult, error) {
	patient := ctx.Get("patient").(PatientRecord)
	start := time.Now()

	// Simulate data validation
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	success := true
	message := "Patient data validated successfully"

	// Basic validation checks
	if patient.Name == "" || patient.ID == "" {
		success = false
		message = "Missing required patient information"
	} else if patient.Age < 0 || patient.Age > 150 {
		success = false
		message = "Invalid patient age"
	}

	return ProcessingResult{
		Step:     "validation",
		Success:  success,
		Duration: time.Since(start),
		Message:  message,
	}, nil
}

func performUrgentScreening(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()

	// Simulate urgent medical screening
	time.Sleep(time.Duration(rand.IntN(500)+300) * time.Millisecond)

	return ProcessingResult{
		Step:     "urgent-screening",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Urgent screening completed - immediate attention required",
		Data:     map[string]interface{}{"screening_score": 8.5, "requires_immediate_care": true},
	}, nil
}

func notifyMedicalTeam(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()

	// Simulate team notification
	time.Sleep(time.Duration(rand.IntN(150)+50) * time.Millisecond)

	return ProcessingResult{
		Step:     "team-notification",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Medical team notified - Dr. Smith assigned",
		Data:     map[string]interface{}{"assigned_doctor": "Dr. Smith", "notification_sent": true},
	}, nil
}

func checkInsurance(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()

	// Simulate insurance verification
	time.Sleep(time.Duration(rand.IntN(300)+200) * time.Millisecond)

	return ProcessingResult{
		Step:     "insurance-check",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Insurance verified - coverage approved",
		Data:     map[string]interface{}{"coverage_approved": true, "copay": 25.00},
	}, nil
}

func performStandardScreening(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()

	// Simulate standard screening
	time.Sleep(time.Duration(rand.IntN(400)+200) * time.Millisecond)

	return ProcessingResult{
		Step:     "standard-screening",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Standard screening completed",
		Data:     map[string]interface{}{"screening_score": 6.2, "follow_up_needed": false},
	}, nil
}

func scheduleAppointment(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()

	// Simulate appointment scheduling
	time.Sleep(time.Duration(rand.IntN(250)+100) * time.Millisecond)

	appointmentTime := time.Now().Add(24 * time.Hour)

	return ProcessingResult{
		Step:     "appointment-scheduling",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Appointment scheduled successfully",
		Data:     map[string]interface{}{"appointment_time": appointmentTime, "room": "Room 205"},
	}, nil
}

func updateMedicalRecord(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()
	patient := ctx.Get("patient").(PatientRecord)

	// Simulate medical record update
	time.Sleep(time.Duration(rand.IntN(200)+100) * time.Millisecond)

	return ProcessingResult{
		Step:     "record-update",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Medical record updated in EHR system",
		Data:     map[string]interface{}{"record_id": "EHR-" + patient.ID, "updated_at": time.Now()},
	}, nil
}

func generateReport(ctx orchestrator.Context) (ProcessingResult, error) {
	start := time.Now()
	patient := ctx.Get("patient").(PatientRecord)

	// Simulate report generation
	time.Sleep(time.Duration(rand.IntN(300)+150) * time.Millisecond)

	return ProcessingResult{
		Step:     "report-generation",
		Success:  true,
		Duration: time.Since(start),
		Message:  "Patient processing report generated",
		Data:     map[string]interface{}{"report_id": "RPT-" + patient.ID, "pages": 3},
	}, nil
}

// Helper functions
func displayHealthcareResults(result *orchestrator.Result, patient PatientRecord) {
	fmt.Println("📊 Healthcare Processing Results")
	fmt.Println(strings.Repeat("=", 60))

	// Determine which steps were executed based on priority
	var steps []string
	if patient.Priority == "CRITICAL" || patient.Priority == "HIGH" {
		steps = []string{"validation", "urgent-screening", "team-notification",
			"insurance-check", "record-update", "report-generation"}
	} else {
		steps = []string{"validation", "standard-screening", "appointment-scheduling",
			"record-update", "report-generation"}
	}

	totalDuration := time.Duration(0)
	successfulSteps := 0

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

				fmt.Printf("%s %-20s | %8v | %s\n",
					status, procResult.Step, procResult.Duration, procResult.Message)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Processing Summary:\n")
	fmt.Printf("  Patient: %s (%s)\n", patient.Name, patient.ID)
	fmt.Printf("  Priority: %s\n", patient.Priority)
	fmt.Printf("  Total Duration: %v\n", totalDuration)
	fmt.Printf("  Successful Steps: %d/%d\n", successfulSteps, len(steps))
	fmt.Printf("  Status: %s\n", getProcessingStatus(successfulSteps, len(steps)))
	fmt.Println()
}

func getProcessingStatus(successful, total int) string {
	if successful == total {
		return "COMPLETED"
	} else if successful > total/2 {
		return "PARTIALLY_COMPLETED"
	} else {
		return "FAILED"
	}
}

func handleProcessingFailure(patient PatientRecord, err error) {
	fmt.Println("🚨 Healthcare Processing Failed")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Patient: %s (%s)\n", patient.Name, patient.ID)
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("Recovery actions:")
	fmt.Println("  - Patient flagged for manual review")
	fmt.Println("  - Medical staff notified of processing failure")
	fmt.Println("  - Incident logged in healthcare system")
	fmt.Println("  - Backup processing initiated")
}
