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
	/* R-Type */
	Inst_Add: "add",
	Inst_Sub: "sub",
	Inst_Mul: "mul",
	Inst_Div: "div",
	Inst_Rem: "rem",
	Inst_Xor: "xor",
	Inst_Or:  "or",
	Inst_And: "and",

	/* I-Type */
	Inst_Addi: "addi",
	Inst_Subi: "subi",
	Inst_Xori: "xori",
	Inst_Ori:  "ori",
	Inst_Andi: "andi",
	Inst_Jalr: "jalr",
	Inst_Lw:   "lw",
	Inst_Lh:   "lh",
	Inst_Lb:   "lb",
	Inst_Slli: "slli",
	Inst_Srli: "srli",
	Inst_Srai: "srai",

	/* S-Type */
	Inst_Sw: "sw",
	Inst_Sh: "sh",
	Inst_Sb: "sb",

	/* B-Type */
	Inst_Beq: "beq",
	Inst_Bne: "bne",
	Inst_Blt: "blt",
	Inst_Bge: "bge",

	/* J-Type */
	Inst_Jal: "jal",

	/* U-Type */
	Inst_Lui:   "lui",
	Inst_Auipc: "auipc",

	/* Pseudo Instructions */
	Inst_Mv:   "mv",
	Inst_Not:  "not",
	Inst_Neg:  "neg",
	Inst_Li:   "li",
	Inst_Jr:   "jr",
	Inst_Ret:  "ret",
	Inst_Ble:  "ble",
	Inst_Bgt:  "bgt",
	Inst_J:    "j",
	Inst_Call: "call",
	Inst_End:  "end",
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
	Tok_Comma
	Tok_OpenParen
	Tok_CloseParen

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
func isSpace(ch rune) bool {
	return unicode.IsSpace(ch) || ch == ';' // || ch == ',' || ch == '(' || ch == ')'
}

func isSeperator(ch rune) bool {
	return unicode.IsSpace(ch) || ch == ',' || ch == '(' || ch == ')'
}

