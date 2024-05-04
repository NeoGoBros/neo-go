package neotest

import (
	"flag"

	"github.com/nspcc-dev/neo-go/pkg/compiler"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
)

var rawCoverage = make(map[scriptHash]*scriptRawCoverage)

type scriptHash = util.Uint160

type scriptRawCoverage struct {
	debugInfo      *compiler.DebugInfo
	offsetsVisited []int
}

type coverBlock struct {
	startLine uint // Line number for block start.
	startCol  uint // Column number for block start.
	endLine   uint // Line number for block end.
	endCol    uint // Column number for block end.
	stmts     uint // Number of statements included in this block.
	counts    uint
}

type documentName = string

type documentCover struct {
	blocks []coverBlock
}

func isCoverageEnabled() bool {
	enabled := true // TODO: = false
	flag.VisitAll(func(f *flag.Flag) {
		if f.Name == "test.gocoverdir" && f.Value != nil {
			enabled = true
		}
	})
	return enabled
}

func coverageHook() vm.OnExecHook {
	return func(scriptHash util.Uint160, offset int, opcode opcode.Opcode) {
		if cov, ok := rawCoverage[scriptHash]; ok {
			cov.offsetsVisited = append(cov.offsetsVisited, offset)
		}
	}
}

func reportCoverage() {
	for _, scriptRawCoverage := range rawCoverage {

		di := scriptRawCoverage.debugInfo

		var scriptSeqPoints []compiler.DebugSeqPoint
		for _, methodDebugInfo := range di.Methods {
			scriptSeqPoints = append(scriptSeqPoints, methodDebugInfo.SeqPoints...)
		}

		blocks := make(map[int]*coverBlock)
		for _, point := range scriptSeqPoints {
			b := coverBlock{
				startLine: uint(point.StartLine),
				startCol: uint(point.StartCol),
				endLine: uint(point.EndLine),
				endCol: uint(point.EndCol),
				stmts: 1 + uint(point.EndLine) - uint(point.StartLine),
				counts: 0,
			}
			blocks[point.Opcode] = &b
		}

		for _, offset := range scriptRawCoverage.offsetsVisited {
			for _, point := range scriptSeqPoints {
				if point.Opcode == offset {
					blocks[point.Opcode].counts++
				}
			}
		}
	}
}
