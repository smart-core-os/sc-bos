package devices

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
)

type pathSegment struct {
	Name    string
	Index   int
	IsIndex bool
}

var (
	emptyPathErr    = errors.New("empty path")
	leadingDotErr   = errors.New("leading '.'")
	parsingPathErr  = errors.New("unable to parse path")
	parsingNameErr  = errors.New("unable to parse path name")
	parsingIndexErr = errors.New("unable to parse path index")
)

// parsePath returns a pathSegment for each part of the path:
//
//   - a property or map dereference, indicated by a Name and IsIndex=false segment
//   - an array index access, indicated by an IsIndex and Index segment
//
// For example `foo.bar[1][2].baz` would return:
//
//	[{Name:"foo"},{Name:"bar"},{Index:1,IsIndex:true},{Index:2,IsIndex:true},{Name:"baz"}]
func parsePath(path string) ([]pathSegment, error) {
	if len(path) == 0 {
		return nil, emptyPathErr
	}
	reader := newPathReader(path)
	var segments []pathSegment

	// special case for the first segment (no leading . required)
	name, r, _ := reader.Until("[.")
	if name != "" {
		segments = append(segments, pathSegment{Name: name})
	} else if r == '.' {
		return segments, leadingDotErr
	}

	for i, r := range reader.Runes() {
		switch r {
		case '.':
			seg, err := parsePathName(reader)
			if err != nil {
				return segments, err
			}
			segments = append(segments, seg)
		case '[':
			reader.Push() // undo consuming the [
			seg, err := parsePathIndex(reader)
			if err != nil {
				return segments, err
			}
			segments = append(segments, seg)
		default:
			return segments, fmt.Errorf("unexpected character %q at %d: %w", r, i, parsingPathErr)
		}
	}
	return segments, nil
}

// parsePathName consumes characters from buf representing a name.
// buf should be positioned at the start of a name before calling this function.
func parsePathName(buf *pathReader) (pathSegment, error) {
	var seg pathSegment
	c := buf.cursor
	seg.Name, _, _ = buf.Until("[.")
	if seg.Name == "" {
		return seg, fmt.Errorf("expecting name at %d: %w", c, parsingNameErr)
	}
	return seg, nil
}

func parsePathIndex(buf *pathReader) (pathSegment, error) {
	seg := pathSegment{IsIndex: true}
	_, err := buf.Pop() // [
	if err != nil {
		return seg, err
	}
	c := buf.cursor
	indexStr, _, err := buf.Until("]")
	if err != nil {
		return seg, fmt.Errorf("expecting closing ] after %d: %w", c, parsingIndexErr)
	}
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return seg, fmt.Errorf("expecting integer at %d: %w", c, parsingIndexErr)
	}
	_, _ = buf.Pop() // ], can ignore err as buf.Until already checked for it
	seg.Index = index
	return seg, nil
}

// pathReader helps us to parse a path into segments.
type pathReader struct {
	path   []rune
	cursor int
	mark   int
}

func newPathReader(p string) *pathReader {
	return &pathReader{
		path: []rune(p),
	}
}

func (p *pathReader) Runes() iter.Seq2[int, rune] {
	return func(yield func(int, rune) bool) {
		for {
			c, err := p.Pop()
			if err != nil {
				return
			}
			if !yield(p.cursor, c) {
				return
			}
		}
	}
}

func (p *pathReader) Peek() (rune, error) {
	if p.cursor >= len(p.path) {
		return 0, io.EOF
	}
	return p.path[p.cursor], nil
}

func (p *pathReader) Pop() (rune, error) {
	if p.cursor >= len(p.path) {
		return 0, io.EOF
	}
	c := p.path[p.cursor]
	p.cursor++
	return c, nil
}

func (p *pathReader) Push() {
	p.cursor--
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *pathReader) ConsumeIf(r rune) bool {
	if p.cursor >= len(p.path) {
		return false
	}
	if p.path[p.cursor] != r {
		return false
	}
	p.cursor++
	return true
}

func (p *pathReader) Len() int {
	return len(p.path) - p.cursor
}

// Mark marks the current cursor position, returning the consumed runes since the last mark.
func (p *pathReader) Mark() string {
	s := string(p.path[p.mark:p.cursor])
	p.mark = p.cursor
	return s
}

// Until consumes runes until one of the specified runes are found.
// If no rune is found, Until consumes all remaining runes, returning the consumed runes, and an error.
func (p *pathReader) Until(candidates string) (string, rune, error) {
	var consumed strings.Builder
	for _, c := range p.Runes() {
		if strings.ContainsRune(candidates, c) {
			p.Push() // undo consuming the rune
			return consumed.String(), c, nil
		}
		consumed.WriteRune(c)
	}
	return consumed.String(), 0, fmt.Errorf("expecting one of %q", candidates)
}
