package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Coding Theory: Hamming Distance, Reed-Solomon Encode & Berlekamp-Welch Decode
// -------------------------------------------------------------------------

// HammingDistance computes the number of positions where elements differ between u and v.
func HammingDistance(u, v []*big.Int) (int, error) {
	if len(u) != len(v) {
		return 0, fmt.Errorf("%s", i18n.T("coding.err_vector_length_mismatch", len(u), len(v)))
	}
	dist := 0
	for i := range u {
		if u[i].Cmp(v[i]) != 0 {
			dist++
		}
	}
	return dist, nil
}

// RSEncode encodes message polynomial coefficients msg into a codeword evaluated at points over F_p.
// P(x) = msg[0] + msg[1]*x + ... + msg[k-1]*x^(k-1)
func RSEncode(msg, points []*big.Int, prime *big.Int) ([]*big.Int, error) {
	if prime == nil || prime.Sign() <= 0 || !prime.ProbablyPrime(20) {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_modulus_must_be_prime"))
	}
	if len(msg) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_empty_message"))
	}
	if len(points) < len(msg) {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_points_less_than_message_length"))
	}

	// Verify points are distinct mod p
	pointSet := make(map[string]bool)
	for _, pt := range points {
		val := new(big.Int).Mod(pt, prime)
		k := val.String()
		if pointSet[k] {
			return nil, fmt.Errorf("%s", i18n.T("coding.err_duplicate_evaluation_point", k))
		}
		pointSet[k] = true
	}

	codeword := make([]*big.Int, len(points))
	for i, pt := range points {
		codeword[i] = evalPolyMod(msg, pt, prime)
	}
	return codeword, nil
}

// evalPolyMod evaluates P(pt) mod p using Horner's method.
func evalPolyMod(coeffs []*big.Int, pt, prime *big.Int) *big.Int {
	res := big.NewInt(0)
	for i := len(coeffs) - 1; i >= 0; i-- {
		res.Mul(res, pt)
		res.Add(res, coeffs[i])
		res.Mod(res, prime)
	}
	if res.Sign() < 0 {
		res.Add(res, prime)
	}
	return res
}

