package java

import (
	"fmt"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"

	"github.com/lumiostack/lumioguard-cc/internal/adapter/java/syntax"
)

// parsedFile bundles the parse tree with the token stream and the offset
// tables the adapter needs.
type parsedFile struct {
	text   string
	tree   syntax.ICompilationUnitContext
	tokens []antlr.Token
	// runeToByte maps ANTLR character (rune) indices to byte offsets, and is
	// nil for the common case of an all-ASCII file, where they are equal.
	runeToByte []int
	// symbolic names the lexer's token types.
	symbolic []string
}

type syntaxError struct {
	line    int
	message string
}

func (e *syntaxError) Error() string {
	return fmt.Sprintf("%s (line %d)", e.message, e.line)
}

// errorListener records the syntax errors of the lexer and the parser.
type errorListener struct {
	*antlr.DefaultErrorListener
	errors []syntaxError
}

func newErrorListener() *errorListener {
	return &errorListener{DefaultErrorListener: antlr.NewDefaultErrorListener()}
}

// SyntaxError implements antlr.ErrorListener.
func (l *errorListener) SyntaxError(_ antlr.Recognizer, _ any, line, _ int, message string, _ antlr.RecognitionException) {
	l.errors = append(l.errors, syntaxError{line: line, message: message})
}

func (l *errorListener) first() *syntaxError {
	if len(l.errors) > 0 {
		return &l.errors[0]
	}
	return &syntaxError{line: 1, message: "parse failed"}
}

// reportQuietly reports errors like the default strategy but never prints to
// stdout, which would corrupt JSON output.
func reportQuietly(strategy *antlr.DefaultErrorStrategy, recognizer antlr.Parser, e antlr.RecognitionException) {
	if strategy.InErrorRecoveryMode(recognizer) {
		return
	}
	switch t := e.(type) {
	case *antlr.NoViableAltException:
		strategy.ReportNoViableAlternative(recognizer, t)
	case *antlr.InputMisMatchException:
		strategy.ReportInputMisMatch(recognizer, t)
	case *antlr.FailedPredicateException:
		strategy.ReportFailedPredicate(recognizer, t)
	default:
		recognizer.NotifyErrorListeners(e.GetMessage(), e.GetOffendingToken(), e)
	}
}

// quietStrategy is the default error strategy with quiet reporting.
type quietStrategy struct {
	*antlr.DefaultErrorStrategy
}

// ReportError implements antlr.ErrorStrategy.
func (s *quietStrategy) ReportError(recognizer antlr.Parser, e antlr.RecognitionException) {
	reportQuietly(s.DefaultErrorStrategy, recognizer, e)
}

// parse runs one full LL-prediction pass. Errors reach the listener, never stdout.
func parse(text string) (file *parsedFile, failure *syntaxError) {
	listener := newErrorListener()
	defer func() {
		if recover() != nil {
			file, failure = nil, listener.first()
		}
	}()
	lexer := syntax.NewJavaLexer(antlr.NewInputStream(text))
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := syntax.NewJavaParser(stream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(listener)
	parser.GetInterpreter().SetPredictionMode(antlr.PredictionModeLL)
	parser.SetErrorHandler(&quietStrategy{DefaultErrorStrategy: antlr.NewDefaultErrorStrategy()})
	tree := parser.CompilationUnit()
	if len(listener.errors) > 0 {
		return nil, listener.first()
	}
	stream.Fill()
	return &parsedFile{
		text:       text,
		tree:       tree,
		tokens:     stream.GetAllTokens(),
		runeToByte: runeOffsets(text),
		symbolic:   lexer.GetSymbolicNames(),
	}, nil
}

// runeOffsets maps rune indices to byte offsets, plus one entry for the end.
// It returns nil for all-ASCII text, where the two indices are equal.
func runeOffsets(text string) []int {
	ascii := true
	for i := 0; i < len(text); i++ {
		if text[i] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		return nil
	}
	offsets := make([]int, 0, len(text)+1)
	for offset := range text {
		offsets = append(offsets, offset)
	}
	return append(offsets, len(text))
}

// byteOffset converts an ANTLR character index to a byte offset.
func (p *parsedFile) byteOffset(runeIndex int) int {
	if runeIndex < 0 {
		return 0
	}
	if p.runeToByte == nil {
		return min(runeIndex, len(p.text))
	}
	if runeIndex >= len(p.runeToByte) {
		return len(p.text)
	}
	return p.runeToByte[runeIndex]
}

// startOffset and endOffset return the byte span of a parse tree node.
func (p *parsedFile) startOffset(ctx antlr.ParserRuleContext) int {
	if ctx == nil || ctx.GetStart() == nil {
		return 0
	}
	return p.byteOffset(ctx.GetStart().GetStart())
}

func (p *parsedFile) endOffset(ctx antlr.ParserRuleContext) int {
	if ctx == nil || ctx.GetStop() == nil {
		return 0
	}
	return p.byteOffset(ctx.GetStop().GetStop() + 1)
}

func (p *parsedFile) tokenName(tokenType int) string {
	if tokenType >= 0 && tokenType < len(p.symbolic) && p.symbolic[tokenType] != "" {
		return p.symbolic[tokenType]
	}
	return fmt.Sprintf("T%d", tokenType)
}

func lineOf(ctx antlr.ParserRuleContext) int {
	if ctx == nil || ctx.GetStart() == nil {
		return 0
	}
	return ctx.GetStart().GetLine()
}

func endLineOf(ctx antlr.ParserRuleContext) int {
	if ctx == nil || ctx.GetStop() == nil {
		return lineOf(ctx)
	}
	return ctx.GetStop().GetLine()
}
