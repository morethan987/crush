package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/diff"
	"github.com/charmbracelet/crush/internal/filepathext"
	"github.com/charmbracelet/crush/internal/filetracker"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/hashline"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/permission"
)

const HashlineEditToolName = "hashline_edit"

//go:embed hashline_edit.md
var hashlineEditDescription []byte

type HashlineEditOperation struct {
	Op    string   `json:"op" description:"Operation type: replace, insert_after, or insert_before"`
	Pos   string   `json:"pos" description:"Line reference anchor (e.g. 42#VK)"`
	End   string   `json:"end,omitempty" description:"End line reference for range replace (e.g. 44#NKT). Only for replace op."`
	Lines []string `json:"lines" description:"Replacement lines of content"`
}

type HashlineEditParams struct {
	FilePath string                  `json:"file_path" description:"The absolute path to the file to modify"`
	Edits    []HashlineEditOperation `json:"edits,omitempty" description:"Array of hashline edit operations to apply bottom-up"`
	Content  string                  `json:"content,omitempty" description:"Initial content for creating a new file. Only used when the file does not exist."`
}

type HashlineEditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type HashlineEditFailed struct {
	Index int                   `json:"index"`
	Error string                `json:"error"`
	Edit  HashlineEditOperation `json:"edit"`
}

type HashlineEditResponseMetadata struct {
	Additions    int                  `json:"additions"`
	Removals     int                  `json:"removals"`
	OldContent   string               `json:"old_content,omitempty"`
	NewContent   string               `json:"new_content,omitempty"`
	EditsApplied int                  `json:"edits_applied"`
	EditsFailed  []HashlineEditFailed `json:"edits_failed,omitempty"`
}

func NewHashlineEditTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		HashlineEditToolName,
		string(hashlineEditDescription),
		func(ctx context.Context, params HashlineEditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			if len(params.Edits) == 0 && params.Content == "" {
				return fantasy.NewTextErrorResponse("at least one edit operation or content is required"), nil
			}

			params.FilePath = filepathext.SmartJoin(workingDir, params.FilePath)

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for editing file")
			}

			fileInfo, err := os.Stat(params.FilePath)
			if err != nil {
				if os.IsNotExist(err) {
					if params.Content == "" {
						return fantasy.NewTextErrorResponse(fmt.Sprintf("file not found: %s", params.FilePath)), nil
					}
					return handleHashlineEditCreateFile(ctx, lspManager, permissions, files, filetracker, workingDir, sessionID, params, call)
				}
				return fantasy.ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
			}

			if fileInfo.IsDir() {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", params.FilePath)), nil
			}

			lastRead := filetracker.LastReadTime(ctx, sessionID, params.FilePath)
			if lastRead.IsZero() {
				return fantasy.NewTextErrorResponse("you must read the file before editing it. Use the View tool first"), nil
			}

			modTime := fileInfo.ModTime().Truncate(time.Second)
			if modTime.After(lastRead) {
				return fantasy.NewTextErrorResponse(
					fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
						params.FilePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
					)), nil
			}

			content, err := os.ReadFile(params.FilePath)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
			}

			oldContent, isCrlf := fsext.ToUnixLineEndings(string(content))

			newContent, editsApplied, failedEdits, applyErr := applyHashlineEdits(oldContent, params.Edits)
			if applyErr != nil {
				return fantasy.NewTextErrorResponse(applyErr.Error()), nil
			}

			if oldContent == newContent {
				if len(failedEdits) > 0 {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("no changes made - all %d edit(s) failed", len(failedEdits))), nil
				}
				return fantasy.NewTextErrorResponse("new content is the same as old content. No changes made."), nil
			}

			_, additions, removals := diff.GenerateDiff(
				oldContent,
				newContent,
				strings.TrimPrefix(params.FilePath, workingDir),
			)

			editsAppliedCount := editsApplied
			var description string
			if len(failedEdits) > 0 {
				description = fmt.Sprintf("Apply %d of %d hashline edits to file %s (%d failed)", editsAppliedCount, len(params.Edits), params.FilePath, len(failedEdits))
			} else {
				description = fmt.Sprintf("Apply %d hashline edits to file %s", editsAppliedCount, params.FilePath)
			}

			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        fsext.PathOrPrefix(params.FilePath, workingDir),
					ToolCallID:  call.ID,
					ToolName:    HashlineEditToolName,
					Action:      "write",
					Description: description,
					Params: HashlineEditPermissionsParams{
						FilePath:   params.FilePath,
						OldContent: oldContent,
						NewContent: newContent,
					},
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			writeContent := newContent
			if isCrlf {
				writeContent, _ = fsext.ToWindowsLineEndings(newContent)
			}

			err = os.WriteFile(params.FilePath, []byte(writeContent), 0o644)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
			}

			file, err := files.GetByPathAndSession(ctx, params.FilePath, sessionID)
			if err != nil {
				_, err = files.Create(ctx, sessionID, params.FilePath, oldContent)
				if err != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
				}
			}
			if file.Content != oldContent {
				_, err = files.CreateVersion(ctx, sessionID, params.FilePath, oldContent)
				if err != nil {
					slog.Error("Error creating file history version", "error", err)
				}
			}

			_, err = files.CreateVersion(ctx, sessionID, params.FilePath, newContent)
			if err != nil {
				slog.Error("Error creating file history version", "error", err)
			}

			filetracker.RecordRead(ctx, sessionID, params.FilePath)

			notifyLSPs(ctx, lspManager, params.FilePath)

			var message string
			if len(failedEdits) > 0 {
				message = fmt.Sprintf("Applied %d of %d hashline edits to file: %s (%d edit(s) failed)", editsAppliedCount, len(params.Edits), params.FilePath, len(failedEdits))
			} else {
				message = fmt.Sprintf("Applied %d hashline edits to file: %s", len(params.Edits), params.FilePath)
			}

			text := fmt.Sprintf("<result>\n%s\n</result>\n", message)
			text += getDiagnostics(params.FilePath, lspManager)

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(text),
				HashlineEditResponseMetadata{
					OldContent:   oldContent,
					NewContent:   newContent,
					Additions:    additions,
					Removals:     removals,
					EditsApplied: editsAppliedCount,
					EditsFailed:  failedEdits,
				},
			), nil
		})
}

