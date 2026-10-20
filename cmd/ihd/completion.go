package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

type completionItem struct {
	Name string
	Desc string
}

type completionData struct {
	Flags       []completionItem
	Subcommands []completionItem
	Functions   []completionItem
	Constants   []completionItem
	Labels      map[string]string
}

// SupportedShells returns the list of shells supported for completion script generation.
func SupportedShells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}

// RunCompletion generates tab-completion script for the target shell and writes it to w.
func RunCompletion(shell string, w io.Writer) error {
	shell = strings.ToLower(strings.TrimSpace(shell))

	flagDefinitions := []struct {
		name    string
		i18nKey string
	}{
		{"--lang", "cli.flag_lang"},
		{"--ascii", "cli.flag_ascii"},
		{"--approx", "cli.flag_approx"},
		{"--latex", "cli.flag_latex"},
		{"--typst", "cli.flag_typst"},
		{"--pretty", "cli.flag_pretty"},
		{"--deg", "cli.flag_deg"},
		{"--explain", "cli.flag_explain"},
		{"--format", "cli.flag_format"},
		{"--verify", "cli.flag_verify"},
		{"--certificate", "cli.flag_certificate"},
		{"--replay", "cli.flag_replay"},
		{"--lean", "cli.flag_lean"},
		{"--lean-file", "cli.flag_lean_file"},
		{"--theme", "cli.flag_theme"},
		{"--no-color", "cli.flag_no_color"},
		{"--no-update-check", "cli.flag_no_update_check"},
		{"--warn-loop", "cli.flag_warn_loop"},
		{"--max-loop", "cli.flag_max_loop"},
		{"-h", "cli.flag_help"},
		{"--help", "cli.flag_help"},
		{"-v", "cli.flag_version"},
		{"--version", "cli.flag_version"},
	}

	var flags []completionItem
	for _, f := range flagDefinitions {
		flags = append(flags, completionItem{
			Name: f.name,
			Desc: i18n.T(f.i18nKey),
		})
	}

	subcommandDefinitions := []struct {
		name    string
		i18nKey string
	}{
		{"run", "cli.subcmd_run"},
		{"completion", "cli.subcmd_completion"},
	}

	var subcommands []completionItem
	for _, s := range subcommandDefinitions {
		subcommands = append(subcommands, completionItem{
			Name: s.name,
			Desc: i18n.T(s.i18nKey),
		})
	}

	constantDefinitions := []struct {
		name    string
		i18nKey string
	}{
		{"pi", "cli.const_pi"},
		{"e", "cli.const_e"},
		{"i", "cli.const_i"},
		{"ans", "cli.const_ans"},
	}

	var constants []completionItem
	for _, c := range constantDefinitions {
		constants = append(constants, completionItem{
			Name: c.name,
			Desc: i18n.T(c.i18nKey),
		})
	}

	defaultFnDesc := i18n.T("cli.completion_builtin_func")
	var functions []completionItem
	for _, fn := range calc.AllFunctionNames() {
		fnDesc := i18n.T("fn." + fn)
		if fnDesc == "fn."+fn || fnDesc == "" {
			fnDesc = defaultFnDesc
		}
		functions = append(functions, completionItem{
			Name: fn,
			Desc: fnDesc,
		})
	}

	labels := map[string]string{
		"Format": i18n.T("cli.completion_label_format"),
		"Theme":  i18n.T("cli.completion_label_theme"),
		"Lang":   i18n.T("cli.completion_label_lang"),
		"Shell":  i18n.T("cli.completion_label_shell"),
	}

	data := completionData{
		Flags:       flags,
		Subcommands: subcommands,
		Functions:   functions,
		Constants:   constants,
		Labels:      labels,
	}

	funcMap := template.FuncMap{
		"escapePs": func(s string) string {
			return strings.ReplaceAll(s, "'", "''")
		},
		"escapeFish": func(s string) string {
			s = strings.ReplaceAll(s, "\\", "\\\\")
			return strings.ReplaceAll(s, "\"", "\\\"")
		},
		"escapeZsh": func(s string) string {
			s = strings.ReplaceAll(s, ":", "\\:")
			s = strings.ReplaceAll(s, "[", "\\[")
			s = strings.ReplaceAll(s, "]", "\\]")
			return strings.ReplaceAll(s, "'", "\\'")
		},
		"trimPrefix": strings.TrimPrefix,
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

	tmpl, err := template.New(shell).Funcs(funcMap).Parse(tmplStr)
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

    local subcommands="{{range .Subcommands}}{{.Name}} {{end}}"
    local flags="{{range .Flags}}{{.Name}} {{end}}"
    local funcs="{{range .Functions}}{{.Name}} {{end}}"
    local constants="{{range .Constants}}{{.Name}} {{end}}"

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
{{range .Subcommands}}        '{{.Name}}:{{escapeZsh .Desc}}'
{{end}}    )

    local -a formats
    formats=('typst' 'latex' 'ascii' 'pretty')

    local -a themes
    themes=('dark' 'light' 'none')

    local -a langs
    langs=('en' 'ja')

    local -a shells
    shells=('bash' 'zsh' 'fish' 'powershell')

    local -a functions
    functions=(
{{range .Functions}}        '{{.Name}}:{{escapeZsh .Desc}}'
{{end}}    )

    local -a constants
    constants=(
{{range .Constants}}        '{{.Name}}:{{escapeZsh .Desc}}'
{{end}}    )

    _arguments -C \
        '(-h --help)'{-h,--help}'[{{escapeZsh (index .Flags 19).Desc}}]' \
        '(-v --version)'{-v,--version}'[{{escapeZsh (index .Flags 21).Desc}}]' \
        '--lang[{{escapeZsh (index .Flags 0).Desc}}]:language:(en ja)' \
        '--format[{{escapeZsh (index .Flags 8).Desc}}]:format:(typst latex ascii pretty)' \
        '--theme[{{escapeZsh (index .Flags 14).Desc}}]:theme:(dark light none)' \
        '--ascii[{{escapeZsh (index .Flags 1).Desc}}]' \
        '--approx[{{escapeZsh (index .Flags 2).Desc}}]' \
        '--latex[{{escapeZsh (index .Flags 3).Desc}}]' \
        '--typst[{{escapeZsh (index .Flags 4).Desc}}]' \
        '--pretty[{{escapeZsh (index .Flags 5).Desc}}]' \
        '--deg[{{escapeZsh (index .Flags 6).Desc}}]' \
        '--explain[{{escapeZsh (index .Flags 7).Desc}}]' \
        '--verify[{{escapeZsh (index .Flags 9).Desc}}]' \
        '--certificate[{{escapeZsh (index .Flags 10).Desc}}]:file:_files' \
        '--replay[{{escapeZsh (index .Flags 11).Desc}}]:file:_files' \
        '--lean[{{escapeZsh (index .Flags 12).Desc}}]' \
        '--lean-file[{{escapeZsh (index .Flags 13).Desc}}]:file:_files' \
        '--no-color[{{escapeZsh (index .Flags 15).Desc}}]' \
        '--no-update-check[{{escapeZsh (index .Flags 16).Desc}}]' \
        '--warn-loop[{{escapeZsh (index .Flags 17).Desc}}]:limit:' \
        '--max-loop[{{escapeZsh (index .Flags 18).Desc}}]:limit:' \
        '1: :->cmd' \
        '*:: :->args'

    case $state in
        cmd)
            _describe -t subcommands 'subcommands' subcommands
            _describe -t functions 'builtin functions' functions
            _describe -t constants 'constants' constants
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
{{range .Subcommands}}complete -c ihd -n "__fish_use_subcommand" -a "{{.Name}}" -d "{{escapeFish .Desc}}"
{{end}}
# Shell choices for completion
complete -c ihd -n "__fish_seen_subcommand_from completion" -a "bash zsh fish powershell"

