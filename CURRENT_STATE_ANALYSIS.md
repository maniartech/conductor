# Current State Analysis - Task Context Enhancement

## Current TaskBuilder Implementation

### Struct Definition
```go
// Location: pkg/builders/task/builder.go:77-80
type TaskBuilder[T any] struct {
    *orchestration.BaseOrchestrationBuilder
    fn func() (T, error)  // Only supports functions without context
}
```

### Current Task Constructor
```go
// Location: pkg/builders/task/builder.go:102-110
func Task[T any](fn func() (T, error)) *TaskBuilder[T] {
    if fn == nil {
        panic("task function cannot be nil")
    }
    return &TaskBuilder[T]{
        BaseOrchestrationBuilder: orchestration.NewBaseOrchestrationBuilder("task"),
        fn:                       fn,
    }
}
```

### Current Task Execution Flow

#### Execute Method
```go
// Location: pkg/builders/task/builder.go:225
func (tb *TaskBuilder[T]) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
    // ... validation and setup ...
    
    // Execute with comprehensive error handling
    taskResult, taskError := tb.safeExecute(execCtx)  // Only passes Go context
    
    // ... result handling ...
}
```

#### SafeExecute Method
```go
// Location: pkg/builders/task/builder.go:347
func (tb *TaskBuilder[T]) safeExecute(ctx context.Context) (T, error) {
    // ... setup and panic recovery ...
    
    // Execute the actual task function
    taskResult, taskError = tb.fn()  // NO CONTEXT PASSED - This is the problem!
    
    // ... completion handling ...
}
```

## Incomplete TaskWithContext Implementation

### Existing Stub
```go
// Location: pkg/builders/task/builder.go:136-157
func TaskWithContext[T any](fn func(Context) (T, error)) *TaskBuilder[T] {
    if fn == nil {
        panic("task function cannot be nil")
    }
    
    // Wrap the context-aware function to match the current task signature
    wrappedFn := func() (T, error) {
        // This will be replaced with proper context passing in Execute method
        var zero T
        return zero, fmt.Errorf("TaskWithContext requires orchestrator context - this should not be called directly")
    }
    
    taskBuilder := &TaskBuilder[T]{
        BaseOrchestrationBuilder: orchestration.NewBaseOrchestrationBuilder("task"),
        fn:                       wrappedFn,
    }
    
    // Store the original context-aware function for execution
    taskBuilder.contextFn = fn  // ERROR: contextFn field doesn't exist!
    
    return taskBuilder
}
```

### Issues with Current Implementation
1. **Missing Field**: `contextFn` field doesn't exist in TaskBuilder struct
2. **Wrong Import**: Uses `Context` instead of `orchContext.Context`
3. **Incomplete Execution**: safeExecute doesn't know how to handle context-aware functions
4. **No Config Passing**: safeExecute doesn't receive config to create orchestrator context

## Current Example State

### Video Processing Example
```go
// Location: examples/video-processing/main.go:47-70
// Current approach: using closures to capture video data
// LIMITATION: Tasks can't access orchestrator context or control orchestration
workflow := orchestrator.Setup(
    orchestrator.Sequential(
        orchestrator.Task(func(ctx orchestrator.Context) (ProcessingResult, error) { 
            return uploadVideoToStorage(video)  // Closure captures video
        }).Named("upload"),
        
        // ... more tasks using closures ...
    ).Named("video-pipeline"),
)
```

### Problems with Current Approach
1. **Closure Dependency**: Each task needs a closure to access data
2. **No Shared State**: Tasks can't share data through orchestrator context
3. **No Flow Control**: Tasks can't influence orchestration behavior
4. **Verbose**: Requires wrapper functions for each task

## Existing Pattern in Conditional Builder

### Working Example
```go
// Location: pkg/builders/conditional/conditional.go:328-331
func (cb *ConditionalBuilder) executeConditional(ctx context.Context, config config.Config, result *result.Result) error {
    // Create orchestration context for condition evaluation
    orchCtx := orchContext.NewContext(config)
    
    // Evaluate condition with comprehensive error handling
    conditionResult, conditionError := cb.safeEvaluateCondition(ctx, orchCtx)
    // ...
}

// Location: pkg/builders/conditional/conditional.go:419
func (cb *ConditionalBuilder) safeEvaluateCondition(ctx context.Context, orchCtx orchContext.Context) (bool, error) {
    // Execute the condition function - now handles both result and error
    conditionResult, conditionError = cb.condition(orchCtx)  // Passes orchestrator context!
    // ...
}
```

### Key Insights from Conditional Pattern
1. **Context Creation**: Uses `orchContext.NewContext(config)` to create orchestrator context
2. **Context Passing**: Passes orchestrator context to the function
3. **Dual Context**: Maintains both Go context and orchestrator context
4. **Working Implementation**: This pattern already works for conditionals

## Required Changes Summary

### 1. TaskBuilder Struct Changes
- Add `contextFn func(orchContext.Context) (T, error)` field
- Add `useContext bool` field to distinguish function types

### 2. Method Signature Changes
- `safeExecute` needs to accept `config.Config` parameter
- `safeExecute` needs to create and pass orchestrator context

### 3. Import Changes
- Already added: `orchContext "github.com/maniartech/orchestrator/pkg/context"`

### 4. Constructor Changes
- Fix `TaskWithContext` to properly set struct fields
- Use correct context type reference

### 5. Execution Logic Changes
- Create orchestrator context in safeExecute
- Choose correct function based on useContext flag
- Pass orchestrator context to context-aware functions

## Next Implementation Steps
1. Fix TaskBuilder struct definition
2. Fix TaskWithContext constructor
3. Update safeExecute method signature and implementation
4. Update Execute method to pass config
5. Test the implementation
6. Update examples to demonstrate new capabilities