package main

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/nyaosorg/go-readline-ny"
)

// ExtractWordBeforeCursor extracts the identifier immediately preceding cursor and its start index.
func ExtractWordBeforeCursor(line string, cursor int) (string, int) {
	if cursor <= 0 || cursor > len(line) {
		return "", cursor
	}

	start := cursor
	for start > 0 {
		r := rune(line[start-1])
		if !isIdentRune(r) {
			break
		}
		start--
	}

	if start == cursor {
		return "", cursor
	}
	return line[start:cursor], start
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// LongestCommonPrefix returns the longest common prefix among strings.
func LongestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

// FindCompletionCandidates returns sorted matching identifiers for the given prefix.
func FindCompletionCandidates(prefix string, env *calc.Env) []string {
	if prefix == "" {
		return nil
	}

	lowerPrefix := strings.ToLower(prefix)
	seen := make(map[string]struct{})
	var candidates []string

	addMatch := func(name string) {
		if strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			if _, exists := seen[name]; !exists {
				seen[name] = struct{}{}
				candidates = append(candidates, name)
			}
		}
	}

	// 1. Built-in functions
	for _, fn := range calc.AllFunctionNames() {
		addMatch(fn)
	}

	// 2. Built-in constants
	constants := []string{"pi", "e", "i", "ans"}
	for _, c := range constants {
		addMatch(c)
	}

	// 3. User session variables from env
	if env != nil {
		for v := range env.All() {
			addMatch(v)
		}
	}

	// 4. REPL commands
	replCommands := []string{"vars", "theme", "highlight", "exit", "quit"}
	for _, cmd := range replCommands {
		addMatch(cmd)
	}

	sort.Strings(candidates)
	return candidates
}

// CompletionFSM manages stateful cycling across Tab keystrokes.
type CompletionFSM struct {
	env        *calc.Env
	active     bool
	wordStart  int
	lastWord   string
	prefix     string
	candidates []string
	index      int
}

// NewCompletionFSM creates an initialized CompletionFSM.
func NewCompletionFSM(env *calc.Env) *CompletionFSM {
	return &CompletionFSM{
		env: env,
	}
}

// Reset clears the cycling state.
func (f *CompletionFSM) Reset() {
	f.active = false
	f.wordStart = 0
	f.lastWord = ""
	f.prefix = ""
	f.candidates = nil
	f.index = 0
}

// Step advances completion at the given wordStart and cursor position.
func (f *CompletionFSM) Step(line string, wordStart int, cursor int) (string, bool) {
	// If currently active and the cursor is right after our last replaced word, cycle to next candidate
	if f.active && len(f.candidates) > 0 && cursor == f.wordStart+len(f.lastWord) && wordStart == f.wordStart {
		f.index = (f.index + 1) % len(f.candidates)
		f.lastWord = f.candidates[f.index]
		return f.lastWord, true
	}

	// Start new completion lookup
	if wordStart < 0 || cursor < wordStart || cursor > len(line) {
		f.Reset()
		return "", false
	}
	currentWord := line[wordStart:cursor]
	if currentWord == "" {
		f.Reset()
		return "", false
	}

	candidates := FindCompletionCandidates(currentWord, f.env)
	if len(candidates) == 0 {
		f.Reset()
		return "", false
	}

	if len(candidates) == 1 {
		f.Reset()
		return candidates[0], true
	}

	// Multiple candidates: check common prefix first
	lcp := LongestCommonPrefix(candidates)
	if len(lcp) > len(currentWord) {
		// Advance up to the common prefix without entering cycling mode yet
		f.Reset()
		return lcp, true
	}

	// Already at or past common prefix: enter cycling mode starting with first candidate
	f.active = true
	f.wordStart = wordStart
	f.prefix = currentWord
	f.candidates = candidates
	f.index = 0
	f.lastWord = candidates[0]
	return f.lastWord, true
}

// NewCompletionCommand returns a readline Command to bind to Tab.
func NewCompletionCommand(fsm *CompletionFSM) readline.Command {
	return readline.NewGoCommand("COMPLETE_OR_CYCLE", func(ctx context.Context, B *readline.Buffer) readline.Result {
		line := B.String()
		cursor := B.Cursor
		word, wordStart := ExtractWordBeforeCursor(line, cursor)
		if word == "" {
			fsm.Reset()
			return readline.CONTINUE
		}

		replacement, ok := fsm.Step(line, wordStart, cursor)
		if ok && replacement != "" {
			B.ReplaceAndRepaint(wordStart, replacement)
		}
		return readline.CONTINUE
	})
}
