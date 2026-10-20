package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

var cliFlags = []string{
	"--lang",
	"--ascii",
	"--approx",
	"--latex",
	"--typst",
	"--pretty",
	"--deg",
	"--explain",
	"--format",
	"--verify",
	"--certificate",
	"--replay",
	"--lean",
	"--lean-file",
	"--theme",
	"--no-color",
	"--no-update-check",
	"--warn-loop",
	"--max-loop",
	"-h",
	"--help",
	"-v",
	"--version",
}

var subCommands = []string{
	"run",
	"completion",
}

var reservedConstants = []string{
	"pi",
	"e",
	"i",
	"ans",
}

type completionData struct {
	Flags       []string
	Subcommands []string
	Functions   []string
	Constants   []string
}

// SupportedShells returns the list of shells supported for completion script generation.
func SupportedShells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}

// RunCompletion generates tab-completion script for the target shell and writes it to w.
func RunCompletion(shell string, w io.Writer) error {
	shell = strings.ToLower(strings.TrimSpace(shell))

	funcs := calc.AllFunctionNames()
	data := completionData{
		Flags:       cliFlags,
		Subcommands: subCommands,
		Functions:   funcs,
		Constants:   reservedConstants,
	}

	var tmplStr string
	switch shell {
	case "bash":
		tmplStr = bashTemplate
	case "zsh":
		tmplStr = zshTemplate
	case "fish":
		tmplStr = fishTemplate
	case "powershell":
		tmplStr = powershellTemplate
	default:
		return fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(SupportedShells(), ", "))
	}

	tmpl, err := template.New(shell).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse completion template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute completion template: %w", err)
	}

	_, err = io.Copy(w, &buf)
	return err
}

const bashTemplate = `# bash completion for ihd                                  -*- shell-script -*-

_ihd_completion() {
    local cur prev words cword
    _init_completion -n = 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    }

    local subcommands="{{range .Subcommands}}{{.}} {{end}}"
    local flags="{{range .Flags}}{{.}} {{end}}"
    local funcs="{{range .Functions}}{{.}} {{end}}"
    local constants="{{range .Constants}}{{.}} {{end}}"

    case "${prev}" in
        --format)
            COMPREPLY=( $(compgen -W "typst latex ascii pretty" -- "${cur}") )
            return 0
            ;;
        --theme)
            COMPREPLY=( $(compgen -W "dark light none" -- "${cur}") )
            return 0
            ;;
        --lang)
            COMPREPLY=( $(compgen -W "en ja" -- "${cur}") )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- "${cur}") )
            return 0
            ;;
        --certificate|--replay|--lean-file|run)
            _filedir 2>/dev/null || COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${flags}" -- "${cur}") )
        return 0
    fi

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${subcommands} ${flags} ${funcs} ${constants}" -- "${cur}") )
        return 0
    fi

    COMPREPLY=( $(compgen -W "${funcs} ${constants}" -- "${cur}") )
    return 0
}

complete -F _ihd_completion ihd
`

