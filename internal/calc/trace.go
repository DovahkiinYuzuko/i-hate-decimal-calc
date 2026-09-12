package calc

import (
	"fmt"
	"strings"
	"sync"
)

// RuleID identifies a specific algebraic term-rewriting rule or algorithm.
type RuleID string

const (
	RuleRationalize    RuleID = "Rationalize"
	RuleDenestRadical  RuleID = "DenestRadical"
	RuleSolveQuadratic RuleID = "SolveQuadratic"
	RuleSolveLinear    RuleID = "SolveLinear"
	RuleDiffPower      RuleID = "DiffPower"
	RuleDiffProduct    RuleID = "DiffProduct"
	RuleDiffChain      RuleID = "DiffChain"
	RuleDiffTrig       RuleID = "DiffTrig"
	RuleTaylor         RuleID = "TaylorSeries"
	RuleSumFaulhaber   RuleID = "SumFaulhaber"
	RuleMatrixDet      RuleID = "MatrixDet"
	RuleMatrixInv      RuleID = "MatrixInv"
)

// RewriteEvent records an atomic term-rewriting step.
type RewriteEvent struct {
	Rule   RuleID
	Before Node
	After  Node
	Note   string
}

// PedagogicalStep represents a user-facing explanation step.
type PedagogicalStep struct {
	StepNumber  int
	Rule        RuleID
	Title       string
	Before      Node
	After       Node
	Explanation string
}

// TraceSink defines an observer that receives rewrite events.
type TraceSink interface {
	RecordRewrite(rule RuleID, before Node, after Node, note string)
}

// MemoryTraceSink buffers rewrite events in memory.
type MemoryTraceSink struct {
	mu     sync.Mutex
	events []RewriteEvent
}

// NewMemoryTraceSink creates a new in-memory trace sink.
func NewMemoryTraceSink() *MemoryTraceSink {
	return &MemoryTraceSink{
		events: make([]RewriteEvent, 0),
	}
}

// RecordRewrite records a rewrite event into the sink.
func (s *MemoryTraceSink) RecordRewrite(rule RuleID, before Node, after Node, note string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, RewriteEvent{
		Rule:   rule,
		Before: before,
		After:  after,
		Note:   note,
	})
}

// Events returns a slice copy of recorded events.
func (s *MemoryTraceSink) Events() []RewriteEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RewriteEvent, len(s.events))
	copy(out, s.events)
	return out
}

var (
	traceMu          sync.RWMutex
	currentTraceSink TraceSink
)

// WithTraceSink executes fn with the given sink set as the active trace sink.
func WithTraceSink(sink TraceSink, fn func()) {
	traceMu.Lock()
	old := currentTraceSink
	currentTraceSink = sink
	traceMu.Unlock()

	defer func() {
		traceMu.Lock()
		currentTraceSink = old
		traceMu.Unlock()
	}()

	fn()
}

// RecordTraceRewrite records a rewrite event to the active sink if one is registered.
func RecordTraceRewrite(rule RuleID, before Node, after Node, note string) {
	traceMu.RLock()
	sink := currentTraceSink
	traceMu.RUnlock()

	if sink != nil {
		sink.RecordRewrite(rule, before, after, note)
	}
}

// ruleTitles maps RuleIDs to human-friendly step titles.
var ruleTitles = map[RuleID]string{
	RuleRationalize:    "分母の有理化",
	RuleDenestRadical:  "二重根号の簡約",
	RuleSolveQuadratic: "2次方程式の求解（解の公式）",
	RuleSolveLinear:    "1次方程式の求解",
	RuleDiffPower:      "べき乗の微分公式の適用",
	RuleDiffProduct:    "積の微分公式の適用",
	RuleDiffChain:      "合成関数の微分（チェインルール）",
	RuleDiffTrig:       "三角関数の微分公式の適用",
	RuleTaylor:         "テイラー級数展開の計算",
	RuleSumFaulhaber:   "Faulhaberの公式によるべき乗和の閉形式導出",
	RuleMatrixDet:      "余因子展開による行列式の計算",
	RuleMatrixInv:      "余因子行列による逆行列の計算",
}

// CompressTrace converts low-level rewrite events into user-facing pedagogical steps.
func CompressTrace(events []RewriteEvent) []PedagogicalStep {
	steps := make([]PedagogicalStep, 0, len(events))
	stepNum := 1

	for _, ev := range events {
		title, ok := ruleTitles[ev.Rule]
		if !ok {
			title = string(ev.Rule)
		}

		steps = append(steps, PedagogicalStep{
			StepNumber:  stepNum,
			Rule:        ev.Rule,
			Title:       title,
			Before:      ev.Before,
			After:       ev.After,
			Explanation: ev.Note,
		})
		stepNum++
	}

	return steps
}

// FormatTrace renders the pedagogical steps and final result into a 2D Unicode tree.
func FormatTrace(inputExpr string, steps []PedagogicalStep, result Node) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("式: %s\n", inputExpr))

	if len(steps) == 0 {
		b.WriteString("├── [Step 0: 単純計算]\n")
		b.WriteString("│   式は直接簡約されました\n")
		b.WriteString(fmt.Sprintf("└── [Result]\n    = %s\n", Format(result)))
		return b.String()
	}

	for i, step := range steps {
		b.WriteString(fmt.Sprintf("├── [Step %d: %s]\n", step.StepNumber, step.Title))
		if step.Explanation != "" {
			b.WriteString(fmt.Sprintf("│   %s\n", step.Explanation))
		}
		if step.Before != nil && step.After != nil {
			b.WriteString(fmt.Sprintf("│   %s  ──>  %s\n", Format(step.Before), Format(step.After)))
		}
		if i < len(steps)-1 {
			b.WriteString("│\n")
		}
	}

	b.WriteString("└── [Result]\n")
	b.WriteString(fmt.Sprintf("    = %s\n", Format(result)))

	return b.String()
}

// EvalWithTrace evaluates an AST node while recording rewrite events into pedagogical steps.
func EvalWithTrace(node Node) (Node, []PedagogicalStep, error) {
	return EvalWithTraceAndEnv(node, nil)
}

// EvalWithTraceAndEnv evaluates an AST node in an environment while recording rewrite events.
func EvalWithTraceAndEnv(node Node, env *Env) (Node, []PedagogicalStep, error) {
	sink := NewMemoryTraceSink()
	var res Node
	var err error
	WithTraceSink(sink, func() {
		if env != nil {
			res, err = EvalWithEnv(node, env)
		} else {
			res, err = Eval(node)
		}
	})
	if err != nil {
		return nil, nil, err
	}
	steps := CompressTrace(sink.Events())
	return res, steps, nil
}
