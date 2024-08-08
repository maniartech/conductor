package orchestrator_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/orchestrator" // Update with the correct module path
	"github.com/stretchr/testify/assert"
)

func TestGoOrchestrationBase(t *testing.T) {
	future := orchestrator.Func(processAsync, "A", 1000)

	isOrchestration := false

	if _, ok := interface{}(future).(*orchestrator.Orchestration); ok {
		isOrchestration = true
	}

	assert.Equal(t, true, isOrchestration)
	assert.Equal(t, true, future.NotStarted())
	assert.Equal(t, false, future.Pending())
	assert.Equal(t, false, future.Finished())
}

func TestGoOrchestration(t *testing.T) {
	future := orchestrator.Func(processAsync, "A", 1000)
	result, err := future.Await()

	assert.Equal(t, true, future.Finished())

	assert.Equal(t, "A", result)
	assert.Equal(t, nil, err)

	future = orchestrator.Func(processAsync, "A", 1000, errors.New("invalid-action"))
	result, err = future.Await()

	assert.Equal(t, true, future.Finished())

	assert.Equal(t, nil, result)
	assert.EqualError(t, err, "invalid-action")

	_, err = future.Orchestrations()
	assert.Error(t, err, "not-a-batch")
}

func TestBatchGo(t *testing.T) {
	vals := make([]string, 0)
	newCB := func() func(string) {
		return func(s string) {
			vals = append(vals, s)
		}
	}

	p := orchestrator.Async(
		orchestrator.Func(processAsync, "A", 3000, newCB()),
		orchestrator.Func(processAsync, "B", 2000, newCB()),
		orchestrator.Sync( // Calls Func routines in sequence!
			orchestrator.Func(processAsync, "C", 1000, newCB()),
			orchestrator.Func(processAsync, "D", 500, newCB()),
			orchestrator.Func(processAsync, "E", 100, newCB()),
		),
		orchestrator.Async(
			orchestrator.Func(processAsync, "F", 200, newCB()),
			orchestrator.Func(processAsync, "G", 0, newCB()),
		),
	)

	assert.Equal(t, true, p.NotStarted())
	p.Await()
	childOrchestrations, err := p.Orchestrations()

	assert.Equal(t, true, p.Finished())
	assert.Equal(t, true, err == nil)
	assert.Equal(t, 4, len(childOrchestrations))
	assert.Equal(t, "G,F,C,D,E,B,A", strings.Join(vals, ","))
}

func processAsync(p *orchestrator.Orchestration, args ...interface{}) {
	s := args[0].(string)
	ms := args[1].(int)

	time.Sleep(time.Duration(ms) * time.Millisecond)

	defer func() {
		// If callback is supplied, call it by passing s!
		if len(args) == 3 {
			switch args[2].(type) {
			case func(string):
				p.Done(s)
				cb := args[2].(func(string))
				cb(s)
			case error:
				p.Done(args[2])
			default:
				p.Done(s)
			}
			return
		}
		p.Done(s)
	}()
}
