package calc

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ErrVMBudgetExceeded indicates that VM execution exceeded its instruction budget.
var ErrVMBudgetExceeded = errors.New("vm execution budget exceeded")

// Opcode represents a 1-byte virtual machine instruction code.
type Opcode byte

const (
	OpConstant Opcode = iota // [OpConstant, const_idx (uint16)]
	OpGetLocal               // [OpGetLocal, slot_idx (uint16)]
	OpSetLocal               // [OpSetLocal, slot_idx (uint16)]
	OpGetGlobal              // [OpGetGlobal, name_const_idx (uint16)]
	OpSetGlobal              // [OpSetGlobal, name_const_idx (uint16)]
	OpAdd                    // [OpAdd] (pop b, pop a, push a + b)
	OpSub                    // [OpSub] (pop b, pop a, push a - b)
	OpMul                    // [OpMul] (pop b, pop a, push a * b)
	OpDiv                    // [OpDiv] (pop b, pop a, push a / b)
	OpPow                    // [OpPow] (pop b, pop a, push a ^ b)
	OpNeg                    // [OpNeg] (pop a, push -a)
	OpFactorial              // [OpFactorial] (pop a, push a!)
	OpCall                   // [OpCall, fn_name_idx (uint16), argc (uint8)]
	OpReturn                 // [OpReturn]
	OpPop                    // [OpPop]
)

// SymbolScope denotes lexical scope level.
type SymbolScope string

const (
	GlobalScope SymbolScope = "GLOBAL"
	LocalScope  SymbolScope = "LOCAL"
)

// Symbol defines an identifier mapping to a stack slot or global storage.
type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

// SymbolTable tracks lexical variable scopes and assigns slot indices.
type SymbolTable struct {
	Outer          *SymbolTable
	store          map[string]Symbol
	numDefinitions int
}

// NewSymbolTable creates a root global SymbolTable.
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store: make(map[string]Symbol),
	}
}

// NewEnclosedSymbolTable creates a nested child SymbolTable.
func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Outer: outer,
		store: make(map[string]Symbol),
	}
}

// Define binds an identifier in the current scope.
func (st *SymbolTable) Define(name string) Symbol {
	scope := LocalScope
	sym := Symbol{
		Name:  name,
		Scope: scope,
		Index: st.numDefinitions,
	}
	st.numDefinitions++
	st.store[name] = sym
	return sym
}

// Resolve looks up an identifier in the current or outer scopes.
func (st *SymbolTable) Resolve(name string) (Symbol, bool) {
	sym, ok := st.store[name]
	if !ok && st.Outer != nil {
		return st.Outer.Resolve(name)
	}
	return sym, ok
}

// Bytecode holds compiled instructions, constant pool, and local requirements.
type Bytecode struct {
	Instructions []byte
	Constants    []Node
	NumLocals    int
}

// Compiler compiles an AST expression into a Bytecode object.
type Compiler struct {
	instructions []byte
	constants    []Node
	symbolTable  *SymbolTable
}

// NewCompiler creates an AST compiler.
func NewCompiler(st *SymbolTable) *Compiler {
	if st == nil {
		st = NewSymbolTable()
	}
	return &Compiler{
		symbolTable: st,
	}
}

// Compile compiles node into bytecode.
func (c *Compiler) Compile(node Node) (*Bytecode, error) {
	if err := c.compileNode(node); err != nil {
		return nil, err
	}
	c.emit(OpReturn)
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
		NumLocals:    c.symbolTable.numDefinitions,
	}, nil
}

func (c *Compiler) addConstant(node Node) int {
	c.constants = append(c.constants, node)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op Opcode, operands ...int) {
	c.instructions = append(c.instructions, byte(op))
	for _, o := range operands {
		// Emit uint16 big-endian
		c.instructions = append(c.instructions, byte(o>>8), byte(o&0xFF))
	}
}

