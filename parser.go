package simleg

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
)

type Parser struct {
	l  *lexer
	pk *item
}

func (p *Parser) Use(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	p.l = lex(string(b))
	return nil
}

func (p *Parser) nextItem() item {
	if p.pk != nil {
		i := *p.pk
		p.pk = nil
		return i
	}
	return p.l.nextItem()
}

func (p *Parser) peek() item {
	i := p.l.nextItem()
	p.pk = &i
	return i
}

func (p *Parser) expect(typ itemType) (string, error) {
	t := p.nextItem()
	if t.typ == itemEOF {
		return "", io.EOF
	}
	if t.typ != typ {
		return "", fmt.Errorf("unexpected token '%s'", t.text)
	}
	return t.text, nil
}

func (p *Parser) has(typ itemType) bool {
	return p.peek().typ == typ
}

func (p *Parser) Next() (as Instruction, err error) {
	name, err := p.expect(itemName)
	if err != nil {
		return as, err
	}
	if p.has(itemColon) {
		p.expect(itemColon)
		as.Label = name
		op, err := p.expect(itemName)
		if err != nil {
			return as, err
		}
		as.Op = op
	} else {
		as.Op = name
	}
	f, ok := opcodes[as.Op]
	if !ok {
		return as, fmt.Errorf("opcode not supported: %s", as.Op)
	}
	if f.p == nil {
		panic("opcode: " + as.Op + " missing parser")
	}
	if err = f.p(p, &as); err != nil {
		return as, err
	}
	return as, nil
}

type formatParser func(p *Parser, as *Instruction) error

func rformatString(w io.Writer, as Instruction) {
	fmt.Fprintf(w, "%s,%s,%s", as.To.Reg, as.From.Reg, as.Reg)
}

func rformatParser(p *Parser, as *Instruction) (err error) {
	as.To.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.From.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("first operand: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("second operand: %v", err)
	}
	return nil
}

func dformatString(w io.Writer, as Instruction) {
	fmt.Fprintf(w, "%s,", as.To.Reg)
	offsetString(w, as.From)
}

func dformatParser(p *Parser, as *Instruction) (err error) {
	as.To.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.From, err = p.expectOffset(as)
	if err != nil {
		return fmt.Errorf("from: %v", err)
	}
	return nil
}

func iformatString(w io.Writer, as Instruction) {
	fmt.Fprintf(w, "%s,%s,#%d", as.To.Reg, as.From.Reg, as.Imm)
}

func iformatParser(p *Parser, as *Instruction) (err error) {
	as.To.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.From.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("first operand: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.Imm, err = p.expectImmediate(16)
	if err != nil {
		return fmt.Errorf("immediate: %v", err)
	}
	return nil
}

func bformatString(w io.Writer, as Instruction) {
	if as.To.Label != "" {
		fmt.Fprint(w, as.To.Label)
		return
	}
	fmt.Fprint(w, as.To.Offset)
}

func bformatParser(p *Parser, as *Instruction) (err error) {
	if as.Op == "BR" {
		as.To.Reg, err = p.expectRegister(as.registerPrefix())
		if err != nil {
			return fmt.Errorf("to: %v", err)
		}
		return nil
	}
	as.To, err = p.expectAddr(as)
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	return nil
}

func cbformatString(w io.Writer, as Instruction) {
	fmt.Fprintf(w, "%s,%s", as.From.Reg, as.To.Label)
}

func cbformatParser(p *Parser, as *Instruction) (err error) {
	as.From.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("from: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.To.Label, err = p.expect(itemName)
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	return nil
}

func iwformatString(w io.Writer, as Instruction) {
	fmt.Fprintf(w, "%s,#%d", as.To.Reg, as.Imm)
}

func iwformatParser(p *Parser, as *Instruction) (err error) {
	as.To.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return fmt.Errorf("to: %v", err)
	}
	if _, err = p.expect(itemComma); err != nil {
		return err
	}
	as.Imm, err = p.expectImmediate(32)
	if err != nil {
		return fmt.Errorf("immediate: %v", err)
	}
	return nil
}

func (p *Parser) expectImmediate(bitsize int) (uint64, error) {
	imm, err := p.expect(itemInteger)
	if err != nil {
		return 0, fmt.Errorf("not an integer: %v", err)
	}
	if imm[0] == '#' {
		imm = imm[1:]
	}
	n, err := strconv.ParseUint(imm, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return n, nil
}

func offsetString(w io.Writer, addr Addr) {
	fmt.Fprintf(w, "[%s,#%d]", addr.Reg, addr.Offset)
}

func (p *Parser) expectOffset(as *Instruction) (addr Addr, err error) {
	if _, err := p.expect(itemLbrack); err != nil {
		return addr, err
	}
	addr.Reg, err = p.expectRegister(as.registerPrefix())
	if err != nil {
		return addr, err
	}
	if _, err := p.expect(itemComma); err != nil {
		return addr, err
	}
	addr.Offset, err = p.expectImmediate(32)
	if err != nil {
		return addr, fmt.Errorf("offset: %v", err)
	}
	if _, err := p.expect(itemRbrack); err != nil {
		return addr, err
	}
	return addr, nil
}

func (p *Parser) expectAddr(as *Instruction) (addr Addr, err error) {
	if p.has(itemInteger) {
		// PC-relative address
		off, err := p.expect(itemInteger)
		if err != nil {
			return addr, fmt.Errorf("to: %v", err)
		}
		addr.Offset, err = strconv.ParseUint(off, 10, 64)
		if err != nil {
			return addr, fmt.Errorf("to: %v", err)
		}
		return addr, nil
	}
	addr.Label, err = p.expect(itemName)
	if err != nil {
		return addr, fmt.Errorf("to: %v", err)
	}
	return addr, nil
}

func (p *Parser) expectRegister(prefix rune) (Register, error) {
	t, err := p.expect(itemName)
	if err != nil {
		return 0, err
	}
	if len(t) > 3 {
		return 0, fmt.Errorf("not a register '%s'", t)
	}
	var base Register
	var max uint64
	switch prefix {
	case 'X':
		base = X0
		max = 30
		break
	case 'S':
		base = S0
		max = 31
	case 'D':
		base = D0
		max = 31
	default:
		return 0, fmt.Errorf("expecting register type '%c'", prefix)
	}
	if prefix == 'X' {
		// might be a special purpose
		switch {
		case t == "SP":
			return SP, nil
		case t == "FP":
			return FP, nil
		case t == "LR":
			return LR, nil
		case t == "XZR":
			return XZR, nil
		case t == "IP0":
			return IP0, nil
		case t == "IP1":
			return IP1, nil
		}
	}
	if rune(t[0]) != prefix {
		return 0, fmt.Errorf("not a register '%s'", t)
	}
	n, err := strconv.ParseUint(t[1:], 10, 8)
	if err != nil || n > max {
		return 0, fmt.Errorf("not a register '%s'", t)
	}
	return base + Register(n), nil
}
