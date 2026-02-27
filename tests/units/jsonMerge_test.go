package units

import (
	"context"
	"encoding/json"
	"testing"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func TestJSONMerge_SimpleMerge(t *testing.T) {
	base := map[string]string{"foo": "bar"}
	patch := map[string]string{"baz": "qux"}

	result, err := utils.JSONMerge(context.Background(), base, patch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]string
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["foo"] != "bar" {
		t.Errorf("expected foo=bar, got foo=%s", merged["foo"])
	}
	if merged["baz"] != "qux" {
		t.Errorf("expected baz=qux, got baz=%s", merged["baz"])
	}
}

func TestJSONMerge_OverwriteValue(t *testing.T) {
	base := map[string]string{"foo": "bar"}
	patch := map[string]string{"foo": "updated"}

	result, err := utils.JSONMerge(context.Background(), base, patch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]string
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["foo"] != "updated" {
		t.Errorf("expected foo=updated, got foo=%s", merged["foo"])
	}
}

func TestJSONMerge_MultiplePatches(t *testing.T) {
	base := map[string]int{"a": 1}
	patch1 := map[string]int{"b": 2}
	patch2 := map[string]int{"c": 3}
	patch3 := map[string]int{"a": 10}

	result, err := utils.JSONMerge(context.Background(), base, patch1, patch2, patch3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]int
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected := map[string]int{"a": 10, "b": 2, "c": 3}
	for k, v := range expected {
		if merged[k] != v {
			t.Errorf("expected %s=%d, got %s=%d", k, v, k, merged[k])
		}
	}
}

func TestJSONMerge_NestedObjects(t *testing.T) {
	type Inner struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
	}

	base := Outer{Name: "test", Inner: Inner{X: 1, Y: 2}}
	patch := map[string]any{
		"inner": map[string]int{"x": 100},
	}

	result, err := utils.JSONMerge(context.Background(), base, patch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged Outer
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged.Name != "test" {
		t.Errorf("expected name=test, got name=%s", merged.Name)
	}
	if merged.Inner.X != 100 {
		t.Errorf("expected inner.x=100, got inner.x=%d", merged.Inner.X)
	}
	// JSON merge patch does deep merge, preserving unmodified nested fields
	if merged.Inner.Y != 2 {
		t.Errorf("expected inner.y=2 (preserved), got inner.y=%d", merged.Inner.Y)
	}
}

func TestJSONMerge_NoPatches(t *testing.T) {
	base := map[string]string{"foo": "bar"}

	result, err := utils.JSONMerge(context.Background(), base)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]string
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["foo"] != "bar" {
		t.Errorf("expected foo=bar, got foo=%s", merged["foo"])
	}
}

func TestJSONMerge_NullRemovesKey(t *testing.T) {
	base := map[string]any{"foo": "bar", "baz": "qux"}
	patch := map[string]any{"foo": nil}

	result, err := utils.JSONMerge(context.Background(), base, patch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]any
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, exists := merged["foo"]; exists {
		t.Errorf("expected foo to be removed, but it still exists")
	}
	if merged["baz"] != "qux" {
		t.Errorf("expected baz=qux, got baz=%v", merged["baz"])
	}
}

func TestJSONMerge_EmptyPatch(t *testing.T) {
	base := map[string]string{"foo": "bar"}
	patch := map[string]string{}

	result, err := utils.JSONMerge(context.Background(), base, patch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var merged map[string]string
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["foo"] != "bar" {
		t.Errorf("expected foo=bar, got foo=%s", merged["foo"])
	}
}
