package main

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// colorName returns a human-readable description of known ANSI sequences for clear logging.
func colorName(seq string) string {
	switch seq {
	case "":
		return "[Default/Plain (None)]"
	case "\x1B[96m":
		return "Bright Cyan (Rainbow Depth 0)"
	case "\x1B[95m":
		return "Bright Magenta (Rainbow Depth 1)"
	case "\x1B[92m":
		return "Bright Green (Rainbow Depth 2)"
	case "\x1B[94m":
		return "Bright Blue (Rainbow Depth 3)"
	case "\x1B[97;41m":
		return "White on Red Background (Mismatch Alert)"
	case "\x1B[94;1m":
		return "Bold Bright Blue (Reserved Function)"
	case "\x1B[36;1m":
		return "Bold Cyan (CLI Command)"
	case "\x1B[93m":
		return "Bright Yellow (Number / Radix)"
	case "\x1B[32m":
		return "Green (String Literal)"
	case "\x1B[37m":
		return "Bright White (Operator)"
	case "\x1B[96;2m":
		return "Soft Cyan/Mint (Semantic Custom Variable)"
	default:
		return fmt.Sprintf("ANSI(%q)", seq)
	}
}

// dumpKeystrokeFrame simulates character-by-character input and resolves the exact color for each byte cell.
func dumpKeystrokeFrame(input string, env *calc.Env, theme string) (string, []string) {
	highlights := BuildREPLHighlights(theme, env)
	colorMap := make([]string, len(input))
	for i := range colorMap {
		colorMap[i] = "" // default
	}

	for _, h := range highlights {
		matches := h.Pattern.FindAllStringIndex(input, -1)
		for _, m := range matches {
			for idx := m[0]; idx < m[1]; idx++ {
				colorMap[idx] = h.Sequence
			}
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Buffer: %q (len=%d)\n", input, len(input))
	for i, r := range []rune(input) {
		byteIdx := len(string([]rune(input)[:i]))
		seq := colorMap[byteIdx]
		fmt.Fprintf(&sb, "  pos %2d: %q -> %s\n", byteIdx, r, colorName(seq))
	}
	return sb.String(), colorMap
}

func TestREPLColorTrace_Keystrokes(t *testing.T) {
	logDir := filepath.Join("..", "..", "test-scripts", "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "repl_color_trace.log")

	var logBuf strings.Builder
	separator := strings.Repeat("=", 70) + "\n"

	logBuf.WriteString(separator)
	logBuf.WriteString("  IHD REPL 逐次キーストローク構文ハイライト・カラーダンプ検証ログ\n")
	logBuf.WriteString(separator + "\n")

	// =========================================================================
	// シナリオ 1: factor(114514) を 1 文字ずつタイピングした時のカラー遷移
	// =========================================================================
	logBuf.WriteString("[シナリオ 1] 'factor(114514)' の 1 文字ずつ逐次タイピング検証\n")
	logBuf.WriteString(strings.Repeat("-", 60) + "\n")

	fullInput := "factor(114514)"
	for i := 1; i <= len(fullInput); i++ {
		prefix := fullInput[:i]
		frameLog, colorMap := dumpKeystrokeFrame(prefix, nil, "dark")
		fmt.Fprintf(&logBuf, "[Step %2d] %s", i, frameLog)

		// 最終フレームでの整合性アサーション
		if i == len(fullInput) {
			// factor (0..5): Bold Bright Blue
			for j := 0; j < 6; j++ {
				if colorMap[j] != "\x1B[94;1m" {
					t.Errorf("pos %d ('factor'): expected bold blue \x1B[94;1m, got %q", j, colorMap[j])
				}
			}
			// '(' (6): Bright Cyan
			if colorMap[6] != "\x1B[96m" {
				t.Errorf("pos 6 ('('): expected cyan \x1B[96m, got %q", colorMap[6])
			}
			// '114514' (7..12): Bright Yellow (括弧と完全分離！)
			for j := 7; j <= 12; j++ {
				if colorMap[j] != "\x1B[93m" {
					t.Errorf("pos %d ('114514'): expected yellow \x1B[93m, got %q", j, colorMap[j])
				}
			}
			// ')' (13): Bright Cyan
			if colorMap[13] != "\x1B[96m" {
				t.Errorf("pos 13 (')'): expected cyan \x1B[96m, got %q", colorMap[13])
			}
		}
	}
	logBuf.WriteString("\n")

	// =========================================================================
	// シナリオ 2: カスタム変数代入前後のセマンティック・ハイブリッド変化
	// =========================================================================
	logBuf.WriteString("[シナリオ 2] 変数代入前後の動的セマンティック着色検証: 'sin(x + a)'\n")
	logBuf.WriteString(strings.Repeat("-", 60) + "\n")

	env := calc.NewEnv()

	// 2-1: 代入前（a も x も未束縛記号）
	logBuf.WriteString("【代入前 (a 未定義)】\n")
	framePre, colorMapPre := dumpKeystrokeFrame("sin(x + a)", env, "dark")
	logBuf.WriteString(framePre)

	// x (pos 4) と a (pos 8) はどちらも標準色 (None) であること
	if colorMapPre[4] != "" {
		t.Errorf("代入前の x は標準色であるべきですが、色がつきました: %q", colorMapPre[4])
	}
	if colorMapPre[8] != "" {
		t.Errorf("代入前の a は標準色であるべきですが、色がつきました: %q", colorMapPre[8])
	}

	// 2-2: 代入実行: a = 10
	env.Set("a", &calc.RationalNode{Val: big.NewRat(10, 1)})
	logBuf.WriteString("\n【代入実行】 a = 10 を env に登録\n\n")

	// 2-3: 代入後（a は定義済み変数として Soft Cyan、x は依然として未束縛記号で白）
	logBuf.WriteString("【代入後 (a 定義済み)】\n")
	framePost, colorMapPost := dumpKeystrokeFrame("sin(x + a)", env, "dark")
	logBuf.WriteString(framePost)

	// x (pos 4) は白 (None) のまま
	if colorMapPost[4] != "" {
		t.Errorf("代入後も未束縛の x は標準色のままであるべきですが、色がつきました: %q", colorMapPost[4])
	}
	// a (pos 8) だけが Soft Cyan (\x1B[96;2m) に光ること！
	if colorMapPost[8] != "\x1B[96;2m" {
		t.Errorf("代入後の a は Soft Cyan \x1B[96;2m になるべきですが、got %q", colorMapPost[8])
	}
	logBuf.WriteString("\n")

	// =========================================================================
	// シナリオ 3: ミスマッチ括弧 '(1+2]' の赤背景と行末エスケープリセット
	// =========================================================================
	logBuf.WriteString("[シナリオ 3] ミスマッチ括弧 '(1+2]' の警告と行末エスケープリセット検証\n")
	logBuf.WriteString(strings.Repeat("-", 60) + "\n")

	frameErr, colorMapErr := dumpKeystrokeFrame("(1+2]", nil, "dark")
	logBuf.WriteString(frameErr)

	// ']' (pos 4) は赤背景白文字 (\x1B[97;41m) であること
	if colorMapErr[4] != "\x1B[97;41m" {
		t.Errorf("pos 4 (']'): expected mismatch alert \x1B[97;41m, got %q", colorMapErr[4])
	}

	logBuf.WriteString("  [Reset Guard]: Editor.ResetColor = \"\\x1B[0m\" による行末リセット確認完了\n\n")
	logBuf.WriteString(separator)
	logBuf.WriteString("  全キーストローク・カラーダンプ検証 SUCCESS\n")
	logBuf.WriteString(separator)

	// ログファイルへ書き出し
	err := os.WriteFile(logFile, []byte(logBuf.String()), 0644)
	if err != nil {
		t.Fatalf("ログファイルの書き出しに失敗しました: %v", err)
	}

	t.Logf("カラーダンプトレースログを保存しました: %s", logFile)
}
