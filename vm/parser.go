package vm

import (
	"fmt"
	"os"
	"strconv"
	"unicode"
)

var abiToRegNum = map[string]int{
	"zero": 0, // zero register
	"ra":   1, // Return address
	"sp":   2, // Stack pointer
	"gp":   3, // Global pointer
	"tp":   4, // Thread pointer

	// Temporaries
	"t0": 5, "t1": 6, "t2": 7,

	// Saved/frame pointer
	"s0": 8, "fp": 8,

	// Saved register
	"s1": 9,

	// Fn args/return values
	"a0": 10, "a1": 11,

	// Fn args
	"a2": 12, "a3": 13,
	"a4": 14, "a5": 15,
	"a6": 16, "a7": 17,

	// Saved registers
	"s2": 18, "s3": 19,
	"s4": 20, "s5": 21,
	"s6": 22, "s7": 23,
	"s8": 24, "s9": 25,
	"s10": 26, "s11": 27,

	// Temporaries
	"t3": 28, "t4": 29,
	"t5": 30, "t6": 31,
}

var opcodeToStringMap = map[Inst_Op]string{
	Inst_Add:   "add",
	Inst_Sub:   "sub",
	Inst_Mul:   "mul",
	Inst_Div:   "div",
	Inst_Rem:   "rem",
	Inst_Xor:   "xor",
	Inst_Or:    "or",
	Inst_And:   "and",
	Inst_Addi:  "addi",
	Inst_Subi:  "subi",
	Inst_Xori:  "xori",
	Inst_Ori:   "ori",
	Inst_Andi:  "andi",
	Inst_Jalr:  "jalr",
	Inst_Lw:    "lw",
	Inst_Lh:    "lh",
	Inst_Lb:    "lb",
	Inst_Slli:  "slli",
	Inst_Sw:    "sw",
	Inst_Sh:    "sh",
	Inst_Sb:    "sb",
	Inst_Beq:   "beq",
	Inst_Bne:   "bne",
	Inst_Blt:   "blt",
	Inst_Bge:   "bge",
	Inst_Jal:   "jal",
	Inst_Lui:   "lui",
	Inst_Auipc: "auipc",
	Inst_Mv:    "mv",
	Inst_Not:   "not",
	Inst_Neg:   "neg",
	Inst_Li:    "li",
	Inst_Jr:    "jr",
	Inst_Ret:   "ret",
	Inst_Ble:   "ble",
	Inst_Bgt:   "bgt",
	Inst_J:     "j",
	Inst_Call:  "call",
	Inst_End:   "end",
}

var stringToOpcodeMap map[string]Inst_Op = nil

// Returns the corresponding 'Inst_Op' for the given string, uses the stringToOpcode lookup table.
func stringToOpcode(s string) Inst_Op {
	// if stringToOpcodeMap is not created, create it
	if stringToOpcodeMap == nil {
		stringToOpcodeMap = make(map[string]Inst_Op)
		for op, str := range opcodeToStringMap {
			stringToOpcodeMap[str] = op
		}
	}

	val, ok := stringToOpcodeMap[s]
	if !ok {
		return _Inst_Unknown
	}

	return val
}

// ===================================
// ============== LEXER ==============
// ===================================

type Token_Type uint16

const (
	Tok_End Token_Type = iota
	Tok_Colon

	Tok_Number
	Tok_Symbol
	Tok_Invalid
)

type Token struct {
	Type  Token_Type
	Value string

	// Position of the token in the file
	line_num uint32
	start    uint32 // starting point within the line?

	num uint8 // Token number in a line
}

type Lexer struct {
	Content string // The file that we are tokenizing

	Cursor uint32
	Line   uint32 // Line number we are at
	Bol    uint32 // Beginning of line

	tok_num uint8 // Token count in a line
}

func isSymbolStart(b byte) bool {
	ch := rune(b)
	return unicode.IsLetter(ch) || ch == '.' || ch == '_'
}

