package hashline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeLineHashDeterministic(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(42, "func hello() {")
	h2 := ComputeLineHash(42, "func hello() {")
	require.Equal(t, h1, h2)
	require.Len(t, h1, HashLen)
}

func TestComputeLineHashDifferentContent(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "func hello() {")
	h2 := ComputeLineHash(1, "func goodbye() {")
	require.NotEqual(t, h1, h2)
}

func TestComputeLineHashTrailingWhitespaceIgnored(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "content")
	h2 := ComputeLineHash(1, "content   ")
	require.Equal(t, h1, h2)
}

func TestComputeLineHashCRLFNormalized(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "content")
	h2 := ComputeLineHash(1, "content\r")
	require.Equal(t, h1, h2)
}

func TestComputeLineHashBlankLinesUseLineNumber(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "")
	h2 := ComputeLineHash(2, "")
	require.NotEqual(t, h1, h2, "blank lines should have different hashes based on line number")
}

func TestComputeLineHashSymbolOnlyLinesUseLineNumber(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "    ")
	h2 := ComputeLineHash(2, "    ")
	require.NotEqual(t, h1, h2, "whitespace-only lines should use line number as seed")
}

func TestComputeLineHashSignificantContentIgnoresLineNumber(t *testing.T) {
	t.Parallel()

	h1 := ComputeLineHash(1, "func hello() {")
	h2 := ComputeLineHash(100, "func hello() {")
	require.Equal(t, h1, h2, "lines with letters/digits should ignore line number")
}

func TestHashAlphabetChars(t *testing.T) {
	t.Parallel()

	for _, c := range NibbleStr {
		require.True(t, c >= 'A' && c <= 'Z', "expected uppercase letter, got %c", c)
	}
	require.Len(t, NibbleStr, 16)
}

func TestHashlineDictSize(t *testing.T) {
	t.Parallel()

	require.Len(t, hashlineDict, 65536)
	for i, h := range hashlineDict {
		require.Len(t, h, HashLen, "hashlineDict[%d] should be %d chars", i, HashLen)
	}
}

func TestFormatHashLine(t *testing.T) {
	t.Parallel()

	result := FormatHashLine(42, "func hello() {")
	require.Contains(t, result, "42#")
	require.Contains(t, result, "|func hello() {")

	hash := ComputeLineHash(42, "func hello() {")
	require.Contains(t, result, "42#"+hash+"|")
}

func TestFormatHashLines(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3"
	result := FormatHashLines(content, 1)

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 3)

	require.Contains(t, lines[0], "1#")
	require.Contains(t, lines[1], "2#")
	require.Contains(t, lines[2], "3#")
}

func TestFormatHashLinesEmpty(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", FormatHashLines("", 1))
}

func TestFormatHashLinesCustomStart(t *testing.T) {
	t.Parallel()

	content := "line a\nline b"
	result := FormatHashLines(content, 10)

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "10#")
	require.Contains(t, lines[1], "11#")
}

func TestParseLineRefValid(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("42#VKQR")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VKQR", lr.Hash)
}

func TestParseLineRefWithWhitespace(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("  42#VKQR  ")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VKQR", lr.Hash)
}

func TestParseLineRefWithPipe(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("42#VKQR|some content")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VKQR", lr.Hash)
}

func TestParseLineRefWithLLMPrefix(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef(">>> 42#VKQR")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VKQR", lr.Hash)
}

func TestParseLineRefInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid line reference format")
}

func TestParseLineRefZeroLine(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("0#VKQR")
	require.Error(t, err)
}

func TestParseLineRefInvalidHashChar(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("42#ABCD")
	require.Error(t, err, "A, B, C, D are not in the NibbleStr alphabet")
}

func TestParseLineRefExtracted(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("some prefix 42#VKQR and suffix")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VKQR", lr.Hash)
}

func TestValidateLineRefValid(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	hash := ComputeLineHash(2, "second")
	err := ValidateLineRef(lines, "2#"+hash)
	require.NoError(t, err)
}

func TestValidateLineRefOutOfBounds(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second"}
	err := ValidateLineRef(lines, "10#VKQR")
	require.Error(t, err)
	require.Contains(t, err.Error(), "out of bounds")
}

func TestValidateLineRefHashMismatch(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	err := ValidateLineRef(lines, "2#ZZZZ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "changed since last read")
}

func TestValidateLineRefsAllValid(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	h1 := ComputeLineHash(1, "first")
	h2 := ComputeLineHash(3, "third")
	err := ValidateLineRefs(lines, []string{"1#" + h1, "3#" + h2})
	require.NoError(t, err)
}

func TestValidateLineRefsMismatch(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	err := ValidateLineRefs(lines, []string{"1#ZZZZ", "3#ZZZZ"})
	require.Error(t, err)
	mismatchErr, ok := err.(*MismatchError)
	require.True(t, ok, "expected MismatchError")
	require.Len(t, mismatchErr.Mismatches, 2)
}

func TestMismatchErrorFormat(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	err := &MismatchError{
		Mismatches: []HashMismatch{{Line: 2, Expected: "ZZ", Actual: ComputeLineHash(2, "second")}},
		FileLines:  lines,
	}
	msg := err.Error()
	require.Contains(t, msg, "1 line has changed")
	require.Contains(t, msg, ">>>")
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	content := "package main\n\nfunc hello() {\n\treturn \"world\"\n}\n"
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		hash := ComputeLineHash(lineNum, line)
		ref := fmt.Sprintf("%d#%s", lineNum, hash)
		err := ValidateLineRef(lines, ref)
		require.NoError(t, err, "line %d (%q) hash validation failed", lineNum, line)
	}
}

func TestFormatHashLinesRoundTrip(t *testing.T) {
	t.Parallel()

	content := "package main\n\nfunc hello() {\n\treturn \"world\"\n}\n"
	formatted := FormatHashLines(content, 1)

	lines := strings.Split(content, "\n")
	for formattedLine := range strings.SplitSeq(formatted, "\n") {
		ref := formattedLine[:strings.Index(formattedLine, "|")]
		err := ValidateLineRef(lines, ref)
		require.NoError(t, err, "ref %q failed validation", ref)
	}
}

func TestResolveLineRefExactMatch(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	hash := ComputeLineHash(2, "second")
	resolved, err := ResolveLineRef(lines, "2#"+hash)
	require.NoError(t, err)
	require.Equal(t, 2, resolved.ResolvedLine)
	require.False(t, resolved.AutoResolved)
}

func TestResolveLineRefAutoResolveAfterShift(t *testing.T) {
	t.Parallel()

	// Original file: line 5 has "target"
	// After insertions, "target" is now at line 8
	// But the anchor still says 5#<hash>
	lines := []string{"line 1", "line 2", "line 3", "line 4", "target", "line 6"}
	_ = lines // original positions for reference
	originalHash := ComputeLineHash(5, "target")

	// Simulate shift: same content but "target" is at line 8
	shiftedLines := []string{"line 1", "inserted", "inserted", "inserted", "line 2", "line 3", "line 4", "target", "line 6"}
	resolved, err := ResolveLineRef(shiftedLines, "5#"+originalHash)
	require.NoError(t, err)
	require.Equal(t, 8, resolved.ResolvedLine)
	require.True(t, resolved.AutoResolved)
}

func TestResolveLineRefNotFound(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	_, err := ResolveLineRef(lines, "2#ZZZZ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found in file")
}