const zshTemplate = `#compdef ihd

_ihd() {
    local -a subcommands
    subcommands=(
        'run:Execute calculations from an ihd script file'
        'completion:Generate shell completion script'
    )

    local -a formats
    formats=('typst:Typst mathematical formatting' 'latex:LaTeX math formatting' 'ascii:Pure ASCII notation' 'pretty:2D Unicode pretty print')

    local -a themes
    themes=('dark:Dark terminal color theme' 'light:Light terminal color theme' 'none:Disable syntax color')

    local -a langs
    langs=('en:English' 'ja:Japanese')

    local -a shells
    shells=('bash:Bash completion' 'zsh:Zsh completion' 'fish:Fish completion' 'powershell:PowerShell completion')

    local -a functions
    functions=({{range .Functions}}'{{.}}' {{end}})

    _arguments -C \
        '(-h --help)'{-h,--help}'[Show help message]' \
        '(-v --version)'{-v,--version}'[Show version]' \
        '--lang[Set locale]:language:(en ja)' \
        '--format[Output format]:format:(typst latex ascii pretty)' \
        '--theme[REPL theme]:theme:(dark light none)' \
        '--ascii[ASCII output]' \
        '--approx[Show decimal approximation]' \
        '--latex[LaTeX output]' \
        '--typst[Typst output]' \
        '--pretty[2D pretty print]' \
        '--deg[Trigonometric degree mode]' \
        '--explain[Step-by-step derivation]' \
        '--verify[Algebraic verification]' \
        '--certificate[Save proof certificate]:file:_files' \
        '--replay[Replay proof certificate]:file:_files' \
        '--lean[Lean 4 proof generation]' \
        '--lean-file[Save Lean 4 proof]:file:_files' \
        '--no-color[Disable colors]' \
        '--no-update-check[Disable update check]' \
        '--warn-loop[Iteration warning limit]:limit:' \
        '--max-loop[Iteration maximum limit]:limit:' \
        '1: :->cmd' \
        '*:: :->args'

    case $state in
        cmd)
            _describe -t subcommands 'subcommands' subcommands
            _describe -t functions 'builtin functions' functions
            ;;
        args)
            case $words[1] in
                completion)
                    _describe -t shells 'supported shells' shells
                    ;;
                run)
                    _files
                    ;;
            esac
            ;;
    esac
}

_ihd "$@"
`

const fishTemplate = `# fish completion for ihd

# Disable file completion by default
complete -c ihd -f

# Subcommands
complete -c ihd -n "__fish_use_subcommand" -a "run" -d "Execute script file"
complete -c ihd -n "__fish_use_subcommand" -a "completion" -d "Generate completion script"

# Shell choices for completion
complete -c ihd -n "__fish_seen_subcommand_from completion" -a "bash zsh fish powershell" -d "Target shell"

# Global options
complete -c ihd -s h -l help -d "Show help"
complete -c ihd -s v -l version -d "Show version"
complete -c ihd -l lang -x -a "en ja" -d "Language locale"
complete -c ihd -l format -x -a "typst latex ascii pretty" -d "Output format"
complete -c ihd -l theme -x -a "dark light none" -d "REPL syntax theme"
complete -c ihd -l ascii -d "Output ASCII only"
complete -c ihd -l approx -d "Display decimal approximation"
complete -c ihd -l latex -d "Output in LaTeX format"
complete -c ihd -l typst -d "Output in Typst format"
complete -c ihd -l pretty -d "Output in 2D pretty print"
complete -c ihd -l deg -d "Degree mode"
complete -c ihd -l explain -d "Step-by-step algebraic explanation"
complete -c ihd -l verify -d "Autonomously verify algebraic correctness"
complete -c ihd -l certificate -r -d "Save proof certificate JSON"
complete -c ihd -l replay -r -d "Replay proof certificate JSON"
complete -c ihd -l lean -d "Output Lean 4 formal proof"
complete -c ihd -l lean-file -r -d "Save Lean 4 proof file"
complete -c ihd -l no-color -d "Disable syntax colors"
complete -c ihd -l no-update-check -d "Disable update check"
complete -c ihd -l warn-loop -d "Loop step warning limit"
complete -c ihd -l max-loop -d "Loop maximum step limit"

# Built-in functions & constants
{{range .Functions}}complete -c ihd -n "__fish_use_subcommand" -a "{{.}}" -d "Builtin function"
{{end}}
{{range .Constants}}complete -c ihd -n "__fish_use_subcommand" -a "{{.}}" -d "Mathematical constant"
{{end}}
`