// applyHashlineEdits applies hashline edit operations to content.
// Edits are applied bottom-up (sorted by line number descending) so earlier
// edits don't shift line numbers for later edits.
func applyHashlineEdits(content string, edits []HashlineEditOperation) (string, int, []HashlineEditFailed, error) {
	if content == "" {
		lines := []string{""}
		return applyEditsToLines(lines, edits)
	}

	lines := strings.Split(content, "\n")
	return applyEditsToLines(lines, edits)
}

func applyEditsToLines(lines []string, edits []HashlineEditOperation) (string, int, []HashlineEditFailed, error) {
	type indexedEdit struct {
		index int
		edit  HashlineEditOperation
		line  int
	}

	var validEdits []indexedEdit
	var failedEdits []HashlineEditFailed

	for i, edit := range edits {
		if edit.Op == "" {
			edit.Op = "replace"
		}

		if err := validateOp(edit); err != nil {
			failedEdits = append(failedEdits, HashlineEditFailed{
				Index: i + 1,
				Error: err.Error(),
				Edit:  edit,
			})
			continue
		}

		lr, err := hashline.ParseLineRef(edit.Pos)
		if err != nil {
			failedEdits = append(failedEdits, HashlineEditFailed{
				Index: i + 1,
				Error: err.Error(),
				Edit:  edit,
			})
			continue
		}

		validEdits = append(validEdits, indexedEdit{index: i, edit: edit, line: lr.Line})
	}

	if len(validEdits) == 0 {
		return strings.Join(lines, "\n"), 0, failedEdits, nil
	}

	allRefs := make([]string, 0, len(validEdits)*2)
	for _, ie := range validEdits {
		allRefs = append(allRefs, ie.edit.Pos)
		if ie.edit.End != "" {
			allRefs = append(allRefs, ie.edit.End)
		}
	}

	if err := hashline.ValidateLineRefs(lines, allRefs); err != nil {
		return "", 0, nil, err
	}

	sorted := make([]indexedEdit, len(validEdits))
	copy(sorted, validEdits)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].line > sorted[i].line {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	editsApplied := 0
	currentLines := make([]string, len(lines))
	copy(currentLines, lines)

	for _, ie := range sorted {
		result, err := applyOneEdit(currentLines, ie.edit)
		if err != nil {
			failedEdits = append(failedEdits, HashlineEditFailed{
				Index: ie.index + 1,
				Error: err.Error(),
				Edit:  ie.edit,
			})
			continue
		}
		currentLines = result
		editsApplied++
	}

	return strings.Join(currentLines, "\n"), editsApplied, failedEdits, nil
}

func validateOp(edit HashlineEditOperation) error {
	switch edit.Op {
	case "replace", "insert_after", "insert_before":
	default:
		return fmt.Errorf("invalid op %q: must be replace, insert_after, or insert_before", edit.Op)
	}
	if edit.Pos == "" {
		return fmt.Errorf("pos (line reference) is required")
	}
	return nil
}

func applyOneEdit(lines []string, edit HashlineEditOperation) ([]string, error) {
	switch edit.Op {
	case "replace":
		return applyReplace(lines, edit)
	case "insert_after":
		return applyInsertAfter(lines, edit)
	case "insert_before":
		return applyInsertBefore(lines, edit)
	default:
		return nil, fmt.Errorf("unknown op: %s", edit.Op)
	}
}

