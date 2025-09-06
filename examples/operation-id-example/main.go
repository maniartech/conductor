package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/orchestrator/pkg/builders/concurrent"
	"github.com/maniartech/orchestrator/pkg/builders/conditional"
	"github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
)

func main() {
	// Create various orchestration types and demonstrate GetOperationID() usage

	// Task example
	userTask := task.Task(func(ctx orchContext.Context) (string, error) {
		return "user-data", nil
	}).Named("fetch-user")

	fmt.Printf("Task Operation ID: %s\n", userTask.GetOperationID())

	// Sequential example
	dataFlow := sequential.Sequential(
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "database-connection", nil
		}).Named("connect-db"),
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "query-result", nil
		}).Named("execute-query"),
	).Named("data-processing")

	fmt.Printf("Sequential Operation ID: %s\n", dataFlow.GetOperationID())

	// Concurrent example
	parallelTasks := concurrent.Concurrent(
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "service-a", nil
		}).Named("call-service-a"),
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "service-b", nil
		}).Named("call-service-b"),
	).Named("parallel-api-calls")

	fmt.Printf("Concurrent Operation ID: %s\n", parallelTasks.GetOperationID())

	// Conditional example
	businessLogic := conditional.Conditional(
		func(ctx orchContext.Context) (bool, error) {
			return true, nil // Condition function
		},
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "premium-flow", nil
		}).Named("premium-user"),
		task.Task(func(ctx orchContext.Context) (string, error) {
			return "standard-flow", nil
		}).Named("standard-user"),
	).Named("user-type-logic")

	fmt.Printf("Conditional Operation ID: %s\n", businessLogic.GetOperationID())

	// Demonstrate operation ID usage in error tracking
	_, err := userTask.Execute(context.Background(), config.Config{})
	if err != nil {
		log.Printf("Operation %s failed: %v", userTask.GetOperationID(), err)
	} else {
		fmt.Printf("Operation %s succeeded\n", userTask.GetOperationID())
	}

	// Demonstrate operation ID consistency
	fmt.Printf("Operation ID consistency check: %s == %s? %t\n",
		userTask.GetOperationID(),
		userTask.GetOperationID(),
		userTask.GetOperationID() == userTask.GetOperationID())

	// Demonstrate uniqueness between different instances
	anotherTask := task.Task(func(ctx orchContext.Context) (string, error) {
		return "another-result", nil
	}).Named("fetch-user") // Same name, different instance

	fmt.Printf("Different instances with same name have same IDs: %s == %s? %t\n",
		userTask.GetOperationID(),
		anotherTask.GetOperationID(),
		userTask.GetOperationID() == anotherTask.GetOperationID())

	// Demonstrate true uniqueness with unnamed tasks
	unnamedTask1 := task.Task(func(ctx orchContext.Context) (string, error) {
		return "result1", nil
	})
	unnamedTask2 := task.Task(func(ctx orchContext.Context) (string, error) {
		return "result2", nil
	})

	fmt.Printf("Different unnamed instances have different IDs: %s != %s? %t\n",
		unnamedTask1.GetOperationID(),
		unnamedTask2.GetOperationID(),
		unnamedTask1.GetOperationID() != unnamedTask2.GetOperationID())
}
