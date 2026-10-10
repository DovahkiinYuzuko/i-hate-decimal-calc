package calc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// CanonicalizeJSON converts arbitrary JSON bytes into RFC 8785 (JSON Canonicalization Scheme: JCS) format.
func CanonicalizeJSON(input []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()

	var val interface{}
	if err := dec.Decode(&val); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("proof_trace.err_invalid_certificate_json"), err)
	}

	// Ensure no trailing tokens exist
	if dec.More() {
		return nil, fmt.Errorf("%s: trailing data in JSON", i18n.T("proof_trace.err_invalid_certificate_json"))
	}

	var buf bytes.Buffer
	if err := serializeCanonical(val, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CanonicalHashSHA256 canonicalizes the input JSON using RFC 8785 and computes its SHA-256 hex digest.
func CanonicalHashSHA256(input []byte) (string, error) {
	canonical, err := CanonicalizeJSON(input)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(canonical)
	return hex.EncodeToString(h[:]), nil
}

func serializeCanonical(v interface{}, w io.Writer) error {
	if v == nil {
		_, err := w.Write([]byte("null"))
		return err
	}

	switch val := v.(type) {
	case bool:
		if val {
			_, err := w.Write([]byte("true"))
			return err
		}
		_, err := w.Write([]byte("false"))
		return err

	case json.Number:
		return serializeCanonicalNumber(val, w)

	case string:
		return serializeCanonicalString(val, w)

	case []interface{}:
		return serializeCanonicalArray(val, w)

	case map[string]interface{}:
		return serializeCanonicalObject(val, w)

	default:
		return fmt.Errorf("jcs: unsupported data type %T", v)
	}
}

// serializeCanonicalNumber serializes JSON numbers per RFC 8785 (ECMAScript 7.1.12.1 ToString applied to Number).
func serializeCanonicalNumber(num json.Number, w io.Writer) error {
	s := num.String()
	// If it's a pure integer, emit it as-is
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		_, writeErr := fmt.Fprintf(w, "%d", i)
		return writeErr
	}
	if u, err := strconv.ParseUint(s, 10, 64); err == nil {
		_, writeErr := fmt.Fprintf(w, "%d", u)
		return writeErr
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("jcs: invalid number %q: %w", s, err)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("jcs: NaN and Infinity are not permitted in RFC 8785 JSON")
	}
	// -0 must be serialized as 0
	if f == 0.0 {
		_, writeErr := w.Write([]byte("0"))
		return writeErr
	}

	// ECMAScript-compatible float formatting: standard strconv.FormatFloat 'g', -1, 64
	formatted := strconv.FormatFloat(f, 'g', -1, 64)
	_, writeErr := w.Write([]byte(formatted))
	return writeErr
}

// serializeCanonicalString serializes strings per RFC 8785 section 3.2.2.2.
func serializeCanonicalString(s string, w io.Writer) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("jcs: invalid UTF-8 string")
	}

	var buf bytes.Buffer
	buf.WriteByte('"')

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return fmt.Errorf("jcs: invalid UTF-8 rune")
		}

		// Reject lone surrogate runes if any
		if r >= 0xD800 && r <= 0xDFFF {
			return fmt.Errorf("jcs: lone surrogates are forbidden in RFC 8785")
		}

		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
		i += size
	}

	buf.WriteByte('"')
	_, err := w.Write(buf.Bytes())
	return err
}

func serializeCanonicalArray(arr []interface{}, w io.Writer) error {
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}
	for i, elem := range arr {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := serializeCanonical(elem, w); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("]"))
	return err
}

func serializeCanonicalObject(obj map[string]interface{}, w io.Writer) error {
	if _, err := w.Write([]byte("{")); err != nil {
		return err
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}

	// RFC 8785: Keys must be sorted by UTF-16 code units
	sort.Slice(keys, func(i, j int) bool {
		return compareUTF16(keys[i], keys[j]) < 0
	})

	for i, k := range keys {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := serializeCanonicalString(k, w); err != nil {
			return err
		}
		if _, err := w.Write([]byte(":")); err != nil {
			return err
		}
		if err := serializeCanonical(obj[k], w); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte("}"))
	return err
}

// compareUTF16 compares two UTF-8 strings by their UTF-16 code units per RFC 8785 section 3.2.3.
func compareUTF16(a, b string) int {
	u16A := utf16.Encode([]rune(a))
	u16B := utf16.Encode([]rune(b))

	minLen := len(u16A)
	if len(u16B) < minLen {
		minLen = len(u16B)
	}

	for i := 0; i < minLen; i++ {
		if u16A[i] < u16B[i] {
			return -1
		}
		if u16A[i] > u16B[i] {
			return 1
		}
	}

	if len(u16A) < len(u16B) {
		return -1
	}
	if len(u16A) > len(u16B) {
		return 1
	}
	return 0
}
