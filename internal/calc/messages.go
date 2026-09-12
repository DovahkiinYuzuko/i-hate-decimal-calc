package calc

// System messages and UI strings externalized for internationalization.
const (
	PromptREPL     = "ihd> "
	MsgREPLWelcome = "ihd: Exact Arithmetic Calculator\nType 'exit' or 'quit' to exit."
	MsgREPLExit    = "Goodbye."
	MsgErrorPrefix     = "Error: "
	MsgNoVars          = "No variables defined."
	MsgErrReservedWord = "cannot assign to reserved identifier %q"

	MsgErrProbOutOfRange      = "probability must be between 0 and 1"
	MsgErrProbNonNegativeInt  = "number of trials and successes must be non-negative integers"
	MsgErrProbPositiveInt     = "trial count must be a positive integer"
	MsgErrHyperParams         = "invalid hypergeometric distribution parameters"
	MsgErrBayesZeroEvidence   = "marginal probability (evidence) cannot be zero"
	MsgErrInvalidDistribution = "invalid or unsupported distribution specification"
	MsgErrProbSumNotOne       = "sum of probabilities in discrete distribution must equal 1"

	MsgErrPlotInvalidDomain     = "invalid plot domain: lower bound must be less than upper bound"
	MsgErrPlotVariableNotFound  = "plot requires an expression with a single variable"
	MsgErrPlotEmptyDomain       = "plot domain cannot be empty"

	MsgHelp = `ihd - Exact Arithmetic Calculator

Usage:
  ihd [flags] [expression]
  ihd [flags]                (Starts interactive REPL)
  echo "expr" | ihd [flags]   (Evaluates piped input)

Flags:
  --ascii     Output using standard ASCII text (e.g., sqrt, pi) instead of Unicode
  --approx    Display approximate decimal value alongside exact form
  --latex     Output expression in LaTeX format ($$ ... $$)
  --pretty    Output expression using 2D pretty-printed formatting
  --deg       Use degree mode for trigonometric and inverse trigonometric functions
  --explain   Show step-by-step algebraic rewriting explanations
  -h, --help  Show this help message`
)
