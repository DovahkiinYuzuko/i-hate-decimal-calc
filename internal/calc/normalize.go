package calc

import "strings"

// NormalizeInput converts full-width (Zenkaku) characters into half-width (Hankaku) ASCII characters.
// This allows seamless calculation when Japanese IME is accidentally active.
func NormalizeInput(input string) string {
	var sb strings.Builder
	sb.Grow(len(input))

	for _, r := range input {
		switch {
		// Full-width digits: '０' (0xFF10) - '９' (0xFF19) -> '0' - '9'
		case r >= '０' && r <= '９':
			sb.WriteRune(r - '０' + '0')

		// Full-width uppercase letters: 'Ａ' (0xFF21) - 'Ｚ' (0xFF3A) -> 'A' - 'Z'
		case r >= 'Ａ' && r <= 'Ｚ':
			sb.WriteRune(r - 'Ａ' + 'A')

		// Full-width lowercase letters: 'ａ' (0xFF41) - 'ｚ' (0xFF5A) -> 'a' - 'z'
		case r >= 'ａ' && r <= 'ｚ':
			sb.WriteRune(r - 'ａ' + 'a')

		// Full-width space
		case r == '　':
			sb.WriteRune(' ')

		// Full-width plus: '＋'
		case r == '＋':
			sb.WriteRune('+')

		// Minus / Hyphen / Dash variants
		case r == '－' || r == '−' || r == 'ー' || r == '―' || r == '‐' || r == '—':
			sb.WriteRune('-')

		// Multiplication: '＊', '×'
		case r == '＊' || r == '×':
			sb.WriteRune('*')

		// Division: '／', '÷'
		case r == '／' || r == '÷':
			sb.WriteRune('/')

		// Power: '＾'
		case r == '＾':
			sb.WriteRune('^')

		// Factorial: '！'
		case r == '！':
			sb.WriteRune('!')

		// Parentheses: '（', '）'
		case r == '（':
			sb.WriteRune('(')
		case r == '）':
			sb.WriteRune(')')

		// Equal: '＝'
		case r == '＝':
			sb.WriteRune('=')

		// Comma and period
		case r == '，' || r == '、':
			sb.WriteRune(',')
		case r == '．' || r == '。':
			sb.WriteRune('.')

		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