// RSDecode decodes a corrupted Reed-Solomon codeword using the Berlekamp-Welch algorithm over F_p.
// Solves Q(a_i) - y_i * E(a_i) = 0 mod p, computes P(x) = Q(x)/E(x), and verifies error distance <= e_max.
func RSDecode(received, points []*big.Int, k int, prime *big.Int) ([]*big.Int, error) {
	if prime == nil || prime.Sign() <= 0 || !prime.ProbablyPrime(20) {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_modulus_must_be_prime"))
	}
	n := len(received)
	if n != len(points) {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_points_codeword_length_mismatch", len(points), n))
	}
	if k <= 0 || k > n {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_invalid_message_length_k", k, n))
	}

	eMax := (n - k) / 2
	if eMax < 0 {
		eMax = 0
	}

	// Try decreasing assumed error count e from eMax down to 0
	for e := eMax; e >= 0; e-- {
		msg, err := tryBerlekampWelch(received, points, k, e, prime)
		if err == nil {
			// Re-encode and verify actual Hamming distance <= eMax
			reEncoded, encErr := RSEncode(msg, points, prime)
			if encErr == nil {
				dist, distErr := HammingDistance(received, reEncoded)
				if distErr == nil && dist <= eMax {
					return msg, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("coding.err_decoding_capacity_exceeded", eMax))
}

func tryBerlekampWelch(y, alpha []*big.Int, k, e int, prime *big.Int) ([]*big.Int, error) {
	n := len(y)
	numQ := k + e
	numE := e
	numVars := numQ + numE

	if n < numVars {
		return nil, fmt.Errorf("system underdetermined")
	}

	// Matrix size: n rows, numVars columns, plus RHS
	matrix := make([][]*big.Int, n)
	rhs := make([]*big.Int, n)

	for i := 0; i < n; i++ {
		row := make([]*big.Int, numVars)
		a_i := new(big.Int).Mod(alpha[i], prime)
		y_i := new(big.Int).Mod(y[i], prime)

		// Q terms: q_j * a_i^j
		powA := big.NewInt(1)
		for j := 0; j < numQ; j++ {
			row[j] = new(big.Int).Set(powA)
			powA.Mul(powA, a_i)
			powA.Mod(powA, prime)
		}

		// E terms: - y_i * e_j * a_i^j
		powA = big.NewInt(1)
		for j := 0; j < numE; j++ {
			coeff := new(big.Int).Mul(y_i, powA)
			coeff.Neg(coeff)
			coeff.Mod(coeff, prime)
			if coeff.Sign() < 0 {
				coeff.Add(coeff, prime)
			}
			row[numQ+j] = coeff

			powA.Mul(powA, a_i)
			powA.Mod(powA, prime)
		}

		// RHS: y_i * a_i^e
		rVal := new(big.Int).Mul(y_i, powA)
		rVal.Mod(rVal, prime)
		if rVal.Sign() < 0 {
			rVal.Add(rVal, prime)
		}
		rhs[i] = rVal

		matrix[i] = row
	}

	// Solve linear system over F_p using Gaussian elimination
	sol, err := solveLinearSystemFp(matrix, rhs, numVars, prime)
	if err != nil {
		return nil, err
	}

	// Extract Q(x) and E(x)
	qCoeffs := make([]*big.Int, numQ)
	for j := 0; j < numQ; j++ {
		qCoeffs[j] = sol[j]
	}
	eCoeffs := make([]*big.Int, numE+1)
	for j := 0; j < numE; j++ {
		eCoeffs[j] = sol[numQ+j]
	}
	eCoeffs[numE] = big.NewInt(1) // Monic leading coefficient

	// Polynomial division Q(x) / E(x) mod p
	quot, rem, err := polyDivModFp(qCoeffs, eCoeffs, prime)
	if err != nil {
		return nil, err
	}

	// Remainder must be identically 0
	for _, r := range rem {
		if r.Sign() != 0 {
			return nil, fmt.Errorf("non-zero remainder in Q/E")
		}
	}

	// Quotient degree must be < k
	msg := make([]*big.Int, k)
	for j := 0; j < k; j++ {
		if j < len(quot) {
			msg[j] = quot[j]
		} else {
			msg[j] = big.NewInt(0)
		}
	}
	for j := k; j < len(quot); j++ {
		if quot[j].Sign() != 0 {
			return nil, fmt.Errorf("quotient degree exceeds k-1")
		}
	}

	return msg, nil
}

// solveLinearSystemFp solves A * x = b over F_p using Gaussian elimination with back-substitution.
func solveLinearSystemFp(A [][]*big.Int, b []*big.Int, numVars int, prime *big.Int) ([]*big.Int, error) {
	n := len(A)
	M := make([][]*big.Int, n)
	for i := range A {
		M[i] = make([]*big.Int, numVars+1)
		for j := 0; j < numVars; j++ {
			M[i][j] = new(big.Int).Set(A[i][j])
		}
		M[i][numVars] = new(big.Int).Set(b[i])
	}

	row := 0
	for col := 0; col < numVars && row < n; col++ {
		// Find pivot
		pivotRow := -1
		for r := row; r < n; r++ {
			if M[r][col].Sign() != 0 {
				pivotRow = r
				break
			}
		}
		if pivotRow == -1 {
			continue // Free variable
		}

		M[row], M[pivotRow] = M[pivotRow], M[row]

		// Normalize pivot row
		inv := new(big.Int).ModInverse(M[row][col], prime)
		if inv == nil {
			return nil, fmt.Errorf("modular inverse does not exist")
		}
		for c := col; c <= numVars; c++ {
			M[row][c].Mul(M[row][c], inv)
			M[row][c].Mod(M[row][c], prime)
		}

		// Eliminate other rows
		for r := 0; r < n; r++ {
			if r != row && M[r][col].Sign() != 0 {
				factor := new(big.Int).Set(M[r][col])
				for c := col; c <= numVars; c++ {
					sub := new(big.Int).Mul(factor, M[row][c])
					M[r][c].Sub(M[r][c], sub)
					M[r][c].Mod(M[r][c], prime)
					if M[r][c].Sign() < 0 {
						M[r][c].Add(M[r][c], prime)
					}
				}
			}
		}
		row++
	}

	// Check consistency for rows row..n-1
	for r := row; r < n; r++ {
		if M[r][numVars].Sign() != 0 {
			return nil, fmt.Errorf("inconsistent linear system")
		}
	}

	// Extract solution (free variables set to 0)
	sol := make([]*big.Int, numVars)
	for j := range sol {
		sol[j] = big.NewInt(0)
	}

	for r := 0; r < row; r++ {
		leadCol := -1
		for c := 0; c < numVars; c++ {
			if M[r][c].Sign() != 0 {
				leadCol = c
				break
			}
		}
		if leadCol != -1 {
			sol[leadCol] = M[r][numVars]
		}
	}

	return sol, nil
}

// polyDivModFp computes quotient and remainder of polyA / polyB over F_p.
func polyDivModFp(a, b []*big.Int, prime *big.Int) (quot, rem []*big.Int, err error) {
	degA := len(a) - 1
	for degA >= 0 && a[degA].Sign() == 0 {
		degA--
	}
	degB := len(b) - 1
	for degB >= 0 && b[degB].Sign() == 0 {
		degB--
	}
	if degB < 0 {
		return nil, nil, fmt.Errorf("division by zero polynomial")
	}

	if degA < degB {
		remCopy := make([]*big.Int, degA+1)
		for i := 0; i <= degA; i++ {
			remCopy[i] = new(big.Int).Mod(a[i], prime)
		}
		return []*big.Int{big.NewInt(0)}, remCopy, nil
	}

	r := make([]*big.Int, degA+1)
	for i := 0; i <= degA; i++ {
		r[i] = new(big.Int).Mod(a[i], prime)
	}

	degQ := degA - degB
	q := make([]*big.Int, degQ+1)
	for i := range q {
		q[i] = big.NewInt(0)
	}

	leadInv := new(big.Int).ModInverse(b[degB], prime)
	if leadInv == nil {
		return nil, nil, fmt.Errorf("leading coeff not invertible mod prime")
	}

	term := new(big.Int)

	for i := degA; i >= degB; i-- {
		if r[i].Sign() == 0 {
			continue
		}
		scale := new(big.Int).Mul(r[i], leadInv)
		scale.Mod(scale, prime)
		q[i-degB] = scale

		for j := 0; j <= degB; j++ {
			term.Mul(scale, b[j])
			r[i-degB+j].Sub(r[i-degB+j], term)
			r[i-degB+j].Mod(r[i-degB+j], prime)
			if r[i-degB+j].Sign() < 0 {
				r[i-degB+j].Add(r[i-degB+j], prime)
			}
		}
	}

	// Trim remainder
	remDeg := degB - 1
	for remDeg >= 0 && r[remDeg].Sign() == 0 {
		remDeg--
	}
	var trimmedRem []*big.Int
	if remDeg < 0 {
		trimmedRem = []*big.Int{big.NewInt(0)}
	} else {
		trimmedRem = r[:remDeg+1]
	}

	return q, trimmedRem, nil
}

// -------------------------------------------------------------------------
// CLI / REPL Handlers
// -------------------------------------------------------------------------

func init() {
	RegisterFunction(FunctionSpec{
		Name:            "hamming_dist",
		MinArgs:         2,
		MaxArgs:         2,
		AllowBareSymbol: true,
		Handler:         handleHammingDistance,
	})
	RegisterFunction(FunctionSpec{
		Name:            "rs_encode",
		MinArgs:         3,
		MaxArgs:         3,
		AllowBareSymbol: true,
		Handler:         handleRSEncode,
	})
	RegisterFunction(FunctionSpec{
		Name:            "rs_decode",
		MinArgs:         4,
		MaxArgs:         4,
		AllowBareSymbol: true,
		Handler:         handleRSDecode,
	})
}

func extractBigIntList(n ast.Node) ([]*big.Int, error) {
	listNode, ok := n.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_expected_list_argument", n.String()))
	}
	res := make([]*big.Int, len(listNode.Elements))
	for i, elem := range listNode.Elements {
		r, ok := elem.(*ast.RationalNode)
		if !ok || !r.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("coding.err_list_elements_must_be_integers"))
		}
		res[i] = new(big.Int).Set(r.Val.Num())
	}
	return res, nil
}

