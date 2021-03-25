package async_test

import (
	"strings"
	"testing"
	"time"

	"github.com/maniartech/async"
	"github.com/stretchr/testify/assert"
)

func TestBatchGo(t *testing.T) {

	vals := make([]string, 0)
	newCB := func() func(string) {
		return func(s string) {
			vals = append(vals, s)
		}
	}

	async.Go(
		async.Go(fakeAsync, "A", 3000, newCB()),
		async.Go(fakeAsync, "B", 2000, newCB()),
		async.GoQ( // Calls Go routines in queue!
			async.Go(fakeAsync, "C", 1000, newCB()),
			async.Go(fakeAsync, "D", 500, newCB()),
			async.Go(fakeAsync, "E", 100, newCB()),
		),
		async.Go(
			async.Go(fakeAsync, "F", 200, newCB()),
			async.Go(fakeAsync, "G", 0, newCB()),
		),
	).Await()

	assert.Equal(t, "G,F,C,D,E,B,A", strings.Join(vals, ","))
}

func fakeAsync(p *async.Promise, args ...interface{}) {
	s := args[0].(string)
	ms := args[1].(int)
	cb := args[2].(func(string))

	time.Sleep(time.Duration(ms) * time.Millisecond)
	cb(s)
	p.Done(s)
}
