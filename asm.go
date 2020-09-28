package simleg

import (
	"fmt"
	"io"
	"strings"
)

type Register uint8

// String implements Stringer for Register.
func (r Register) String() (s string) {
	switch {
	case r <= X30:
		return fmt.Sprintf("X%d", r)
	case r == XZR:
		return "XZR"
	case r >= S0 && r <= S31:
		return fmt.Sprintf("S%d", r-S0)
	case r >= D0 && r <= D31:
		return fmt.Sprintf("D%d", r-D0)
	default:
		return "invalid"
	}
}

// Registers
const (
	X0 Register = iota
	X1
	X2
	X3
	X4
	X5
	X6
	X7
	X8
	X9
	X10
	X11
	X12
	X13
	X14
	X15
	X16
	X17
	X18
	X19
	X20
	X21
	X22
	X23
	X24
	X25
	X26
	X27
	X28
	X29
	X30
	XZR // always zero

	// single-point floats
	S0
	S1
	S2
	S3
	S4
	S5
	S6
	S7
	S8
	S9
	S10
	S11
	S12
	S13
	S14
	S15
	S16
	S17
	S18
	S19
	S20
	S21
	S22
	S23
	S24
	S25
	S26
	S27
	S28
	S29
	S30
	S31

	// double-point floats
	D0
	D1
	D2
	D3
	D4
	D5
	D6
	D7
	D8
	D9
	D10
	D11
	D12
	D13
	D14
	D15
	D16
	D17
	D18
	D19
	D20
	D21
	D22
	D23
	D24
	D25
	D26
	D27
	D28
	D29
	D30
	D31
)

// Special purpose registers
const (
	IP0 = X16
	IP1 = X17

	SP = X28 // stack pointer
	FP = X29 // frame pointer
	LR = X30 // link register
)

type Addr struct {
	Reg    Register
	Offset uint64
	Label  string
}

type Instruction struct {
	Op    string   // mnemonic
	To    Addr     // destination
	From  Addr     // 1st operand
	Reg   Register // 2nd operand register
	Imm   uint64   // 2nd operand for iformat
	Label string
}

func (as Instruction) writeString(s *strings.Builder) {
	f, ok := opcodes[as.Op]
	if !ok {
		return
	}
	if as.Label != "" {
		s.WriteString(as.Label)
		s.WriteString(": ")
	}
	s.WriteString(as.Op)
	s.WriteByte(' ')
	f.s(s, as)
}

func (as Instruction) String() (s string) {
	sb := &strings.Builder{}
	as.writeString(sb)
	return sb.String()
}

func (as Instruction) registerPrefix() rune {
	switch {
	case strings.HasPrefix(as.Op, "F"):
		if strings.HasSuffix(as.Op, "S") {
			return 'S'
		}
		return 'D'
	case as.Op == "LDURS":
	case as.Op == "STURS":
		return 'S'
	case as.Op == "LDURD":
	case as.Op == "STURD":
		return 'D'
	}
	return 'X'
}

type Program []Instruction

func (p Program) String() string {
	indent := 0
	for _, as := range p {
		if len(as.Label) > indent {
			indent = len(as.Label)
		}
	}
	if indent > 0 {
		indent += 2
	}
	sb := &strings.Builder{}
	for _, as := range p {
		n := 0
		if as.Label != "" {
			n = (len(as.Label) + 2)
		}
		for i := 0; i < indent-n; i++ {
			sb.WriteByte(' ')
		}
		as.writeString(sb)
		sb.WriteByte('\n')
	}
	return sb.String()
}

type insFormat struct {
	p formatParser
	s func(w io.Writer, as Instruction)
}

var (
	rformat  = insFormat{rformatParser, rformatString}
	iformat  = insFormat{iformatParser, iformatString}
	dformat  = insFormat{dformatParser, dformatString}
	bformat  = insFormat{bformatParser, bformatString}
	cbformat = insFormat{cbformatParser, cbformatString}
	iwformat = insFormat{iwformatParser, iwformatString}
	imformat = insFormat{}
)

var opcodes = map[string]insFormat{
	"ADD":   rformat,
	"ADDI":  iformat,
	"ADDIS": iformat,
	"ADDS":  rformat,
	"AND":   rformat,
	"ANDI":  iformat,
	"ANDIS": iformat,
	"ANDS":  rformat,
	"B":     bformat,
	"B.EQ":  bformat,
	"B.NE":  bformat,
	"B.LT":  bformat,
	"B.LE":  bformat,
	"B.GT":  bformat,
	"B.GE":  bformat,
	"B.LO":  bformat,
	"B.LS":  bformat,
	"B.HI":  bformat,
	"B.HS":  bformat,
	"B.MI":  bformat,
	"B.PL":  bformat,
	"B.VS":  bformat,
	"B.VC":  bformat,
	"BL":    bformat,
	"BR":    bformat,
	"CBNZ":  cbformat,
	"CBZ":   cbformat,
	"EOR":   rformat,
	"EORI":  iformat,
	"LDUR":  dformat,
	"LDURB": dformat,
	"LDURH": dformat,
	"LDURS": dformat,
	"LDXR":  dformat,
	"LSL":   iformat,
	"LSR":   iformat,
	"MOVK":  imformat,
	"MOVZ":  imformat,
	"ORR":   rformat,
	"ORRI":  iformat,
	"STUR":  dformat,
	"STURB": dformat,
	"STURH": dformat,
	"STURW": dformat,
	"STXR":  dformat,
	"SUB":   rformat,
	"SUBI":  iformat,
	"SUBIS": iformat,
	"SUBS":  rformat,

	"FADDS": rformat,
	"FADDD": rformat,
	"FCMPS": rformat,
	"FCMPD": rformat,
	"FDIVS": rformat,
	"FDIVD": rformat,
	"FMULS": rformat,
	"FMULD": rformat,
	"FSUBD": rformat,
	"LDURD": dformat,
	"MUL":   rformat,
	"SDIV":  rformat,
	"SMULH": rformat,
	"STURS": dformat,
	"STURD": dformat,
	"UDIV":  rformat,
	"UMULH": rformat,
}
