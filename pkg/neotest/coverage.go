package neotest

import (
	"flag"

	"github.com/nspcc-dev/neo-go/pkg/compiler"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
)

type scriptHash = util.Uint160

type scriptCoverage struct {
	debugInfo *compiler.DebugInfo
	offsetsVisited []int
}

var coverage = make(map[scriptHash]*scriptCoverage)

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
		if cov, ok := coverage[scriptHash]; ok {
			cov.offsetsVisited = append(cov.offsetsVisited, offset)
		}
	}
}