const powershellTemplate = `using namespace System.Management.Automation
using namespace System.Management.Automation.Language

Register-ArgumentCompleter -Native -CommandName 'ihd' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $subcommands = @(
        [CompletionResult]::new('run', 'run', [CompletionResultType]::Command, 'Execute script file')
        [CompletionResult]::new('completion', 'completion', [CompletionResultType]::Command, 'Generate shell completion script')
    )

    $flags = @(
        [CompletionResult]::new('--format', '--format', [CompletionResultType]::ParameterName, 'Specify output format (typst, latex, ascii, pretty)')
        [CompletionResult]::new('--approx', '--approx', [CompletionResultType]::ParameterName, 'Show approximate decimal value')
        [CompletionResult]::new('--ascii', '--ascii', [CompletionResultType]::ParameterName, 'Output ASCII characters only')
        [CompletionResult]::new('--latex', '--latex', [CompletionResultType]::ParameterName, 'Output expression in LaTeX')
        [CompletionResult]::new('--typst', '--typst', [CompletionResultType]::ParameterName, 'Output expression in Typst')
        [CompletionResult]::new('--pretty', '--pretty', [CompletionResultType]::ParameterName, 'Output expression in 2D pretty-printed format')
        [CompletionResult]::new('--deg', '--deg', [CompletionResultType]::ParameterName, 'Use degree mode')
        [CompletionResult]::new('--explain', '--explain', [CompletionResultType]::ParameterName, 'Show step-by-step algebraic derivation')
        [CompletionResult]::new('--verify', '--verify', [CompletionResultType]::ParameterName, 'Verify algebraic correctness')
        [CompletionResult]::new('--certificate', '--certificate', [CompletionResultType]::ParameterName, 'Save proof certificate JSON')
        [CompletionResult]::new('--replay', '--replay', [CompletionResultType]::ParameterName, 'Replay proof certificate JSON')
        [CompletionResult]::new('--lean', '--lean', [CompletionResultType]::ParameterName, 'Output Lean 4 formal proof')
        [CompletionResult]::new('--lean-file', '--lean-file', [CompletionResultType]::ParameterName, 'Save Lean 4 code to file')
        [CompletionResult]::new('--theme', '--theme', [CompletionResultType]::ParameterName, 'Set REPL theme (dark, light, none)')
        [CompletionResult]::new('--lang', '--lang', [CompletionResultType]::ParameterName, 'Set locale (en, ja)')
        [CompletionResult]::new('--no-color', '--no-color', [CompletionResultType]::ParameterName, 'Disable syntax highlighting')
        [CompletionResult]::new('--no-update-check', '--no-update-check', [CompletionResultType]::ParameterName, 'Disable update check')
        [CompletionResult]::new('--warn-loop', '--warn-loop', [CompletionResultType]::ParameterName, 'Iteration warning limit')
        [CompletionResult]::new('--max-loop', '--max-loop', [CompletionResultType]::ParameterName, 'Iteration max limit')
        [CompletionResult]::new('-h', '-h', [CompletionResultType]::ParameterName, 'Show help')
        [CompletionResult]::new('--help', '--help', [CompletionResultType]::ParameterName, 'Show help')
        [CompletionResult]::new('-v', '-v', [CompletionResultType]::ParameterName, 'Show version')
        [CompletionResult]::new('--version', '--version', [CompletionResultType]::ParameterName, 'Show version')
    )

    $functions = @(
{{range .Functions}}        [CompletionResult]::new('{{.}}', '{{.}}', [CompletionResultType]::Method, 'Builtin function')
{{end}}    )

    $constants = @(
{{range .Constants}}        [CompletionResult]::new('{{.}}', '{{.}}', [CompletionResultType]::Variable, 'Constant')
{{end}}    )

    $elements = $commandAst.CommandElements
    $prev = if ($elements.Count -gt 1) { $elements[-2].Extent.Text } else { "" }

    if ($prev -eq '--format') {
        return @('typst', 'latex', 'ascii', 'pretty') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "Format: $_")
        }
    }
    if ($prev -eq '--theme') {
        return @('dark', 'light', 'none') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "Theme: $_")
        }
    }
    if ($prev -eq '--lang') {
        return @('en', 'ja') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "Language: $_")
        }
    }
    if ($prev -eq 'completion') {
        return @('bash', 'zsh', 'fish', 'powershell') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "Shell: $_")
        }
    }

    if ($wordToComplete -like "-*") {
        return $flags | Where-Object { $_.CompletionText -like "$wordToComplete*" }
    }

    $all = @()
    $all += $subcommands
    $all += $flags
    $all += $functions
    $all += $constants

    return $all | Where-Object { $_.CompletionText -like "$wordToComplete*" }
}
`
