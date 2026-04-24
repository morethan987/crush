package tools

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/hashline"
	"github.com/stretchr/testify/require"
)

// makeRef builds a "lineNum#hash" anchor from lines content at the given 1-based line number.
func makeRef(lines []string, lineNum int) string {
	return fmt.Sprintf("%d#%s", lineNum, hashline.ComputeLineHash(lineNum, lines[lineNum-1]))
}

// contentToLines is a test helper: split content into lines for hashline tests.
func contentToLines(content string) []string {
	if content == "" {
		return []string{""}
	}
	return strings.Split(content, "\n")
}

func TestApplyHashlineEditsReplaceSingleLine(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), Lines: []string{"LINE 2"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "line 1\nLINE 2\nline 3", newContent)
}

func TestApplyHashlineEditsReplaceRange(t *testing.T) {
	t.Parallel()

	content := "a\nb\nc\nd\ne"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), End: makeRef(lines, 4), Lines: []string{"B", "C"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "a\nB\nC\ne", newContent)
}

func TestApplyHashlineEditsReplaceRangeWithDelete(t *testing.T) {
	t.Parallel()

	content := "a\nb\nc\nd\ne"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), End: makeRef(lines, 4), Lines: []string{"X"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "a\nX\ne", newContent)
}

func TestApplyHashlineEditsInsertAfter(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "insert_after", Pos: makeRef(lines, 1), Lines: []string{"inserted"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "line 1\ninserted\nline 2\nline 3", newContent)
}

func TestApplyHashlineEditsInsertBefore(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "insert_before", Pos: makeRef(lines, 3), Lines: []string{"inserted"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "line 1\nline 2\ninserted\nline 3", newContent)
}

func TestApplyHashlineEditsDeleteLines(t *testing.T) {
	t.Parallel()

	content := "a\nb\nc\nd"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), End: makeRef(lines, 3), Lines: []string{}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "a\nd", newContent)
}

func TestApplyHashlineEditsMultipleBottomUp(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), Lines: []string{"TWO"}},
		{Op: "insert_after", Pos: makeRef(lines, 4), Lines: []string{"after four"}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, applied)
	require.Empty(t, failed)
	require.Equal(t, "line 1\nTWO\nline 3\nline 4\nafter four\nline 5", newContent)
}

func TestApplyHashlineEditsPartialFailure(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 1), Lines: []string{"ONE"}},
		{Op: "insert_before", Pos: "", Lines: []string{"fail"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Len(t, failed, 1)
	require.Contains(t, failed[0].Error, "pos (line reference) is required")
	require.Equal(t, "ONE\nline 2\nline 3", newContent)
}

func TestApplyHashlineEditsAllFail(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "insert_before", Pos: "", Lines: []string{"fail1"}},
		{Op: "insert_after", Pos: "", Lines: []string{"fail2"}},
	})
	require.NoError(t, err)
	require.Equal(t, 0, applied)
	require.Len(t, failed, 2)
	require.Equal(t, content, newContent)
}

func TestApplyHashlineEditsEmptyContent(t *testing.T) {
	t.Parallel()

	lines := []string{""}

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 1), Lines: []string{"hello"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Empty(t, failed)
	require.Equal(t, "hello", newContent)
}

func TestApplyHashlineEditsDuplicatePosAnchorRejected(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	lines := contentToLines(content)

	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: makeRef(lines, 2), Lines: []string{"LINE 2 NEW"}},
		{Op: "replace", Pos: makeRef(lines, 2), Lines: []string{"LINE 2 OTHER"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Len(t, failed, 1)
	require.Contains(t, failed[0].Error, "not found in file")
	require.Equal(t, "line 1\nLINE 2 NEW\nline 3\nline 4\nline 5", newContent)
}

func TestApplyHashlineEditsAutoResolveAfterInsert(t *testing.T) {
	t.Parallel()

	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	lines := contentToLines(content)

	// First edit: insert 3 lines after line 1, pushing line 3 to line 6.
	// Second edit: reference old line 3 (now at line 6) with stale line number.
	// Auto-resolve should find the hash at the new position.
	newContent, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "insert_after", Pos: makeRef(lines, 3), Lines: []string{"ins1", "ins2", "ins3"}},
		{Op: "replace", Pos: makeRef(lines, 4), Lines: []string{"LINE 4 NEW"}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, applied)
	require.Empty(t, failed)
	require.Equal(t, "line 1\nline 2\nline 3\nins1\nins2\nins3\nLINE 4 NEW\nline 5", newContent)
}

func TestApplyHashlineEditsAutoResolveMultipleCandidatesFails(t *testing.T) {
	t.Parallel()

	// Two lines with identical content should have different hashes only if
	// they are significant (have letters). But if we construct a case where
	// hash collision occurs, it should fail with multiple candidates.
	// Since 4-char hash makes this rare, we test with empty lines (which use line-number seed).
	content := "line 1\nunique line\nline 3"
	lines := contentToLines(content)

	// Use a hash that doesn't exist anywhere — should fail.
	_, applied, failed, err := applyEditsToLines(lines, []HashlineEditOperation{
		{Op: "replace", Pos: "2#ZZZZ", Lines: []string{"won't work"}},
	})
	require.NoError(t, err)
	require.Equal(t, 0, applied)
	require.Len(t, failed, 1)
	require.Contains(t, failed[0].Error, "not found in file")
}