# Argument files for run
complete -c ihd -n "__fish_seen_subcommand_from run" -F

# Flags
{{range .Flags}}complete -c ihd -l "{{trimPrefix .Name "--"}}" -d "{{escapeFish .Desc}}"
{{end}}
# Flag choices
complete -c ihd -l format -a "typst latex ascii pretty"
complete -c ihd -l theme -a "dark light none"
complete -c ihd -l lang -a "en ja"

# Built-in functions
{{range .Functions}}complete -c ihd -a "{{.Name}}" -d "{{escapeFish .Desc}}"
{{end}}
# Constants
{{range .Constants}}complete -c ihd -a "{{.Name}}" -d "{{escapeFish .Desc}}"
{{end}}`

const powershellTemplate = `using namespace System.Management.Automation
using namespace System.Management.Automation.Language

Register-ArgumentCompleter -Native -CommandName 'ihd' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $subcommands = @(
{{range .Subcommands}}        [CompletionResult]::new('{{.Name}}', '{{.Name}}', [CompletionResultType]::Command, '{{escapePs .Desc}}')
{{end}}    )

    $flags = @(
{{range .Flags}}        [CompletionResult]::new('{{.Name}}', '{{.Name}}', [CompletionResultType]::ParameterName, '{{escapePs .Desc}}')
{{end}}    )

    $functions = @(
{{range .Functions}}        [CompletionResult]::new('{{.Name}}', '{{.Name}}', [CompletionResultType]::Method, '{{escapePs .Desc}}')
{{end}}    )

    $constants = @(
{{range .Constants}}        [CompletionResult]::new('{{.Name}}', '{{.Name}}', [CompletionResultType]::Variable, '{{escapePs .Desc}}')
{{end}}    )

    $elements = $commandAst.CommandElements
    $prev = if ($elements.Count -gt 1) { $elements[-2].Extent.Text } else { "" }

    if ($prev -eq '--format') {
        return @('typst', 'latex', 'ascii', 'pretty') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "{{.Labels.Format}}$_")
        }
    }
    if ($prev -eq '--theme') {
        return @('dark', 'light', 'none') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "{{.Labels.Theme}}$_")
        }
    }
    if ($prev -eq '--lang') {
        return @('en', 'ja') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "{{.Labels.Lang}}$_")
        }
    }
    if ($prev -eq 'completion') {
        return @('bash', 'zsh', 'fish', 'powershell') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, "{{.Labels.Shell}}$_")
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
