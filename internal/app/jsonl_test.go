package app

import (
	"path/filepath"
	"testing"
)

type jsonlTestRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestAppendAndReadJSONLRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records", "records.jsonl")

	if err := appendJSONLRecord(path, jsonlTestRecord{ID: "1", Name: "first"}); err != nil {
		t.Fatalf("appendJSONLRecord first returned error: %v", err)
	}
	if err := appendJSONLRecord(path, jsonlTestRecord{ID: "2", Name: "second"}); err != nil {
		t.Fatalf("appendJSONLRecord second returned error: %v", err)
	}

	records, err := readJSONLRecords[jsonlTestRecord](path)
	if err != nil {
		t.Fatalf("readJSONLRecords returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "1" || records[1].ID != "2" {
		t.Fatalf("unexpected records: %#v", records)
	}
}
