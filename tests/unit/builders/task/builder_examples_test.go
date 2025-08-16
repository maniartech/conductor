package task

import (
	"context"
	"fmt"

	"github.com/maniartech/orchestrator/internal/config"
	. "github.com/maniartech/orchestrator/pkg/builders/task"
)

func ExampleTask() {
	tk := Task(func() (string, error) { return "Hello, World!", nil })
	res, err := tk.Execute(context.Background(), config.DefaultConfig())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Result: %v\n", res.Get("task_result"))
	// Output:
	// Result: Hello, World!
}

func ExampleTaskBuilder_Named() {
	tk := Task(func() (int, error) { return 42, nil }).Named("answer-task")
	res, err := tk.Execute(context.Background(), config.DefaultConfig())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Answer: %v\n", res.Get("answer-task"))
	// Output:
	// Answer: 42
}

func ExampleTaskBuilder_GetStatus() {
	tk := Task(func() (string, error) { return "Hello, World!", nil })
	fmt.Printf("Initial status: %v\n", tk.GetStatus())
	tk.Execute(context.Background(), config.DefaultConfig())
	fmt.Printf("Final status: %v\n", tk.GetStatus())
	// Output:
	// Initial status: NotStarted
	// Final status: Completed
}