func isSymbol(b byte) bool {
	ch := rune(b)
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// TODO: Check if paranthesis are valid
// We consider params and ',' as a space
func (l *Lexer) isSpace(ch rune) bool {
	return unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == ',' || ch == ';'
}

// If cursor goes to a newline returns true, otherwise false
func (l *Lexer) trimSpace() bool {
	newLine := false
	for int(l.Cursor) < len(l.Content) && l.isSpace(rune(l.Content[l.Cursor])) {
		if l.Content[l.Cursor] == ';' {
			for int(l.Cursor) < len(l.Content) && l.Content[l.Cursor] != '\n' {
				l.Cursor++
			}
		}

		// We need to increment the beginning of line and line counter too
		// If this is a newline
		ch := l.Content[l.Cursor]
		l.Cursor++
		if ch == '\n' {
			newLine = true
			l.Bol = l.Cursor
			l.Line++
		}
	}

	return newLine
}

func (l Lexer) peekNextToken() Token {
	tok := l.nextToken()
	return tok
}

func (l *Lexer) nextToken() Token {
	// Consume spaces
	newLine := l.trimSpace()
	if newLine {
		l.tok_num = 0
	}

	tok := Token{}
	tok.line_num = l.Line
	tok.start = l.Cursor - l.Bol
	tok.num = l.tok_num

	// Reached the end of content
	if int(l.Cursor) >= len(l.Content) {
		tok.Type = Tok_End
		l.tok_num++
		return tok
	}

	if isSymbolStart(l.Content[l.Cursor]) {
		tok.Type = Tok_Symbol
		l.Cursor++
		for int(l.Cursor) < len(l.Content) && isSymbol(l.Content[l.Cursor]) {
			l.Cursor++
		}

		tok.Value = l.Content[l.Bol+tok.start : l.Cursor]

		l.tok_num++
		return tok
	}

	if unicode.IsDigit(rune(l.Content[l.Cursor])) || l.Content[l.Cursor] == '-' {
		l.Cursor++

		tok.Type = Tok_Number
		for int(l.Cursor) < len(l.Content) && !l.isSpace(rune(l.Content[l.Cursor])) {

			// If any character after the first digit is not a digit, this is not a valid number
			// We still want to get the whole token until a space character
			// for reporting the whole word as an Tok_Invalid
			if !unicode.IsDigit(rune(l.Content[l.Cursor])) {
				tok.Type = Tok_Invalid
			}

			l.Cursor++
		}

		tok.Value = l.Content[l.Bol+tok.start : l.Cursor]
		if tok.Value == "-" {
			tok.Type = Tok_Invalid
		}

		l.tok_num++
		return tok
	}

	if l.Content[l.Cursor] == ':' {
		tok.Type = Tok_Colon
		tok.Value = ":"
		l.Cursor++

		l.tok_num++
		return tok
	}

	tok.Type = Tok_Invalid
	tok.Value = string(l.Content[l.Cursor])

	l.tok_num++
	return tok
}

// ====================================
// ============== PARSER ==============
// ====================================

type Parser struct {
	inst_count uint32

	// Symbol table holding label_str -> line_num
	symbol_table        map[string]uint32
	insts_missing_label map[uint32]string

	Program []Instruction
}

// Returns list of instructions parsed, the default pc and an error.
func ParseProgramFromFile(filename string) ([]Instruction, uint32, error) {
	str, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to read file for parsing '%v': %v", filename, err.Error())
	}

	return ParseProgramFromString(string(str))
}

