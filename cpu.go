package simleg

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"strings"
)

// Memory offsets
const (
	StackOffset = 0x500000
)

type condFlag uint8

const (
	flagN condFlag = 1 << iota
	flagZ
	flagV
	flagC
)

type CPU struct {
	PC        uint64
	Registers [32]uint64
	Flags     condFlag
	Err       error

	Memory *Memory

	labels map[string]uint64
	prog   []Instruction
}

func (cpu *CPU) Load(prog Program) error {
	cpu.Memory = &Memory{}

	for i := 0; i < len(cpu.Registers); i++ {
		cpu.Registers[i] = random.Uint64()
	}

	cpu.labels = make(map[string]uint64)
	cpu.prog = prog
	for i, as := range prog {
		if as.Label != "" {
			cpu.labels[as.Label] = uint64(i)
		}
	}
	return nil
}

// Step runs the instruction that PC points to.
func (cpu *CPU) Step() bool {
	if len(cpu.prog) == 0 {
		return false
	}
	as := cpu.prog[cpu.PC]
	switch {
	case cpu.arith(as):
		cpu.PC++
		break
	case cpu.branch(as):
		break
	case cpu.memory(as):
		cpu.PC++
		break
	}
	return cpu.PC < uint64(len(cpu.prog))
}

func (cpu CPU) valuesFor(as Instruction) (dst Register, a, b uint64) {
	switch {
	case strings.HasSuffix(as.Op, "I"):
		return as.To.Reg, cpu.Registers[as.From.Reg], as.Imm
	default:
		return as.To.Reg, cpu.Registers[as.From.Reg], cpu.Registers[as.Reg]
	}
}

func (cpu *CPU) setFlags(as Instruction, carry uint64) {
	if !strings.HasSuffix(as.Op, "S") {
		return
	}
	v := cpu.Registers[as.To.Reg]
	switch {
	case v == 0:
		cpu.Flags |= flagZ
		fallthrough
	case v < 0:
		cpu.Flags |= flagN
		fallthrough
	case carry == 1:
		cpu.Flags |= flagC
		return
	default:
		cpu.Flags = 0
	}
}

func (cpu *CPU) arith(as Instruction) bool {
	var carry uint64
	dst, x, y := cpu.valuesFor(as)
	switch {
	case strings.HasPrefix(as.Op, "ADD"):
		cpu.Registers[dst], carry = bits.Add64(x, y, 0)
		cpu.setFlags(as, carry)
		return true
	case strings.HasPrefix(as.Op, "SUB"):
		cpu.Registers[dst], carry = bits.Sub64(x, y, 0)
		cpu.setFlags(as, carry)
		return true
	case strings.HasPrefix(as.Op, "EOR"):
		cpu.Registers[dst] = x ^ y
		cpu.setFlags(as, 0)
		return true
	case strings.HasPrefix(as.Op, "ORR"):
		cpu.Registers[dst] = x | y
		cpu.setFlags(as, 0)
		return true
	case strings.HasPrefix(as.Op, "AND"):
		cpu.Registers[dst] = x & y
		cpu.setFlags(as, 0)
		return true
	case as.Op == "LSL":
		cpu.Registers[dst] = x << y
		return true
	case as.Op == "LSR":
		cpu.Registers[dst] = x >> y
		return true
	case as.Op == "MUL":
		cpu.Registers[dst] = x * y
		return true
	default:
		return false
	}
}

func (cpu CPU) shouldBranch(cond string) (bool, error) {
	val := func(f condFlag) condFlag { return cpu.Flags & f }
	switch cond {
	case "EQ":
		return val(flagZ) == 1, nil
	case "NE":
		return val(flagZ) == 0, nil
	case "LT":
		return val(flagN) != val(flagV), nil
	case "LE":
		return !(val(flagZ) == 0 && val(flagN) == val(flagV)), nil
	case "GT":
		return val(flagZ) == 0 && val(flagN) == val(flagV), nil
	case "GE":
		return val(flagN) == val(flagV), nil
	case "LO":
		return val(flagC) == 0, nil
	case "LS":
		return !(val(flagZ) == 0 && val(flagC) == 1), nil
	case "HI":
		return val(flagZ) == 0 && val(flagC) == 1, nil
	case "HS":
		return val(flagC) == 1, nil
	default:
		return false, errors.New("unknown comparison")
	}
}

func (cpu *CPU) branch(as Instruction) bool {
	addr := func(to Addr) uint64 {
		if to.Label != "" {
			return cpu.labels[to.Label]
		}
		return cpu.PC + to.Offset
	}
	switch {
	case as.Op == "B":
		cpu.PC = addr(as.To)
	case as.Op == "BR":
		cpu.PC = cpu.Registers[as.To.Reg]
		return true
	case as.Op == "BL":
		cpu.Registers[LR] = uint64(cpu.PC)
		cpu.PC = addr(as.To)
		return true
	case as.Op == "CBZ":
		if cpu.Registers[as.To.Reg] != 0 {
			cpu.PC++
			return true
		}
		cpu.PC = addr(as.From)
		return true
	case as.Op == "CBNZ":
		if cpu.Registers[as.To.Reg] == 0 {
			cpu.PC++
			return true
		}
		cpu.PC = addr(as.From)
		return true
	case strings.HasPrefix(as.Op, "B."):
		cond := as.Op[len("B."):]
		ok, err := cpu.shouldBranch(cond)
		if err != nil {
			panic(err)
		}
		if !ok {
			cpu.PC++
			return true
		}
		cpu.PC = addr(as.To)
		return true
	default:
		return false
	}
	return false
}

func (cpu *CPU) memory(as Instruction) bool {
	switch {
	case as.Op == "STUR":
		var d [8]byte
		binary.LittleEndian.PutUint64(d[:], cpu.Registers[as.To.Reg])
		cpu.Memory.Write(d[:], cpu.Registers[as.From.Reg]+as.From.Offset)
		return true
	case as.Op == "LDUR":
		var d [8]byte
		cpu.Memory.Read(d[:], cpu.Registers[as.From.Reg]+as.From.Offset)
		v := binary.LittleEndian.Uint64(d[:])
		cpu.Registers[as.To.Reg] = v
		return true
	}
	return false
}
