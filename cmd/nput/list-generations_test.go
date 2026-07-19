package main

import (
	"encoding/json"
	"testing"

	"github.com/yasunori0418/niface/go/conformance"

	"github.com/yasunori0418/nput/internal/engine"
)

// TestListGenerationsJSONInfoGenerations pins list-generations' --json shape (→ issue #132,
// ADR-0043 §5): the listing rides in result.info.generations as {number, date, current} rows,
// items stays an empty array, and the SubjectResult.generation slot stays absent (info and
// generation carrying the same numbers would be double-encoding). The envelope is conformant.
func TestListGenerationsJSONInfoGenerations(t *testing.T) {
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}

	gens := []engine.Generation{
		{Number: 1, Date: "2026-07-18 09:00:00"},
		{Number: 2, Date: "2026-07-19 12:00:00", Current: true},
	}
	r, buf := newTestRun("list-generations")
	r.beginSubject("home")
	r.setPayload(&nifacePayload{info: map[string]any{"generations": generationRows(gens)}})
	if err := r.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings: %v\ndocument: %s", findings, buf.String())
	}

	doc := decodeEnvelope(t, buf)
	if doc["dryRun"] != false {
		t.Errorf("dryRun = %v, want false (a read-only command reflects only the flag)", doc["dryRun"])
	}
	sr := doc["results"].([]any)[0].(map[string]any)
	if _, ok := sr["generation"]; ok {
		t.Errorf("generation slot present, want absent (info.generations is the single encoding): %v", sr["generation"])
	}
	res := sr["result"].(map[string]any)
	if items := res["items"].([]any); len(items) != 0 {
		t.Errorf("items = %v, want empty (the enumeration is info, not items)", items)
	}
	rows := res["info"].(map[string]any)["generations"].([]any)
	if len(rows) != 2 {
		t.Fatalf("info.generations = %v, want 2 rows", rows)
	}
	current := rows[1].(map[string]any)
	if current["number"] != json.Number("2") || current["date"] != "2026-07-19 12:00:00" || current["current"] != true {
		t.Errorf("rows[1] = %v, want number=2 date verbatim current=true", current)
	}
	if first := rows[0].(map[string]any); first["current"] != false {
		t.Errorf("rows[0].current = %v, want false (the key is always present)", first["current"])
	}
}

// TestGenerationRowsEmptyStaysArray pins that a profile without generations still lists
// "generations": [] (a non-nil slice; nil would marshal the key away).
func TestGenerationRowsEmptyStaysArray(t *testing.T) {
	got := generationRows(nil)
	if got == nil || len(got) != 0 {
		t.Errorf("generationRows(nil) = %#v, want a non-nil empty slice", got)
	}
}
