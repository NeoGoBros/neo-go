package neotest

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/compiler"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/stretchr/testify/require"
)

type CoverageCollector struct {
	instructions []InstrHash
}

type InstrHash struct {
	Offset     int
	Opcode     opcode.Opcode
	ScriptHash util.Uint160
}

func NewCoverageCollector() *CoverageCollector {
	return &CoverageCollector{make([]InstrHash, 0)}
}

func (cc *CoverageCollector) GetOnExecHook() vm.OnExecHook {
	return func(scriptHash util.Uint160, offset int, opcode opcode.Opcode) {
		instr := InstrHash{offset, opcode, scriptHash}
		cc.instructions = append(cc.instructions, instr)
	}
}

func (cc *CoverageCollector) ExportTestCoverage(t testing.TB, contract *Contract, outputFilePath string) {
	makeCoverage(t, cc.instructions, contract, contract.DebugInfo.MainDocument, outputFilePath)
}

// --- Legacy Code ---

type Cover struct {
	sync.Mutex
	testing.Cover
	Opcodes map[string][]int
}

var cover = &Cover{}

// MakeCoverage generates an output file with coverage info in correct format.
func makeCoverage(
	t testing.TB,
	instructions []InstrHash,
	ctrdi *Contract,
	substr string,
	fileName string,
) {
	cover.Lock()
	defer cover.Unlock()
	if cover.Opcodes == nil {
		cover.Opcodes = make(map[string][]int)
		cover.Blocks = make(map[string][]testing.CoverBlock)
		cover.Counters = make(map[string][]uint32)
	}

	docs := getDocuments(t, ctrdi.DebugInfo.Documents, substr)
	setBlocks(ctrdi.DebugInfo, docs)
	countInstructions(ctrdi.DebugInfo.Documents, docs, instructions)
	printToFile(t, fileName)
}

// getDocuments returns compiler.DebugInfo.Documents indexes which contain specific substring.
func getDocuments(t testing.TB, docs []string, substr string) []int {
	res := make([]int, 0, len(docs))

	for i, cDoc := range docs {
		if strings.Contains(cDoc, substr) {
			res = append(res, i)
		}
	}
	if len(res) == 0 {
		t.Log("Cannot get document\n")
		t.FailNow()
	}
	return res
}

// setBlocks extracts sequence points for every specific document
// from compiler.DebugInfo and stores them in testing.Cover format
func setBlocks(di *compiler.DebugInfo, docs []int) {
	for _, method := range di.Methods {
		for _, seqPoint := range method.SeqPoints {
			statements := seqPoint.EndLine - seqPoint.StartLine + 1
			docStr := di.Documents[seqPoint.Document]
			if isValidDocument(seqPoint.Document, docs) {
				cover.Blocks[docStr] = append(cover.Blocks[docStr], testing.CoverBlock{
					Line0: uint32(seqPoint.StartLine),
					Col0:  uint16(seqPoint.StartCol),
					Line1: uint32(seqPoint.EndLine),
					Col1:  uint16(seqPoint.EndCol),
					Stmts: uint16(statements),
				})
				cover.Opcodes[docStr] = append(cover.Opcodes[docStr], seqPoint.Opcode)
			}
		}
	}
	for doc, seqPoints := range cover.Blocks {
		if hasNoCounters(doc) {
			cover.Counters[doc] = make([]uint32, len(seqPoints))
		}
	}
}

// isValidDocument checks if document index exists in an array of document indexes.
func isValidDocument(iDocToCheck int, docs []int) bool {
	for _, iDoc := range docs {
		if iDoc == iDocToCheck {
			return true
		}
	}
	return false
}

func hasNoBlocks(doc string) bool {
	if _, exists := cover.Blocks[doc]; exists {
		return false
	}
	return true
}

func hasNoCounters(doc string) bool {
	if _, exists := cover.Counters[doc]; exists {
		return false
	}
	return true
}

// countInstructions finds for every instruction a corresponding sequence point
// and sets IsCovered flag to true.
func countInstructions(diDocs []string, validDocs []int, instrs []InstrHash) {
	sValidDocs := getValidStrDocs(diDocs, validDocs)
	for doc, ops := range cover.Opcodes {
		if isValidStrDoc(doc, sValidDocs) {
			cover.Counters[doc] = getNewCounts(doc, ops, instrs)
		}
	}
}

func getNewCounts(doc string, ops []int, instrs []InstrHash) []uint32 {
	counts := cover.Counters[doc]
	for _, instr := range instrs {
		for i, op := range ops {
			if instr.Offset == op {
				counts[i]++
			}
		}
	}
	return counts
}

func getValidStrDocs(diDocs []string, validDocs []int) []string {
	res := make([]string, len(validDocs))
	for i, iDoc := range validDocs {
		res[i] = diDocs[iDoc]
	}
	return res
}

func isValidStrDoc(docToCheck string, docs []string) bool {
	for _, doc := range docs {
		if doc == docToCheck {
			return true
		}
	}
	return false
}

// printToFile writes coverage info to file.
func printToFile(t testing.TB, name string) {
	fileName := name
	if fileName == "" {
		// if no specific file name was provided, the output file
		// will have formatted timestamp with no spaces as its name
		fileName = fmt.Sprintf("%s.out", time.Now().Format(time.RFC3339Nano))
	}

	f, err := os.OpenFile(fileName, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.WriteString("mode: set\n")
	require.NoError(t, err)

	var count uint32
	for name, counts := range cover.Counters {
		blocks := cover.Blocks[name]
		for i := range counts {
			stmts := int64(blocks[i].Stmts)
			count = counts[i]

			// mode: set
			if count != 0 {
				count = 1
			}
			if f != nil && stmts == 1 {
				_, err := fmt.Fprintf(f, "%s:%d.%d,%d.%d %d %d\n", name,
					blocks[i].Line0, blocks[i].Col0,
					blocks[i].Line1, blocks[i].Col1,
					stmts,
					count)
				require.NoError(t, err)
			}
		}
	}

}
