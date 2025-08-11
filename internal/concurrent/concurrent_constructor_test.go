package concurrent

import (
	"testing"

	"github.com/maniartech/orchestrator/internal/task"
)

func TestConcurrent_Constructor(t *testing.T){
	c := Concurrent(task.Task(func()(string,error){return "ok",nil}))
	if c==nil { t.Fatal("expected builder") }
	if c.GetChildCount()!=1 { t.Errorf("expected 1 child got %d", c.GetChildCount()) }

	// panic: zero orchestrations
	func(){
		defer func(){ if r:=recover(); r==nil { t.Error("expected panic for zero orchestrations") } }()
		Concurrent()
	}()

	// panic: nil child
	func(){
		defer func(){ if r:=recover(); r==nil { t.Error("expected panic for nil orchestration") } }()
		Concurrent(task.Task(func()(string,error){return "x",nil}), nil)
	}()
}
