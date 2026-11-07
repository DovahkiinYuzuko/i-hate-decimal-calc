package calc

import (
	"math/big"
	"testing"
)

func TestHammingDistance(t *testing.T) {
	u := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	v := []*big.Int{big.NewInt(1), big.NewInt(4), big.NewInt(3)}

	dist, err := HammingDistance(u, v)
	if err != nil {
		t.Fatalf("HammingDistance failed: %v", err)
	}
	if dist != 1 {
		t.Errorf("expected distance 1, got %d", dist)
	}

	// Length mismatch
	w := []*big.Int{big.NewInt(1), big.NewInt(2)}
	if _, err := HammingDistance(u, w); err == nil {
		t.Errorf("expected error for length mismatch, got nil")
	}
}

func TestReedSolomon_Encode_Decode_Roundtrip(t *testing.T) {
	// GF(5): prime = 5
	// Message P(x) = 1 + 2x (k = 2)
	// Evaluation points: [1, 2, 3, 4] (n = 4)
	// P(1) = 3 mod 5
	// P(2) = 5 = 0 mod 5
	// P(3) = 7 = 2 mod 5
	// P(4) = 9 = 4 mod 5
	prime := big.NewInt(5)
	msg := []*big.Int{big.NewInt(1), big.NewInt(2)}
	points := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}

	codeword, err := RSEncode(msg, points, prime)
	if err != nil {
		t.Fatalf("RSEncode failed: %v", err)
	}

	expectedCode := []*big.Int{big.NewInt(3), big.NewInt(0), big.NewInt(2), big.NewInt(4)}
	for i := range expectedCode {
		if codeword[i].Cmp(expectedCode[i]) != 0 {
			t.Errorf("codeword[%d] = %s, want %s", i, codeword[i].String(), expectedCode[i].String())
		}
	}

	// 1. Decode without error
	decoded, err := RSDecode(codeword, points, 2, prime)
	if err != nil {
		t.Fatalf("RSDecode (no error) failed: %v", err)
	}
	if len(decoded) != 2 || decoded[0].Cmp(msg[0]) != 0 || decoded[1].Cmp(msg[1]) != 0 {
		t.Errorf("RSDecode recovered %v, want %v", decoded, msg)
	}

	// 2. Decode with 1 error at index 1: codeword becomes [3, 4, 2, 4]
	// Error capacity e_max = floor((4-2)/2) = 1
	corrupted := []*big.Int{big.NewInt(3), big.NewInt(4), big.NewInt(2), big.NewInt(4)}
	decodedCorrupted, err := RSDecode(corrupted, points, 2, prime)
	if err != nil {
		t.Fatalf("RSDecode (1 error) failed: %v", err)
	}
	if len(decodedCorrupted) != 2 || decodedCorrupted[0].Cmp(msg[0]) != 0 || decodedCorrupted[1].Cmp(msg[1]) != 0 {
		t.Errorf("RSDecode with error recovered %v, want %v", decodedCorrupted, msg)
	}

	// 3. Decode with 2 errors (exceeds e_max = 1): [3, 4, 1, 4]
	uncorrectable := []*big.Int{big.NewInt(3), big.NewInt(4), big.NewInt(1), big.NewInt(4)}
	if _, err := RSDecode(uncorrectable, points, 2, prime); err == nil {
		t.Errorf("expected decoding error for uncorrectable corrupted word, got nil")
	}
}
