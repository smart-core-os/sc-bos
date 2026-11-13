package block

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/exp/slices"
)

// PathSegment represents one part of a path to a block in a data structure.
type PathSegment struct {
	Field     string
	ArrayKey  string
	ArrayElem any // must be comparable
}

func (ps *PathSegment) IsField() bool {
	return ps.Field != "" && ps.ArrayKey == ""
}

func (ps *PathSegment) IsArrayElem() bool {
	return ps.Field == "" && ps.ArrayKey != ""
}

func (ps *PathSegment) MarshalJSON() ([]byte, error) {
	if ps.IsField() {
		return json.Marshal(ps.Field)
	} else if ps.IsArrayElem() {
		return json.Marshal(map[string]any{ps.ArrayKey: ps.ArrayElem})
	} else {
		return nil, ErrInvalidPathSegment
	}
}

func (ps *PathSegment) UnmarshalJSON(data []byte) error {
	var m map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // match the json.Number keys Diff produces
	if err := dec.Decode(&m); err == nil {
		if len(m) != 1 {
			return ErrInvalidPathSegment
		}
		for k, v := range m {
			if !reflect.ValueOf(v).Comparable() {
				return ErrInvalidPathSegment
			}
			ps.ArrayKey = k
			ps.ArrayElem = v
		}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		ps.Field = s
		return nil
	}

	return ErrInvalidPathSegment
}

// Path represents how to reach a block within a data structure.
// Each PathSegment can be a field access or reference to an array element, by key.
//
// Path has a string representation similar to XMLPath or JSONPath.
//
// Examples:
//   - / 				    - root of the data structure
//   - /foo/bar 		    - access field "bar" of field "foo"
//   - /foo[name="bar"]     - access element of array "foo" where key "name" equals "bar"
//   - /foo[id=42] 		    - access element of array "foo" where key "id" equals 42
//   - /foo[name="bar"]/baz - access field "baz" of element of array "foo" where key "name" equals "bar"
//
// Field names and array keys must be quoted if they contain special characters (outside of [A-Za-z0-9_-]).
// Within strings, escape sequences are supported (see strconv.Quote).
//
// The output JSON representation using the string format.
// The input JSON representation can be either the string format or an array of PathSegments.
type Path []PathSegment

//goland:noinspection GoMixedReceiverTypes
func (p *Path) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*p, err = ParsePath(str)
		return err
	}

	var arr []PathSegment
	err := json.Unmarshal(data, &arr)
	if err != nil {
		return err
	}
	*p = arr
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (p Path) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

//goland:noinspection GoMixedReceiverTypes
func (p Path) String() string {
	var b strings.Builder
	for _, ps := range p {
		writePathSegment(&b, ps)
	}
	if b.Len() == 0 {
		b.WriteByte('/')
	}
	return b.String()
}

func writePathSegment(b *strings.Builder, ps PathSegment) {
	if ps.IsField() {
		b.WriteByte('/')
		writeIdentifier(b, ps.Field)
	} else if ps.IsArrayElem() {
		b.WriteByte('[')
		writeIdentifier(b, ps.ArrayKey)
		b.WriteByte('=')
		value, err := json.Marshal(ps.ArrayElem)
		if err != nil {
			panic(err)
		}
		b.Write(value)
		b.WriteByte(']')
	}
}

func writeIdentifier(b *strings.Builder, s string) {
	if requiresQuotes(s) {
		b.WriteString(strconv.Quote(s))
	} else {
		b.WriteString(s)
	}
}

func requiresQuotes(s string) bool {
	return !simpleNameRegexp.MatchString(s)
}

var simpleNameRegexp = regexp.MustCompile(`^[a-zA-Z-_][a-zA-Z-_0-9]*$`)

func ParsePath(p string) (Path, error) {
	if strings.TrimSpace(p) == "/" {
		// single slash is an empty path, this would be rejected by main parsing logic
		return nil, nil
	}

	r := strings.NewReader(p)
	// parse top-level array subscripts
	parsed, err := readSubscripts(r, nil)
	if err != nil {
		return nil, err
	}

	for {
		err := readSeparator(r)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		field, err := readIdentifier(r)
		if err != nil {
			return nil, err
		}

		parsed = append(parsed, PathSegment{Field: field})

		parsed, err = readSubscripts(r, parsed)
		if err != nil {
			return nil, err
		}
	}
	if len(parsed) == 0 {
		return nil, &PathParseError{Where: position(r), Expected: "path segment", Found: "EOF"}
	}
	return parsed, nil
}

func readSeparator(r *strings.Reader) error {
	skipWhitespace(r)
	return matchRune(r, '/')
}

