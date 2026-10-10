package calc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalizeJSON_KeySorting(t *testing.T) {
	input := []byte(`{"zebra": 1, "alpha": 2, "beta": {"nested_z": true, "nested_a": false}}`)
	expected := `{"alpha":2,"beta":{"nested_a":false,"nested_z":true},"zebra":1}`

	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(canonical) != expected {
		t.Errorf("expected %s, got %s", expected, string(canonical))
	}
}

func TestCanonicalizeJSON_WhitespaceRemoval(t *testing.T) {
	input := []byte(`  {   "a" :   [ 1 ,  2 , 3 ] ,   "b" : "hello world"  }  `)
	expected := `{"a":[1,2,3],"b":"hello world"}`

	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(canonical) != expected {
		t.Errorf("expected %s, got %s", expected, string(canonical))
	}
}

func TestCanonicalizeJSON_ArrayOrderPreserved(t *testing.T) {
	input := []byte(`{"list": [3, 1, 2, {"y": 2, "x": 1}]}`)
	expected := `{"list":[3,1,2,{"x":1,"y":2}]}`

	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(canonical) != expected {
		t.Errorf("expected %s, got %s", expected, string(canonical))
	}
}

func TestCanonicalizeJSON_StringEscapes(t *testing.T) {
	input := []byte(`{"text": "line1\nline2\t\"quoted\""}`)
	expected := `{"text":"line1\nline2\t\"quoted\""}`

	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(canonical) != expected {
		t.Errorf("expected %s, got %s", expected, string(canonical))
	}
}

func TestCanonicalizeJSON_UTF16KeySortEdgeCase(t *testing.T) {
	// In UTF-16 code units: "\u00e9" (é, U+00E9, 233) vs "z" (U+007A, 122).
	// "z" (122) comes before "é" (233).
	// In UTF-8, "é" is 0xC3 0xA9 (195, 169), while "z" is 0x7A (122).
	input := []byte(`{"\u00e9": 1, "z": 2, "a": 3}`)
	// Note: in JCS, unescaped UTF-8 characters are emitted as raw UTF-8 (é), and keys sorted by UTF-16 code units:
	// 'a' (97) < 'z' (122) < 'é' (233)
	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 'é' decoded is UTF-8 bytes for U+00E9
	expectedUTF8 := `{"a":3,"z":2,"é":1}`
	if string(canonical) != expectedUTF8 {
		t.Errorf("expected %s, got %s", expectedUTF8, string(canonical))
	}
}

func TestCanonicalHashSHA256_Deterministic(t *testing.T) {
	json1 := []byte(`{"a": 1, "b": "xyz"}`)
	json2 := []byte(`{  "b" : "xyz" , "a" : 1  }`)

	hash1, err1 := CanonicalHashSHA256(json1)
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	hash2, err2 := CanonicalHashSHA256(json2)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}

	if hash1 != hash2 {
		t.Errorf("expected hashes to match across differently-spaced/keyed JSON, got %s != %s", hash1, hash2)
	}
	if len(hash1) != 64 {
		t.Errorf("expected 64-char hex SHA-256 hash, got %d chars", len(hash1))
	}
}

func TestProofTrace_ToCanonicalJSON(t *testing.T) {
	trace := NewProofTrace(&VarNode{Name: "x"}, &VarNode{Name: "x"})
	trace.AddStep(ProofStep{
		Before:          &VarNode{Name: "x"},
		After:           &VarNode{Name: "x"},
		Rule:            "Identity",
		RuleDescription: "Reflexivity",
		IsSelfVerified:  true,
	})
	trace.Metadata["domain"] = "general"
	trace.Metadata["author"] = "ihd"

	canonical, err := trace.ToCanonicalJSON()
	if err != nil {
		t.Fatalf("failed to produce canonical JSON: %v", err)
	}

	// Verify no formatting whitespace
	if strings.Contains(string(canonical), "\n") || strings.Contains(string(canonical), "  ") {
		t.Errorf("canonical JSON should contain no indentation or unneeded whitespace: %s", string(canonical))
	}

	// Verify it can be round-tripped
	var parsed ProofTraceJSON
	if err := json.Unmarshal(canonical, &parsed); err != nil {
		t.Fatalf("canonical JSON must be valid JSON: %v", err)
	}
	if parsed.Metadata["domain"] != "general" {
		t.Errorf("round-trip metadata domain mismatch: %s", parsed.Metadata["domain"])
	}

	hash, err := trace.ComputeProofHash()
	if err != nil {
		t.Fatalf("failed to compute proof hash: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %s", hash)
	}
}