func ParseProgramFromString(program_str string) ([]Instruction, uint32, error) {
	parser := Parser{}
	parser.symbol_table = make(map[string]uint32)
	parser.insts_missing_label = make(map[uint32]string)

	// Push End to the beginning for ret's at the end of the program.
	parser.pushInstruction(newInstruction(Inst_End, 0, 0, 0))

	lexer := Lexer{}
	lexer.Content = program_str

	inst := Instruction{}
	tok := lexer.nextToken()
	for tok.Type != Tok_End {
		if tok.Type == Tok_Invalid {
			return nil, 0, fmt.Errorf("%v:%v Invalid token '%v'", tok.line_num, tok.start+1, tok.Value)
		}

		// First token of line MUST be a symbol
		if tok.num == 0 && tok.Type != Tok_Symbol {
			return nil, 0, fmt.Errorf("%v:%v Expected 'symbol', got '%v'", tok.line_num, tok.start+1, tok.Value)
		}

		next := lexer.peekNextToken()
		switch tok.Type {
		case Tok_Symbol:
			if next.Type == Tok_Colon { // If the next token is ':', this is a label declaration.
				parser.symbol_table[tok.Value] = parser.inst_count
			} else {
				err := parser.fillInstructionToken(&inst, tok)
				if err != nil {
					return nil, 0, err
				}
			}

		case Tok_Number:
			err := parser.fillInstructionToken(&inst, tok)
			if err != nil {
				return nil, 0, err
			}
		}

		// Next token is in another line, push the instruction
		if (next.num == 0 || next.Type == Tok_End) && inst != (Instruction{}) {
			// Push the previous instruction
			parser.pushInstruction(inst)
			inst = Instruction{}
		}

		// fmt.Printf("%v %v:%v %v\n", tok.num, tok.line_num+1, tok.start+1, tok.Value)

		tok = lexer.nextToken()
	}

	// Fill the missing label calls
	for n, label := range parser.insts_missing_label {
		target, ok := parser.symbol_table[label]
		if !ok {
			return nil, 0, fmt.Errorf("Undeclared label '%v'", label)
		}

		offset := target - n

		inst := &parser.Program[n]
		// based on different control instructions, the offset is stored in different place
		switch inst._fmt {
		case Fmt_B:
			inst.Rs2 = int32(offset)
		case Fmt_J:
			inst.Rs1 = int32(offset)
		default:
			return nil, 0, fmt.Errorf("Illegal label use: '%s'", label)
		}
	}

	pc, ok := parser.symbol_table["main"]
	if !ok {
		pc = 1
	}

	return parser.Program, pc, nil
}

// Expandes if pseudo instruction then pushes to the program
// Returns the pushed instruction
func (p *Parser) pushInstruction(inst Instruction) Instruction {
	inst = expandPseudoInstruction(inst)
	inst._fmt = getInstructionFmt(inst)
	p.Program = append(p.Program, inst)
	p.inst_count++
	return inst
}

func (p *Parser) fillInstructionToken(inst *Instruction, tok Token) error {
	if tok.num == 0 {
		op := stringToOpcode(tok.Value)
		if op == _Inst_Unknown {
			return fmt.Errorf("%v:%v Unknown opcode '%v'\n", tok.line_num, tok.start, tok.Value)
		}

		inst.Op = op
		return nil
	}

	if inst == nil {
		return nil
	}

	var val int32
	switch tok.Type {
	case Tok_Symbol: // Register name or label call
		reg, ok := abiToRegNum[tok.Value]
		if ok {
			val = int32(reg)
		} else { // Then this is a label call
			l, ok := p.symbol_table[tok.Value]
			if ok {
				val = int32(l - p.inst_count)
			} else {
				// Add a record to the inst missing label
				p.insts_missing_label[p.inst_count] = tok.Value
			}
		}
	case Tok_Number:
		num, _ := strconv.Atoi(tok.Value)
		val = int32(num)
	}

	switch tok.num {
	case 1: // Rd
		inst.Rd = val
	case 2: // Rs1
		inst.Rs1 = val
	case 3: // Rs2
		inst.Rs2 = val
	default:
		return fmt.Errorf("%v:%v Unexpected token '%v'\n", tok.line_num, tok.start, tok.Value)
	}

	return nil
}
