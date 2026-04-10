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
	require.Len(t, h1, 2)
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

	require.Len(t, hashlineDict, 256)
	for i, h := range hashlineDict {
		require.Len(t, h, 2, "hashlineDict[%d] should be 2 chars", i)
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

	lr, err := ParseLineRef("42#VK")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VK", lr.Hash)
}

func TestParseLineRefWithWhitespace(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("  42#VK  ")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VK", lr.Hash)
}

func TestParseLineRefWithPipe(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("42#VK|some content")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VK", lr.Hash)
}

func TestParseLineRefWithLLMPrefix(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef(">>> 42#VK")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VK", lr.Hash)
}

func TestParseLineRefInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid line reference format")
}

func TestParseLineRefZeroLine(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("0#VK")
	require.Error(t, err)
}

func TestParseLineRefInvalidHashChar(t *testing.T) {
	t.Parallel()

	_, err := ParseLineRef("42#AB")
	require.Error(t, err, "A and B are not in the NibbleStr alphabet")
}

func TestParseLineRefExtracted(t *testing.T) {
	t.Parallel()

	lr, err := ParseLineRef("some prefix 42#VK and suffix")
	require.NoError(t, err)
	require.Equal(t, 42, lr.Line)
	require.Equal(t, "VK", lr.Hash)
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
	err := ValidateLineRef(lines, "10#VK")
	require.Error(t, err)
	require.Contains(t, err.Error(), "out of bounds")
}

func TestValidateLineRefHashMismatch(t *testing.T) {
	t.Parallel()

	lines := []string{"first", "second", "third"}
	err := ValidateLineRef(lines, "2#ZZ")
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
	err := ValidateLineRefs(lines, []string{"1#ZZ", "3#ZZ"})
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