// If cursor goes to a newline returns true, otherwise false
func (l *Lexer) trimSpace() bool {
	newLine := false
	for int(l.Cursor) < len(l.Content) && isSpace(rune(l.Content[l.Cursor])) {
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

func (l *Lexer) expectAndConsumeToken(tok_type Token_Type) (bool, Token) {
	tok := l.nextToken()
	if tok.Type != tok_type {
		return false, tok
	}

	return true, tok
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
		for int(l.Cursor) < len(l.Content) && !isSeperator(rune(l.Content[l.Cursor])) {

			// If any character after the first digit is not a digit, this is not a valid number
			// We still want to get the whole token until a seperator character
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

	if l.Content[l.Cursor] == '(' {
		tok.Type = Tok_OpenParen
		tok.Value = "("
		l.Cursor++

		// We don't increment the tok_num for the seperators
		// l.tok_num++
		return tok
	}

	if l.Content[l.Cursor] == ')' {
		tok.Type = Tok_CloseParen
		tok.Value = ")"
		l.Cursor++

		return tok
	}

	if l.Content[l.Cursor] == ',' {
		tok.Type = Tok_Comma
		tok.Value = ","
		l.Cursor++

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
	lexer      *Lexer
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

	// These, holds the **index** of instruction in the program array
	// Multiply by 4 to convert to instruction address.
	parser.symbol_table = make(map[string]uint32)
	parser.insts_missing_label = make(map[uint32]string)

	// Push End to the beginning for ret's at the end of the program.
	parser.pushInstruction(newInstruction(Inst_End, 0, 0, 0))

	lexer := Lexer{}
	lexer.Content = program_str
	parser.lexer = &lexer

	// inst := Instruction{}
	tok := lexer.nextToken()
	for tok.Type != Tok_End {
		if tok.Type == Tok_Invalid {
			return nil, 0, fmt.Errorf("%v:%v Invalid token '%v'", tok.line_num+1, tok.start+1, tok.Value)
		}

		// First token of line MUST be a symbol
		if tok.num == 0 && tok.Type != Tok_Symbol {
			return nil, 0, fmt.Errorf("%v:%v Expected 'symbol', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
		}

		next := lexer.peekNextToken()
		if tok.Type == Tok_Symbol {
			if next.Type == Tok_Colon {
				parser.symbol_table[tok.Value] = parser.inst_count
			} else {

				ok, op := isOpcodeToken(tok)
				if !ok {
					return nil, 0, fmt.Errorf("%v:%v Expected 'opcode' got '%v'", tok.line_num+1, tok.start+1, tok.Value)
				}

				inst, err := parser.parseInstruction(op)
				if err != nil {
					return nil, 0, err
				}

				parser.pushInstruction(inst)
			}
		}

		tok = lexer.nextToken()
	}

	// for i, inst := range parser.Program {
	// 	fmt.Printf("%v: %v\n", i, inst.Str())
	// }

	// Fill the missing label calls
	for n, label := range parser.insts_missing_label {
		target, ok := parser.symbol_table[label]
		if !ok {
			return nil, 0, fmt.Errorf("Undeclared label '%v'", label)
		}

		offset := (target - n) * 4

		inst := &parser.Program[n]

		// based on different control instructions, the offset is stored in different place
		switch inst._fmt {
		case Fmt_B:
			inst.Rs2 = int32(offset)
		case Fmt_J:
			inst.Rs1 = int32(offset)
		default:
			// Inst_Jalr is an Fmt_I instruction but also a branch.
			if inst.Op == Inst_Jalr {
				inst.Rs2 = int32(offset)
				break
			}
			return nil, 0, fmt.Errorf("Illegal label use: '%s'", label)
		}
	}

	entry, ok := parser.symbol_table["main"]
	if !ok {
		entry = 1
	}

	return parser.Program, entry, nil
}

// Expandes if pseudo instruction then pushes to the program
// Returns the pushed instruction
func (p *Parser) pushInstruction(inst Instruction) Instruction {
	// inst = expandPseudoInstruction(inst)
	// inst._fmt = getInstructionFmt(inst)
	p.Program = append(p.Program, inst)
	p.inst_count++
	return inst
}

func (p *Parser) parseInstruction(op Inst_Op) (Instruction, error) {
	lexer := *p.lexer
	_ = lexer

	inst := Instruction{}
	inst.Op = op

	// we can fill out the inst._fmt
	inst._fmt = getInstructionFmt(inst)

	var err error
	switch inst._fmt {
	// opcode reg, reg, reg
	case Fmt_R:
		err = p.parseRType(&inst)
	case Fmt_U:
		err = p.parseUType(&inst)
	// opcode reg, reg, imm
	case Fmt_I:
		// TODO: this is a hack. Make it proper!!
		if inst.isLoad(){
			err = p.parseSType(&inst)
		}else{
			err = p.parseIType(&inst)
		}
	// opcode reg, reg, label
	case Fmt_B:
		err = p.parseBType(&inst)
	// opcode reg, label
	case Fmt_J:
		err = p.parseJType(&inst)
	// opcode reg, imm(reg)
	case Fmt_S:
		err = p.parseSType(&inst)

	// opcode ...
	case _Fmt_Pseudo:
		switch inst.Op {
		// opcode reg, reg
		case Inst_Mv, Inst_Not, Inst_Neg:
			reg, err := p.parseRegisterSymbol()
			if err != nil {
				return inst, err
			}
			ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
			if !ok {
				return inst, fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
			}
			inst.Rd = int32(reg)

			reg, err = p.parseRegisterSymbol()
			if err != nil {
				return inst, err
			}
			inst.Rs1 = int32(reg)

		// opcode reg, imm
		case Inst_Li:
			reg, err := p.parseRegisterSymbol()
			if err != nil {
				return inst, err
			}
			ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
			if !ok {
				return inst, fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
			}
			inst.Rd = int32(reg)

			imm, err := p.parseImmedieateValue()
			if err != nil {
				return inst, err
			}
			inst.Rs1 = imm

		// opcode label
		case Inst_J, Inst_Call:
			val, err := p.parseLabelSymbol()
			if err != nil {
				return inst, err
			}
			inst.Rd = val

		// opcode reg
		case Inst_Jr:
			reg, err := p.parseRegisterSymbol()
			if err != nil {
				return inst, err
			}
			inst.Rd = reg

		// opcode reg, reg, label
		case Inst_Ble, Inst_Bgt:
			p.parseBType(&inst)

		// Inst_Ret
		case Inst_Ret:
			break
		}

		inst = expandPseudoInstruction(inst)
		inst._fmt = getInstructionFmt(inst)
	default:
		panic(fmt.Sprintf("unexpected vm.Inst_Fmt: %#v", inst._fmt))
	}

	return inst, err
}

func (p *Parser) parseRType(inst *Instruction) error {
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	reg, err = p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok = p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rs1 = int32(reg)

	reg, err = p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	inst.Rs2 = int32(reg)

	return nil
}

func (p *Parser) parseIType(inst *Instruction) error {
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	reg, err = p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok = p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rs1 = int32(reg)

	imm, err := p.parseImmedieateValue()
	if err != nil {
		return err
	}
	inst.Rs2 = imm

	return nil
}

func (p *Parser) parseBType(inst *Instruction) error {
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	reg, err = p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok = p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rs1 = int32(reg)

	val, err := p.parseLabelSymbol()
	if err != nil {
		return err
	}
	inst.Rs2 = val

	return nil
}

func (p *Parser) parseJType(inst *Instruction) error {
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	val, err := p.parseLabelSymbol()
	if err != nil {
		return err
	}
	inst.Rs1 = val

	return nil
}

func (p *Parser) parseSType(inst *Instruction) error {
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	imm, err := p.parseImmedieateValue()
	if err != nil {
		return err
	}
	ok, tok = p.lexer.expectAndConsumeToken(Tok_OpenParen)
	if !ok {
		return fmt.Errorf("%v:%v Expected '(', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rs1 = imm

	reg, err = p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok = p.lexer.expectAndConsumeToken(Tok_CloseParen)
	if !ok {
		return fmt.Errorf("%v:%v Expected ')', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rs2 = int32(reg)

	return nil

}

func (p *Parser) parseUType(inst *Instruction) error {
	// opcode, reg, imm
	reg, err := p.parseRegisterSymbol()
	if err != nil {
		return err
	}
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Comma)
	if !ok {
		return fmt.Errorf("%v:%v Expected ',', got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}
	inst.Rd = int32(reg)

	imm, err := p.parseImmedieateValue()
	if err != nil {
		return err
	}
	inst.Rs1 = imm
	return nil
}

func (p *Parser) parseLabelSymbol() (int32, error) {
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Symbol)
	if !ok {
		return 0, fmt.Errorf("%v:%v Expected a label, got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}

	l, ok := p.symbol_table[tok.Value]
	if ok {
		val := int32(l-p.inst_count) * 4
		return val, nil
	} else {
		// Add a record to the inst missing label
		p.insts_missing_label[p.inst_count] = tok.Value
		return 0, nil
	}
}

func (p *Parser) parseRegisterSymbol() (int32, error) {
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Symbol)
	if !ok {
		return 0, fmt.Errorf("%v:%v Expected a register, got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}

	reg, ok := abiToRegNum[tok.Value]
	if !ok {
		return 0, fmt.Errorf("%v:%v Expected a register, got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}

	return int32(reg), nil
}

func (p *Parser) parseImmedieateValue() (int32, error) {
	ok, tok := p.lexer.expectAndConsumeToken(Tok_Number)
	if !ok {
		return 0, fmt.Errorf("%v:%v Expected an immediate value, got '%v'", tok.line_num+1, tok.start+1, tok.Value)
	}

	num, _ := strconv.Atoi(tok.Value)

	return int32(num), nil
}

func isOpcodeToken(tok Token) (bool, Inst_Op) {
	if tok.Type != Tok_Symbol {
		return false, _Inst_Unknown
	}

	op := stringToOpcode(tok.Value)
	return op != _Inst_Unknown, op
}
