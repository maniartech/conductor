package orchestration

import "testing"

func TestBaseOrchestrationBuilder_GetOperationID_UniqueUnnamed(t *testing.T) {
	b1 := NewBaseOrchestrationBuilder("sequential")
	b2 := NewBaseOrchestrationBuilder("sequential")
	mock1 := &MockOrchestration{BaseOrchestrationBuilder: b1}
	mock2 := &MockOrchestration{BaseOrchestrationBuilder: b2}
	id1 := b1.GetOperationID(mock1)
	id2 := b2.GetOperationID(mock2)
	if id1 == id2 {
		t.Fatalf("expected unique ids, got same %s", id1)
	}
	if len(id1) == 0 || len(id2) == 0 {
		t.Fatal("empty operation id")
	}
	if want := "sequential-"; id1[:len(want)] != want || id2[:len(want)] != want {
		t.Errorf("expected prefix %s in ids %s %s", want, id1, id2)
	}
}

func TestBaseOrchestrationBuilder_GetOperationID_NamedOverrides(t *testing.T) {
	b := NewBaseOrchestrationBuilder("concurrent")
	b.SetName("pipeline")
	mock := &MockOrchestration{BaseOrchestrationBuilder: b}
	id := b.GetOperationID(mock)
	if id != "concurrent-pipeline" {
		t.Fatalf("expected concurrent-pipeline got %s", id)
	}
}
