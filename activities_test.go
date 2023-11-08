package choreo_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoChoreographyBase(t *testing.T) {
	future := choreographer.Func(processAsync, "A", 1000)

	isChoreography := false

	if _, ok := interface{}(future).(*choreographer.Choreography); ok {
		isChoreography = true
	}

	assert.Equal(t, true, isChoreography)
	assert.Equal(t, true, future.NotStarted())
	assert.Equal(t, false, future.Pending())
	assert.Equal(t, false, future.Finished())
}

func TestGoChoreography(t *testing.T) {
	future := choreographer.Func(processAsync, "A", 1000)
	result, err := future.Await()

	assert.Equal(t, true, future.Finished())

	assert.Equal(t, "A", result)
	assert.Equal(t, nil, err)

	future = choreographer.Func(processAsync, "A", 1000, errors.New("invalid-action"))
	result, err = future.Await()

	assert.Equal(t, true, future.Finished())

	assert.Equal(t, nil, result)
	assert.EqualError(t, err, "invalid-action")

	_, err = future.Choreographys()
	assert.Error(t, err, "not-a-batch")
}

func TestBatchGo(t *testing.T) {
	vals := make([]string, 0)
	newCB := func() func(string) {
		return func(s string) {
			vals = append(vals, s)
		}
	}

	p := choreographer.Async(
		choreographer.Func(processAsync, "A", 3000, newCB()),
		choreographer.Func(processAsync, "B", 2000, newCB()),
		choreographer.Sync( // Calls Func routines in queue!
			choreographer.Func(processAsync, "C", 1000, newCB()),
			choreographer.Func(processAsync, "D", 500, newCB()),
			choreographer.Func(processAsync, "E", 100, newCB()),
		),
		choreographer.Async(
			choreographer.Func(processAsync, "F", 200, newCB()),
			choreographer.Func(processAsync, "G", 0, newCB()),
		),
	)

	assert.Equal(t, true, p.NotStarted())
	p.Await()
	childChoreographys, err := p.Choreographys()

	assert.Equal(t, true, p.Finished())
	assert.Equal(t, true, err == nil)
	assert.Equal(t, 4, len(childChoreographys))
	assert.Equal(t, "G,F,C,D,E,B,A", strings.Join(vals, ","))
}

func processAsync(p *choreographer.Choreography, args ...interface{}) {
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
