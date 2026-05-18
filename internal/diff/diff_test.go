package diff

import "testing"

func TestComputeAddedRemovedChanged(t *testing.T) {
	a := []Row{
		{":id": "r1", "status": "Open", "complaint": "Noise"},
		{":id": "r2", "status": "Open", "complaint": "Trash"},
		{":id": "r3", "status": "Closed", "complaint": "Pothole"},
	}
	b := []Row{
		{":id": "r1", "status": "Closed", "complaint": "Noise"},
		{":id": "r3", "status": "Closed", "complaint": "Pothole"},
		{":id": "r4", "status": "Open", "complaint": "Graffiti"},
	}
	got, err := Compute(a, b, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(got.Added) != 1 || got.Added[0][":id"] != "r4" {
		t.Errorf("added: %+v", got.Added)
	}
	if len(got.Removed) != 1 || got.Removed[0][":id"] != "r2" {
		t.Errorf("removed: %+v", got.Removed)
	}
	if len(got.Changed) != 1 || got.Changed[0].Key != "r1" {
		t.Errorf("changed: %+v", got.Changed)
	}
	if got.Changed[0].Changes[0].Field != "status" {
		t.Errorf("changed field: %+v", got.Changed[0])
	}
}

func TestComputeCustomKey(t *testing.T) {
	a := []Row{{"id": "1", "v": 10}}
	b := []Row{{"id": "1", "v": 20}}
	got, err := Compute(a, b, "id")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(got.Changed) != 1 {
		t.Errorf("changed: %+v", got.Changed)
	}
}

func TestComputeMissingKeyErrors(t *testing.T) {
	a := []Row{{":id": "1"}}
	b := []Row{{"other": "1"}}
	_, err := Compute(a, b, ":id")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestComputeNoDiff(t *testing.T) {
	rows := []Row{{":id": "r1", "x": 1}}
	got, err := Compute(rows, rows, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(got.Added)+len(got.Removed)+len(got.Changed) != 0 {
		t.Errorf("expected zero diff, got %+v", got)
	}
}

func TestComputeDeterministicOrdering(t *testing.T) {
	a := []Row{
		{":id": "z", "x": 1},
		{":id": "a", "x": 1},
	}
	b := []Row{
		{":id": "z", "x": 2},
		{":id": "a", "x": 2},
	}
	for i := 0; i < 5; i++ {
		got, _ := Compute(a, b, "")
		if got.Changed[0].Key != "a" || got.Changed[1].Key != "z" {
			t.Fatalf("non-deterministic ordering: %+v", got.Changed)
		}
	}
}
