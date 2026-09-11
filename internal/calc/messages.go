package calc

// System messages and UI strings externalized for internationalization.
const (
	PromptREPL     = "ihd> "
	MsgREPLWelcome = "ihd: Exact Arithmetic Calculator\nType 'exit' or 'quit' to exit."
	MsgREPLExit    = "Goodbye."
	MsgErrorPrefix = "Error: "

	MsgHelp = `ihd - Exact Arithmetic Calculator

Usage:
  ihd [flags] [expression]
  ihd [flags]                (Starts interactive REPL)
  echo "expr" | ihd [flags]   (Evaluates piped input)

Flags:
  --ascii     Output using standard ASCII text (e.g., sqrt, pi) instead of Unicode
  --approx    Display approximate decimal value alongside exact form
  -h, --help  Show this help message`
)
