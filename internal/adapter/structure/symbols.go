package structure

import (
	"fmt"
	"sort"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

// Record is a function with the scope that names it. The symbol itself lives
// in Scope.Symbol; holding a second copy only invites the two to drift.
type Record struct {
	Function *Function
	Scope    domain.Scope
}

// AssignSymbols orders functions by position and gives each a unique symbol, with
// "#n" for repeats. One rule for every language keeps finding identities stable.
func AssignSymbols(relativePath string, functions []*Function) []Record {
	ordered := append([]*Function(nil), functions...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Start < ordered[j].Start })
	occurrences := make(map[string]int, len(ordered))
	records := make([]Record, 0, len(ordered))
	for _, function := range ordered {
		base := function.Prefix() + function.Name
		occurrences[base]++
		symbol := base
		if occurrences[base] > 1 {
			symbol = fmt.Sprintf("%s#%d", base, occurrences[base])
		}
		records = append(records, Record{
			Function: function,
			Scope:    domain.FunctionScope(relativePath, symbol, function.Line, function.EndLine),
		})
	}
	return records
}