func readIdentifier(r *strings.Reader) (string, error) {
	skipWhitespace(r)
	if first, ok := peek(r); ok && first == '"' {
		return readQuoted(r)
	} else {
		return readBareIdentifier(r)
	}
}

func readBareIdentifier(r *strings.Reader) (string, error) {
	pos := position(r)
	var ident strings.Builder

	initial, _, err := r.ReadRune()
	if err != nil {
		return "", &PathParseError{Where: pos, Expected: "identifier", Err: err}
	}
	if !unicode.IsLetter(initial) {
		return "", &PathParseError{Where: pos, Expected: "identifier", Found: string(initial)}
	}
	ident.WriteRune(initial)

	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}

		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' && ch != '_' {
			_ = r.UnreadRune()
			break
		}
		ident.WriteRune(ch)
	}

	return ident.String(), nil
}

func readQuoted(r *strings.Reader) (string, error) {
	skipWhitespace(r)
	var str string
	_, err := fmt.Fscanf(r, "%q", &str)
	if err != nil {
		return "", &PathParseError{Where: position(r), Expected: "quoted string", Err: err}
	}
	return str, nil
}

func readSubscript(r *strings.Reader) (string, any, error) {
	err := matchRune(r, '[')
	if err != nil {
		return "", nil, err
	}

	key, err := readIdentifier(r)
	if err != nil {
		return "", nil, err
	}

	skipWhitespace(r)
	err = matchRune(r, '=')
	if err != nil {
		return "", nil, err
	}

	value, err := readSubscriptValue(r)
	if err != nil {
		return "", nil, err
	}

	skipWhitespace(r)
	err = matchRune(r, ']')
	if err != nil {
		return "", nil, err
	}

	return key, value, nil
}

func readSubscripts(r *strings.Reader, parsed Path) (Path, error) {
	for {
		skipWhitespace(r)
		if ch, ok := peek(r); !ok || ch != '[' {
			break
		}
		k, v, err := readSubscript(r)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, PathSegment{ArrayKey: k, ArrayElem: v})
	}
	return parsed, nil
}

func readSubscriptValue(r *strings.Reader) (any, error) {
	skipWhitespace(r)
	if ch, ok := peek(r); ok && ch == '"' {
		return readQuoted(r)
	} else {
		var num float64
		_, err := fmt.Fscanf(r, "%f", &num)
		if err != nil {
			return nil, &PathParseError{Where: position(r), Expected: "number or string", Err: err}
		}
		return num, nil
	}
}

func matchRune(r *strings.Reader, expected rune) error {
	ch, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if ch != expected {
		return &PathParseError{Where: position(r), Expected: strconv.QuoteRune(expected), Found: string(ch)}
	}
	return nil
}

func skipWhitespace(r *strings.Reader) {
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return
		}
		if !unicode.IsSpace(ch) {
			_ = r.UnreadRune()
			return
		}
	}
}

func position(r *strings.Reader) int {
	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}
	return int(pos)
}

func peek(r *strings.Reader) (rune, bool) {
	ch, _, err := r.ReadRune()
	if err != nil {
		return 0, false
	}
	_ = r.UnreadRune()
	return ch, true
}

type PathParseError struct {
	Where    int
	Expected string
	Found    string
	Err      error
}

func (e *PathParseError) Unwrap() error {
	return e.Err
}

func (e *PathParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid path: at offset %d expected %s; %v", e.Where, e.Expected, e.Err)
	} else {
		return fmt.Sprintf("invalid path: at offset %d expected %s but found %q", e.Where, e.Expected, e.Found)
	}
}

func ComparePaths(a, b Path) int {
	return slices.CompareFunc(a, b, comparePathSegments)
}

func comparePathSegments(a, b PathSegment) int {
	if a.IsField() && b.IsField() {
		return strings.Compare(a.Field, b.Field)
	} else if a.IsArrayElem() && b.IsArrayElem() {
		if c := strings.Compare(a.ArrayKey, b.ArrayKey); c != 0 {
			return c
		} else {
			return comparePrimitives(a.ArrayElem, b.ArrayElem)
		}
	} else if a.IsField() {
		return -1
	} else if b.IsField() {
		return 1
	} else {
		return 0
	}
}

// compares primitive values a and b
// strings are sorted before other types
// if a and b are not strings, they are compared by their string representation
//
// this is only intended for primitive types (string, numeric, bool) - these are the types valid in path selectors
func comparePrimitives(a, b any) int {
	aStr, okA := a.(string)
	bStr, okB := b.(string)
	if okA && okB {
		return strings.Compare(aStr, bStr)
	} else if okA && !okB {
		return -1
	} else if !okA && okB {
		return 1
	} else {
		// non-strings are compared by their string representation
		return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	}
}
