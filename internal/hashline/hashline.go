// Package hashline provides line-level content hashing for precise code editing.
// Each line is annotated with a 4-char FNV-1a hash when read. The LLM uses these
// anchors to specify edit ranges instead of copying exact text for replacement.
// Line format: "42#VKQR|content of line 42".
// For lines without letters/digits, the line number seeds the hash to prevent
// collisions across blank lines.
package hashline

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

// NibbleStr is the 16-character alphabet for 4-char hash encoding (65536 unique combos).
const NibbleStr = "ZPMQVRWSNKTXJBYH"

// HashLen is the number of characters in a line hash.
const HashLen = 4

var hashlineDict [65536]string

func init() {
	for i := range 65536 {
		b0 := NibbleStr[(i>>12)&0xf]
		b1 := NibbleStr[(i>>8)&0xf]
		b2 := NibbleStr[(i>>4)&0xf]
		b3 := NibbleStr[i&0xf]
		hashlineDict[i] = string([]byte{b0, b1, b2, b3})
	}
}

var reSignificant = regexp.MustCompile(`[\pL\pN]`)

func ComputeLineHash(lineNumber int, content string) string {
	normalized := strings.TrimRight(strings.ReplaceAll(content, "\r", ""), " \t")
	seed := 0
	if !reSignificant.MatchString(normalized) {
		seed = lineNumber
	}
	h := fnv.New32a()
	if seed != 0 {
		var seedBytes [4]byte
		seedBytes[0] = byte(seed >> 24)
		seedBytes[1] = byte(seed >> 16)
		seedBytes[2] = byte(seed >> 8)
		seedBytes[3] = byte(seed)
		h.Write(seedBytes[:])
	}
	h.Write([]byte(normalized))
	index := h.Sum32() % 65536
	return hashlineDict[index]
}

// FormatHashLine formats a single line with its hash annotation.
func FormatHashLine(lineNumber int, content string) string {
	hash := ComputeLineHash(lineNumber, content)
	return formatLineRef(lineNumber, hash, content)
}

// FormatHashLines formats multi-line content with hash annotations (1-based startLine).
func FormatHashLines(content string, startLine int) string {
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		lineNum := i + startLine
		result[i] = FormatHashLine(lineNum, line)
	}
	return strings.Join(result, "\n")
}

// LineRef is a parsed line reference (line number + hash).
type LineRef struct {
	Line int
	Hash string
}

func ParseLineRef(ref string) (LineRef, error) {
	trimmed := strings.TrimSpace(ref)
	trimmed = stripLLMPrefix(trimmed)
	trimmed = normalizeHashSeparator(trimmed)
	trimmed = stripPipeSuffix(trimmed)
	trimmed = strings.TrimSpace(trimmed)

	if lr, ok := tryParseRef(trimmed); ok {
		return lr, nil
	}

	if extracted := extractLineRef(trimmed); extracted != "" {
		if lr, ok := tryParseRef(extracted); ok {
			return lr, nil
		}
	}

	return LineRef{}, errInvalidLineRef(ref)
}

// ResolvedLineRef is a line reference resolved to the actual line in the file.
// If the hash did not match at the original line number, ResolvedLine indicates
// where the hash was found via full-text search.
type ResolvedLineRef struct {
	LineRef
	ResolvedLine int
	AutoResolved bool
}

// ResolveLineRef resolves a line reference against file content.
// If the hash matches at the given line number, it returns immediately.
// If not, it searches the entire file for a line with matching hash.
// Returns an error if the hash is not found anywhere, or found multiple times.
func ResolveLineRef(lines []string, ref string) (ResolvedLineRef, error) {
	lr, err := ParseLineRef(ref)
	if err != nil {
		return ResolvedLineRef{}, err
	}

	// Try exact match first.
	if lr.Line >= 1 && lr.Line <= len(lines) {
		actual := ComputeLineHash(lr.Line, lines[lr.Line-1])
		if actual == lr.Hash {
			return ResolvedLineRef{LineRef: lr, ResolvedLine: lr.Line, AutoResolved: false}, nil
		}
	}

	// Full-text search for the hash.
	var candidates []int
	for i, line := range lines {
		lineNum := i + 1
		if ComputeLineHash(lineNum, line) == lr.Hash {
			candidates = append(candidates, lineNum)
		}
	}

	switch len(candidates) {
	case 0:
		return ResolvedLineRef{}, fmt.Errorf("line %d hash %s not found in file (line content was changed or deleted)", lr.Line, lr.Hash)
	case 1:
		return ResolvedLineRef{LineRef: lr, ResolvedLine: candidates[0], AutoResolved: true}, nil
	default:
		return ResolvedLineRef{}, fmt.Errorf("hash %s matches multiple lines (%v); provide more context or re-read the file", lr.Hash, candidates)
	}
}

// ValidateLineRef validates that a line reference matches actual file content.
func ValidateLineRef(lines []string, ref string) error {
	lr, err := ParseLineRef(ref)
	if err != nil {
		return err
	}
	if lr.Line < 1 || lr.Line > len(lines) {
		return errLineOutOfBounds(lr.Line, len(lines))
	}
	actual := ComputeLineHash(lr.Line, lines[lr.Line-1])
	if actual != lr.Hash {
		return &MismatchError{
			Mismatches: []HashMismatch{
				{Line: lr.Line, Expected: lr.Hash, Actual: actual},
			},
			FileLines: lines,
		}
	}
	return nil
}