func validateEditHashes(lines []string, edit HashlineEditOperation) error {
	startRef, err := hashline.ParseLineRef(edit.Pos)
	if err != nil {
		return err
	}
	if startRef.Line < 1 || startRef.Line > len(lines) {
		return fmt.Errorf("line %d out of bounds (file has %d lines)", startRef.Line, len(lines))
	}
	actual := hashline.ComputeLineHash(startRef.Line, lines[startRef.Line-1])
	if actual != startRef.Hash {
		return fmt.Errorf("line %d has changed since last read (expected %s, got %s). Re-read the file to get updated hashes", startRef.Line, startRef.Hash, actual)
	}
	if edit.End != "" {
		endRef, err := hashline.ParseLineRef(edit.End)
		if err != nil {
			return err
		}
		if endRef.Line < 1 || endRef.Line > len(lines) {
			return fmt.Errorf("end line %d out of bounds (file has %d lines)", endRef.Line, len(lines))
		}
		endActual := hashline.ComputeLineHash(endRef.Line, lines[endRef.Line-1])
		if endActual != endRef.Hash {
			return fmt.Errorf("end line %d has changed since last read (expected %s, got %s). Re-read the file to get updated hashes", endRef.Line, endRef.Hash, endActual)
		}
	}
	return nil
}

func applyReplace(lines []string, edit HashlineEditOperation) ([]string, error) {
	if err := validateEditHashes(lines, edit); err != nil {
		return nil, err
	}

	startRef, err := hashline.ParseLineRef(edit.Pos)
	if err != nil {
		return nil, err
	}

	if edit.End != "" {
		endRef, err := hashline.ParseLineRef(edit.End)
		if err != nil {
			return nil, err
		}
		if startRef.Line > endRef.Line {
			return nil, fmt.Errorf("start line %d cannot be greater than end line %d", startRef.Line, endRef.Line)
		}
		replacement := edit.Lines
		newLines := make([]string, 0, len(lines)-((endRef.Line-startRef.Line)+1)+len(replacement))
		newLines = append(newLines, lines[:startRef.Line-1]...)
		newLines = append(newLines, replacement...)
		newLines = append(newLines, lines[endRef.Line:]...)
		return newLines, nil
	}

	replacement := edit.Lines
	if len(replacement) == 0 {
		replacement = []string{}
	}
	newLines := make([]string, 0, len(lines)-1+len(replacement))
	newLines = append(newLines, lines[:startRef.Line-1]...)
	newLines = append(newLines, replacement...)
	newLines = append(newLines, lines[startRef.Line:]...)
	return newLines, nil
}

func applyInsertAfter(lines []string, edit HashlineEditOperation) ([]string, error) {
	if err := validateEditHashes(lines, edit); err != nil {
		return nil, err
	}

	lr, err := hashline.ParseLineRef(edit.Pos)
	if err != nil {
		return nil, err
	}

	if len(edit.Lines) == 0 {
		return nil, fmt.Errorf("insert_after requires at least one line of content")
	}

	newLines := make([]string, 0, len(lines)+len(edit.Lines))
	newLines = append(newLines, lines[:lr.Line]...)
	newLines = append(newLines, edit.Lines...)
	newLines = append(newLines, lines[lr.Line:]...)
	return newLines, nil
}

func applyInsertBefore(lines []string, edit HashlineEditOperation) ([]string, error) {
	if err := validateEditHashes(lines, edit); err != nil {
		return nil, err
	}

	lr, err := hashline.ParseLineRef(edit.Pos)
	if err != nil {
		return nil, err
	}

	if len(edit.Lines) == 0 {
		return nil, fmt.Errorf("insert_before requires at least one line of content")
	}

	newLines := make([]string, 0, len(lines)+len(edit.Lines))
	newLines = append(newLines, lines[:lr.Line-1]...)
	newLines = append(newLines, edit.Lines...)
	newLines = append(newLines, lines[lr.Line-1:]...)
	return newLines, nil
}

func handleHashlineEditCreateFile(
	ctx context.Context,
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
	sessionID string,
	params HashlineEditParams,
	call fantasy.ToolCall,
) (fantasy.ToolResponse, error) {
	dir := filepath.Dir(params.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	content := params.Content

	_, additions, removals := diff.GenerateDiff(
		"",
		content,
		strings.TrimPrefix(params.FilePath, workingDir),
	)

	p, err := permissions.Request(ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        fsext.PathOrPrefix(params.FilePath, workingDir),
			ToolCallID:  call.ID,
			ToolName:    HashlineEditToolName,
			Action:      "write",
			Description: fmt.Sprintf("Create file %s", params.FilePath),
			Params: HashlineEditPermissionsParams{
				FilePath:   params.FilePath,
				OldContent: "",
				NewContent: content,
			},
		},
	)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(params.FilePath, []byte(content), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	_, err = files.Create(ctx, sessionID, params.FilePath, "")
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
	}

	_, err = files.CreateVersion(ctx, sessionID, params.FilePath, content)
	if err != nil {
		slog.Error("Error creating file history version", "error", err)
	}

	filetracker.RecordRead(ctx, sessionID, params.FilePath)

	notifyLSPs(ctx, lspManager, params.FilePath)

	message := fmt.Sprintf("File created: %s", params.FilePath)
	text := fmt.Sprintf("<result>\n%s\n</result>\n", message)
	text += getDiagnostics(params.FilePath, lspManager)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(text),
		HashlineEditResponseMetadata{
			OldContent:   "",
			NewContent:   content,
			Additions:    additions,
			Removals:     removals,
			EditsApplied: 0,
		},
	), nil
}