func (c *Compiler) compileNode(node Node) error {
	switch v := node.(type) {
	case *RationalNode, *ComplexNode, *ConstNode:
		idx := c.addConstant(v)
		c.emit(OpConstant, idx)
		return nil

	case *VarNode:
		sym, ok := c.symbolTable.Resolve(v.Name)
		if ok {
			if sym.Scope == LocalScope {
				c.emit(OpGetLocal, sym.Index)
				return nil
			}
		}
		// Fallback to Global
		idx := c.addConstant(v)
		c.emit(OpGetGlobal, idx)
		return nil

	case *UnaryOpNode:
		if err := c.compileNode(v.Expr); err != nil {
			return err
		}
		if v.Op == "-" {
			c.emit(OpNeg)
		} else if v.Op == "!" {
			c.emit(OpFactorial)
		} else {
			return fmt.Errorf("unsupported unary op for VM: %s", v.Op)
		}
		return nil

	case *AddNode:
		if len(v.Terms) == 0 {
			idx := c.addConstant(&RationalNode{Val: new(big.Rat)})
			c.emit(OpConstant, idx)
			return nil
		}
		if err := c.compileNode(v.Terms[0]); err != nil {
			return err
		}
		for i := 1; i < len(v.Terms); i++ {
			if err := c.compileNode(v.Terms[i]); err != nil {
				return err
			}
			c.emit(OpAdd)
		}
		return nil

	case *MulNode:
		if len(v.Factors) == 0 {
			idx := c.addConstant(&RationalNode{Val: big.NewRat(1, 1)})
			c.emit(OpConstant, idx)
			return nil
		}
		if err := c.compileNode(v.Factors[0]); err != nil {
			return err
		}
		for i := 1; i < len(v.Factors); i++ {
			if err := c.compileNode(v.Factors[i]); err != nil {
				return err
			}
			c.emit(OpMul)
		}
		return nil

	case *PowNode:
		if err := c.compileNode(v.Base); err != nil {
			return err
		}
		if err := c.compileNode(v.Exp); err != nil {
			return err
		}
		c.emit(OpPow)
		return nil

	case *FuncNode:
		if spec, ok := LookupFunction(v.Name); ok && spec.LazyArgs {
			return fmt.Errorf("function %s with lazy arguments cannot be compiled to bytecode", v.Name)
		}
		// Push arguments onto stack
		for _, arg := range v.Args {
			if err := c.compileNode(arg); err != nil {
				return err
			}
		}
		idx := c.addConstant(&VarNode{Name: v.Name})
		c.emit(OpCall, idx)
		c.instructions = append(c.instructions, byte(len(v.Args)))
		return nil

	default:
		return fmt.Errorf("ast node type %T is not supported for bytecode compilation", node)
	}
}

// CallFrame holds execution context for a bytecode sequence.
type CallFrame struct {
	bytecode    *Bytecode
	ip          int
	basePointer int
}

// VM executes Bytecode instructions on a fast value stack with budget limits.
type VM struct {
	bytecode *Bytecode
	stack    []Node
	sp       int
	frames   []CallFrame
	env      *Env
	budget   int64
}

// DefaultVMBudget is the maximum instruction step limit for a single VM execution pass.
const DefaultVMBudget int64 = 10000000

// NewVM creates a virtual machine instance.
func NewVM(bc *Bytecode, env *Env, budget int64) *VM {
	if budget <= 0 {
		budget = DefaultVMBudget
	}
	stackSize := 2048
	if bc != nil && bc.NumLocals+32 > stackSize {
		stackSize = bc.NumLocals*2 + 1024
	}
	vm := &VM{
		bytecode: bc,
		stack:    make([]Node, stackSize),
		sp:       0,
		frames:   make([]CallFrame, 0, 64),
		env:      env,
		budget:   budget,
	}
	if bc != nil {
		vm.frames = append(vm.frames, CallFrame{
			bytecode:    bc,
			ip:          0,
			basePointer: 0,
		})
	}
	return vm
}

// Push pushes a value onto the evaluation stack.
func (vm *VM) Push(val Node) {
	if vm.sp >= len(vm.stack) {
		newStack := make([]Node, len(vm.stack)*2)
		copy(newStack, vm.stack)
		vm.stack = newStack
	}
	vm.stack[vm.sp] = val
	vm.sp++
}

// Reset re-initializes VM execution pointers and frames for re-running the same bytecode.
func (vm *VM) Reset() {
	vm.sp = 0
	vm.frames = vm.frames[:0]
	if vm.bytecode != nil {
		vm.frames = append(vm.frames, CallFrame{
			bytecode:    vm.bytecode,
			ip:          0,
			basePointer: 0,
		})
	}
}

// PushArgs pushes function arguments into basePointer slots and advances sp past locals.
func (vm *VM) PushArgs(args []Node) {
	for i, arg := range args {
		if i < len(vm.stack) {
			vm.stack[i] = arg
		}
	}
	if len(args) > vm.sp {
		vm.sp = len(args)
	}
	if vm.bytecode != nil && vm.bytecode.NumLocals > vm.sp {
		vm.sp = vm.bytecode.NumLocals
	}
}

