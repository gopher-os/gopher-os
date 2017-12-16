package parser

import "testing"

// TestFindUnmappedOpcodes is a helper test that pinpoints opcodes that have
// not yet been mapped via an opcode table.
func TestFindUnmappedOpcodes(t *testing.T) {
	for opIndex, opRef := range opcodeMap {
		if opRef != badOpcode {
			continue
		}

		for tabIndex, info := range opcodeTable {
			if uint16(info.op) == uint16(opIndex) {
				t.Errorf("set opcodeMap[0x%02x] = 0x%02x // %s\n", opIndex, tabIndex, info.op.String())
				break
			}
		}
	}

	for opIndex, opRef := range extendedOpcodeMap {
		// 0xff (opOnes) is defined in opcodeTable
		if opRef != badOpcode || opIndex == 0 {
			continue
		}

		opIndex += 0xff
		for tabIndex, info := range opcodeTable {
			if uint16(info.op) == uint16(opIndex) {
				t.Errorf("set extendedOpcodeMap[0x%02x] = 0x%02x // %s\n", opIndex-0xff, tabIndex, info.op.String())
				break
			}
		}
	}
}