// ValidateLineRefs validates multiple line references, returning MismatchError with
// context showing updated hashes for changed lines.
func ValidateLineRefs(lines []string, refs []string) error {
	var mismatches []HashMismatch
	for _, ref := range refs {
		lr, err := ParseLineRef(ref)
		if err != nil {
			return err
		}
		if lr.Line < 1 || lr.Line > len(lines) {
			return errLineOutOfBounds(lr.Line, len(lines))
		}
		actual := ComputeLineHash(lr.Line, lines[lr.Line-1])
		if actual != lr.Hash {
			mismatches = append(mismatches, HashMismatch{
				Line: lr.Line, Expected: lr.Hash, Actual: actual,
			})
		}
	}
	if len(mismatches) > 0 {
		return &MismatchError{
			Mismatches: mismatches,
			FileLines:  lines,
		}
	}
	return nil
}

// HashMismatch is a single line hash mismatch.
type HashMismatch struct {
	Line     int
	Expected string
	Actual   string
}

// MismatchError provides context showing correct hashes for changed lines.
type MismatchError struct {
	Mismatches []HashMismatch
	FileLines  []string
}

func (e *MismatchError) Error() string {
	return formatMismatchMessage(e.Mismatches, e.FileLines)
}

const mismatchContext = 2

func formatMismatchMessage(mismatches []HashMismatch, fileLines []string) string {
	var b strings.Builder
	if len(mismatches) == 1 {
		b.WriteString("1 line has changed since last read. ")
	} else {
		fmt.Fprintf(&b, "%d lines have changed since last read. ", len(mismatches))
	}
	b.WriteString("Use updated references below (>>> marks changed lines).\n\n")

	displaySet := make(map[int]bool)
	for _, m := range mismatches {
		for line := max(1, m.Line-mismatchContext); line <= min(len(fileLines), m.Line+mismatchContext); line++ {
			displaySet[line] = true
		}
	}

	mismatchMap := make(map[int]*HashMismatch)
	for i := range mismatches {
		mismatchMap[mismatches[i].Line] = &mismatches[i]
	}

	prevLine := 0
	for line := 1; line <= len(fileLines); line++ {
		if !displaySet[line] {
			continue
		}
		if prevLine != 0 && line > prevLine+1 {
			b.WriteString("    ...\n")
		}
		prevLine = line

		content := fileLines[line-1]
		hash := ComputeLineHash(line, content)
		prefix := formatLineRef(line, hash, content)
		if _, ok := mismatchMap[line]; ok {
			b.WriteString(">>> ")
		} else {
			b.WriteString("    ")
		}
		b.WriteString(prefix)
		b.WriteString("\n")
	}

	return b.String()
}

func formatLineRef(lineNum int, hash, content string) string {
	return fmt.Sprintf("%d#%s|%s", lineNum, hash, content)
}

func tryParseRef(s string) (LineRef, bool) {
	i := strings.IndexByte(s, '#')
	if i <= 0 || i >= len(s)-1 {
		return LineRef{}, false
	}

	lineStr := s[:i]
	hashStr := s[i+1:]

	lineNum := 0
	for _, c := range lineStr {
		if c < '0' || c > '9' {
			return LineRef{}, false
		}
		lineNum = lineNum*10 + int(c-'0')
	}
	if lineNum == 0 {
		return LineRef{}, false
	}

	if len(hashStr) < HashLen {
		return LineRef{}, false
	}
	hashChars := hashStr[:HashLen]
	for _, c := range hashChars {
		valid := false
		for _, n := range NibbleStr {
			if c == n {
				valid = true
				break
			}
		}
		if !valid {
			return LineRef{}, false
		}
	}

	return LineRef{Line: lineNum, Hash: hashChars}, true
}

var lineRefExtractPattern = regexp.MustCompile(`([0-9]+#[ZPMQVRWSNKTXJBYH]{4})`)

func extractLineRef(s string) string {
	m := lineRefExtractPattern.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func stripLLMPrefix(s string) string {
	if after, ok := strings.CutPrefix(s, ">>>"); ok {
		s = after
		s = strings.TrimSpace(s)
	}
	if (strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-")) && len(s) > 1 && s[1] != '#' {
		s = s[1:]
		s = strings.TrimSpace(s)
	}
	return s
}

func normalizeHashSeparator(s string) string {
	return regexp.MustCompile(`\s*#\s*`).ReplaceAllString(s, "#")
}

func stripPipeSuffix(s string) string {
	if idx := strings.IndexByte(s, '|'); idx >= 0 {
		s = s[:idx]
	}
	return s
}

func errInvalidLineRef(ref string) error {
	return fmt.Errorf("invalid line reference format: %q. Expected format: \"{{line_number}}#{{hash_id}}\"", ref)
}

func errLineOutOfBounds(line, total int) error {
	return fmt.Errorf("line number %d out of bounds (file has %d lines)", line, total)
}
