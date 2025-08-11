package task

import (
	"testing"
)

func TestTask(t *testing.T) {
	tk := Task(func() (string, error) { return "hello", nil })
	if tk == nil {
		t.Fatal("Task() returned nil")
	}
	if tk.fn == nil {
		t.Error("Task function should not be nil")
	}
	if tk.GetName() != "" {
		t.Error("Task name should be empty initially")
	}
	if tk.GetConfig() != nil {
		t.Error("Task config should be nil initially")
	}
}

func TestTaskPanicOnNilFunction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Task() should panic when function is nil")
		}
	}()
	Task[string](nil)
}

func TestTaskBuilderGenericTypes(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	cases := []struct {
		name string
		fn   any
	}{
		{"string", func() (string, error) { return "test", nil }},
		{"int", func() (int, error) { return 42, nil }},
		{"bool", func() (bool, error) { return true, nil }},
		{"slice", func() ([]string, error) { return []string{"a", "b", "c"}, nil }},
		{"map", func() (map[string]int, error) { return map[string]int{"k": 1}, nil }},
		{"struct", func() (User, error) { return User{ID: 1, Name: "John"}, nil }},
		{"pointer", func() (*string, error) { s := "ptr"; return &s, nil }},
		{"interface", func() (interface{}, error) { return 3.14, nil }},
		{"channel", func() (chan int, error) { ch := make(chan int, 1); ch <- 1; return ch, nil }},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			switch f := c.fn.(type) {
			case func() (string, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (int, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (bool, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() ([]string, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (map[string]int, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (User, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (*string, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (interface{}, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			case func() (chan int, error):
				if Task(f) == nil {
					t.Error("nil")
				}
			default:
				t.Fatalf("unexpected fn type %T", f)
			}
		})
	}
}
