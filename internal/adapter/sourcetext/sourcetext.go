// Package sourcetext holds the text-level helper shared by all language
// adapters: comment-aware counting of source lines.
package sourcetext

import (
	"sort"
	"strings"
)

// Range is a half-open byte range [Start, End) in the source text.
type Range struct {
	Start int
	End   int
}

// CountSourceLines counts nonblank, noncomment lines of text[start:end]. Comments
// must be sorted by Start, as every adapter's lexer produces them.
func CountSourceLines(text string, start, end int, comments []Range) int {
	if end > len(text) {
		end = len(text)
	}
	if end <= start {
		return 0
	}
	overlapping := commentsWithin(comments, start, end)
	if len(overlapping) == 0 {
		return countNonBlank(text[start:end])
	}
	segment := []byte(text[start:end])
	for _, comment := range overlapping {
		from := max(comment.Start, start) - start
		to := min(comment.End, end) - start
		for index := from; index < to; index++ {
			if segment[index] != '\n' && segment[index] != '\r' {
				segment[index] = ' '
			}
		}
	}
	return countNonBlank(string(segment))
}

// commentsWithin narrows the comments to those touching [start, end), so counting
// every function of a large file is not quadratic.
func commentsWithin(comments []Range, start, end int) []Range {
	first := sort.Search(len(comments), func(i int) bool { return comments[i].End > start })
	last := first
	for last < len(comments) && comments[last].Start < end {
		last++
	}
	return comments[first:last]
}

func countNonBlank(segment string) int {
	count := 0
	for len(segment) > 0 {
		line := segment
		if cut := strings.IndexByte(segment, '\n'); cut >= 0 {
			line, segment = segment[:cut], segment[cut+1:]
		} else {
			segment = ""
		}
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