// Pop pops a value from the evaluation stack.
func (vm *VM) Pop() Node {
	if vm.sp == 0 {
		return nil
	}
	vm.sp--
	val := vm.stack[vm.sp]
	vm.stack[vm.sp] = nil // avoid memory leak
	return val
}

// Run executes the bytecode instructions until return or budget exhaustion.
func (vm *VM) Run() (Node, error) {
	for len(vm.frames) > 0 {
		frame := &vm.frames[len(vm.frames)-1]
		bc := frame.bytecode

		for frame.ip < len(bc.Instructions) {
			if vm.budget <= 0 {
				return nil, fmt.Errorf("%w: %s", ErrVMBudgetExceeded, i18n.T("errors.budget_exceeded"))
			}
			vm.budget--

			op := Opcode(bc.Instructions[frame.ip])
			frame.ip++

			switch op {
			case OpConstant:
				idx := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				vm.Push(bc.Constants[idx])

			case OpGetLocal:
				slot := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				val := vm.stack[frame.basePointer+slot]
				vm.Push(val)

			case OpSetLocal:
				slot := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				vm.stack[frame.basePointer+slot] = vm.Pop()

			case OpGetGlobal:
				idx := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				name := bc.Constants[idx].(*VarNode).Name
				if val, ok := vm.env.Get(name); ok {
					vm.Push(val)
				} else {
					vm.Push(&VarNode{Name: name})
				}

			case OpSetGlobal:
				idx := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				name := bc.Constants[idx].(*VarNode).Name
				vm.env.Set(name, vm.Pop())

			case OpAdd:
				b := vm.Pop()
				a := vm.Pop()
				res, err := EvalWithEnv(&AddNode{Terms: []Node{a, b}}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpSub:
				b := vm.Pop()
				a := vm.Pop()
				res, err := EvalWithEnv(&AddNode{Terms: []Node{a, &UnaryOpNode{Op: "-", Expr: b}}}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpMul:
				b := vm.Pop()
				a := vm.Pop()
				res, err := EvalWithEnv(&MulNode{Factors: []Node{a, b}}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpDiv:
				b := vm.Pop()
				a := vm.Pop()
				res, err := EvalWithEnv(&MulNode{Factors: []Node{a, &PowNode{Base: b, Exp: &RationalNode{Val: big.NewRat(-1, 1)}}}}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpPow:
				exp := vm.Pop()
				base := vm.Pop()
				res, err := EvalWithEnv(&PowNode{Base: base, Exp: exp}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpNeg:
				a := vm.Pop()
				res, err := EvalWithEnv(&UnaryOpNode{Op: "-", Expr: a}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpFactorial:
				a := vm.Pop()
				res, err := EvalWithEnv(&UnaryOpNode{Op: "!", Expr: a}, vm.env)
				if err != nil {
					return nil, err
				}
				vm.Push(res)

			case OpCall:
				nameIdx := int(bc.Instructions[frame.ip])<<8 | int(bc.Instructions[frame.ip+1])
				frame.ip += 2
				argc := int(bc.Instructions[frame.ip])
				frame.ip++

				fnName := bc.Constants[nameIdx].(*VarNode).Name
				args := make([]Node, argc)
				for i := argc - 1; i >= 0; i-- {
					args[i] = vm.Pop()
				}

				// Check user function first
				if userFn, ok := vm.env.GetUserFunc(fnName); ok {
					res, err := evalUserFunctionDirect(userFn, args, vm.env)
					if err != nil {
						return nil, err
					}
					vm.Push(res)
				} else {
					// Fallback to general FuncNode evaluation
					res, err := EvalWithEnv(&FuncNode{Name: fnName, Args: args}, vm.env)
					if err != nil {
						return nil, err
					}
					vm.Push(res)
				}

			case OpReturn:
				retVal := vm.Pop()
				// Pop current frame
				vm.frames = vm.frames[:len(vm.frames)-1]
				if len(vm.frames) == 0 {
					return retVal, nil
				}
				vm.Push(retVal)

			case OpPop:
				vm.Pop()

			default:
				return nil, fmt.Errorf("unknown opcode: 0x%02x", op)
			}
		}
	}

	if vm.sp > 0 {
		return vm.Pop(), nil
	}
	return nil, nil
}
