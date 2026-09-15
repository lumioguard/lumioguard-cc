// Package coverage imports LCOV tracefiles onto analyzed files. It never runs
// tests and never infers branch coverage from line coverage.
package coverage

import (
	"bufio"
	"io"
	"iter"
	"strconv"
	"strings"
)

// Branch is one LCOV BRDA outcome. Taken is nil for the "-" (never evaluated) marker.
type Branch struct {
	Line  int
	Taken *int
}

// FileRecord is the coverage of one SF section.
type FileRecord struct {
	Source   string
	Lines    map[int]int
	Branches []Branch
}

// ParseLCOV reads SF, DA, BRDA and end_of_record entries. Other record types
// are ignored; malformed numeric fields skip the entry.
func ParseLCOV(r io.Reader) ([]FileRecord, error) {
	var records []FileRecord
	var current *FileRecord
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		switch {
		case strings.HasPrefix(line, "SF:"):
			current = &FileRecord{Source: line[3:], Lines: map[int]int{}}
		case strings.HasPrefix(line, "DA:") && current != nil:
			fields := strings.Split(line[3:], ",")
			if len(fields) < 2 {
				continue
			}
			number, errLine := strconv.Atoi(fields[0])
			hits, errHits := strconv.Atoi(fields[1])
			if errLine == nil && errHits == nil {
				current.Lines[number] = hits
			}
		case strings.HasPrefix(line, "BRDA:") && current != nil:
			fields := strings.Split(line[5:], ",")
			if len(fields) < 1 {
				continue
			}
			number, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			branch := Branch{Line: number}
			if len(fields) >= 4 && fields[3] != "-" {
				if taken, err := strconv.Atoi(fields[3]); err == nil {
					branch.Taken = &taken
				}
			}
			current.Branches = append(current.Branches, branch)
		case line == "end_of_record" && current != nil:
			records = append(records, *current)
			current = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		records = append(records, *current)
	}
	return records, nil
}

// KnownBranches iterates the outcomes the instrumenter evaluated at least once.
// It yields instead of copying because callers only count and filter.
func (r FileRecord) KnownBranches() iter.Seq[Branch] {
	return func(yield func(Branch) bool) {
		for _, branch := range r.Branches {
			if branch.Taken == nil {
				continue
			}
			if !yield(branch) {
				return
			}
		}
	}
}