func handleHammingDistance(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_hamming_dist_arg_count"))
	}
	u, err := extractBigIntList(args[0])
	if err != nil {
		return nil, err
	}
	v, err := extractBigIntList(args[1])
	if err != nil {
		return nil, err
	}
	dist, err := HammingDistance(u, v)
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(big.NewRat(int64(dist), 1)), nil
}

func handleRSEncode(args []Node, env *Env) (Node, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_rs_encode_arg_count"))
	}
	msg, err := extractBigIntList(args[0])
	if err != nil {
		return nil, err
	}
	points, err := extractBigIntList(args[1])
	if err != nil {
		return nil, err
	}
	primeNode, ok := args[2].(*ast.RationalNode)
	if !ok || !primeNode.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_modulus_must_be_prime"))
	}
	cw, err := RSEncode(msg, points, primeNode.Val.Num())
	if err != nil {
		return nil, err
	}

	elems := make([]ast.Node, len(cw))
	for i, val := range cw {
		elems[i] = ast.NewRationalFromBigRat(new(big.Rat).SetInt(val))
	}
	return ast.NewList(elems), nil
}

func handleRSDecode(args []Node, env *Env) (Node, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_rs_decode_arg_count"))
	}
	rec, err := extractBigIntList(args[0])
	if err != nil {
		return nil, err
	}
	points, err := extractBigIntList(args[1])
	if err != nil {
		return nil, err
	}
	kNode, ok := args[2].(*ast.RationalNode)
	if !ok || !kNode.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_k_must_be_integer"))
	}
	k := int(kNode.Val.Num().Int64())

	primeNode, ok := args[3].(*ast.RationalNode)
	if !ok || !primeNode.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("coding.err_modulus_must_be_prime"))
	}

	msg, err := RSDecode(rec, points, k, primeNode.Val.Num())
	if err != nil {
		return nil, err
	}

	elems := make([]ast.Node, len(msg))
	for i, val := range msg {
		elems[i] = ast.NewRationalFromBigRat(new(big.Rat).SetInt(val))
	}
	return ast.NewList(elems), nil
}
