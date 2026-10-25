package provenance

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ScannerScanner is an interface satisfied by bufio.Scanner and custom line scanners.
type LineScanner interface {
	Scan() bool
	Text() string
	Err() error
}

// StepExplorer provides an interactive CUI step-by-step walkthrough for derivation trees.
type StepExplorer struct {
	Steps       []*FormattedStep
	Current     int
	Root        *DerivationNode
	In          io.Reader
	LineScanner LineScanner
	Out         io.Writer
}

// NewStepExplorer instantiates a StepExplorer from a root derivation node and an io.Reader.
func NewStepExplorer(root *DerivationNode, in io.Reader, out io.Writer) *StepExplorer {
	visitor := NewExplainVisitor()
	steps := visitor.collectPresentableSteps(root)

	return &StepExplorer{
		Steps:   steps,
		Current: 0,
		Root:    root,
		In:      in,
		Out:     out,
	}
}

// NewStepExplorerWithScanner instantiates a StepExplorer using an existing LineScanner to prevent buffering race conditions.
func NewStepExplorerWithScanner(root *DerivationNode, scanner LineScanner, out io.Writer) *StepExplorer {
	visitor := NewExplainVisitor()
	steps := visitor.collectPresentableSteps(root)

	return &StepExplorer{
		Steps:       steps,
		Current:     0,
		Root:        root,
		LineScanner: scanner,
		Out:         out,
	}
}

// Run starts the interactive navigation loop.
func (e *StepExplorer) Run() error {
	if len(e.Steps) == 0 {
		fmt.Fprintln(e.Out, i18n.T("cli.repl_step_completed"))
		return nil
	}

	scanner := e.LineScanner
	if scanner == nil {
		if e.In == nil {
			fmt.Fprintln(e.Out, i18n.T("cli.repl_step_completed"))
			return nil
		}
		scanner = bufio.NewScanner(e.In)
	}

	for {
		e.renderCurrentStep()

		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(strings.ToLower(scanner.Text()))

		switch cmd {
		case "q", "quit", "exit":
			return nil
		case "b", "back", "p", "prev":
			if e.Current > 0 {
				e.Current--
			}
		case "?", "help":
			fmt.Fprintln(e.Out, i18n.T("cli.repl_step_help"))
		case "t", "typst":
			tv := NewTypstVisitor()
			fmt.Fprintln(e.Out, tv.RenderDocument(e.Root, ""))
		case "l", "lean":
			lv := NewLeanCalcVisitor()
			res, err := lv.RenderCalc(e.Root, "ihd_derivation")
			if err != nil {
				fmt.Fprintf(e.Out, "%s%v\n", i18n.T("cli.error_prefix"), err)
			} else {
				fmt.Fprintln(e.Out, res)
			}
		case "", "n", "next":
			if e.Current < len(e.Steps)-1 {
				e.Current++
			} else {
				fmt.Fprintln(e.Out, i18n.T("cli.repl_step_completed"))
				return nil
			}
		default:
			// Unrecognized command, print hint
			fmt.Fprintln(e.Out, i18n.T("cli.repl_step_nav_hint"))
		}
	}

	return scanner.Err()
}

func (e *StepExplorer) renderCurrentStep() {
	if e.Current < 0 || e.Current >= len(e.Steps) {
		return
	}
	step := e.Steps[e.Current]

	fmt.Fprintln(e.Out)
	fmt.Fprintf(e.Out, i18n.T("cli.repl_step_header")+"\n", e.Current+1, len(e.Steps))
	fmt.Fprintf(e.Out, "  %s\n", fmt.Sprintf(i18n.T("cli.repl_step_rule"), step.Title))
	fmt.Fprintf(e.Out, "  %s\n", fmt.Sprintf(i18n.T("cli.repl_step_relation"), relationSymbol(step.Relation)))
	fmt.Fprintf(e.Out, "  %s\n", fmt.Sprintf(i18n.T("cli.repl_step_before"), step.Before))
	fmt.Fprintf(e.Out, "  %s\n", fmt.Sprintf(i18n.T("cli.repl_step_after"), step.After))

	if len(step.Conditions) > 0 {
		fmt.Fprintf(e.Out, "  Conditions: %s\n", strings.Join(step.Conditions, ", "))
	}
	if step.Certificate != "" {
		fmt.Fprintf(e.Out, "  Certificate: %s\n", step.Certificate)
	}

	fmt.Fprintln(e.Out, i18n.T("cli.repl_step_nav_hint"))
}
