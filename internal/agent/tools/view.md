Read a file by path with hash-annotated line numbers; supports offset and line limit (default 2000, max 100KB); renders images (PNG, JPEG, GIF, BMP, SVG, WebP); use ls for directories.

<usage>
- Provide file path to read
- Optional offset: start reading from specific line (0-based)
- Optional limit: control lines read (default 2000)
- Don't use for directories (use LS tool instead)
- Supports image files (PNG, JPEG, GIF, BMP, SVG, WebP)
</usage>

<features>
- Displays contents with hash-annotated line numbers (e.g., 42#VK|content)
- Each line has a unique 2-character hash for precise editing via hashline_edit tool
- Can read from any file position using offset
- Handles large files by limiting lines read
- Auto-truncates very long lines for display
- Suggests similar filenames when file not found
- Renders image files directly in terminal
</features>

<hashline_format>
Each output line has format: line_number#hash_id|content
Example: 42#VK|func hello() {
- line_number: 1-based line number
- hash_id: 2-character content hash (from alphabet ZPMQVRWSNKTXJBYH)
- Use the line_number#hash_id anchor in hashline_edit tool for precise edits
</hashline_format>

<limitations>
- Max file size: 100KB
- Default limit: 2000 lines
- Lines >2000 chars truncated
- Binary files (except images) cannot be displayed
</limitations>

<cross_platform>
- Handles Windows (CRLF) and Unix (LF) line endings
- Works with forward slashes (/) and backslashes (\)
- Auto-detects text encoding for common formats
</cross_platform>

<tips>
- Use with Glob to find files first
- For code exploration: Grep to find relevant files, then View to examine
- For large files: use offset parameter for specific sections
- View tool automatically detects and renders image files
- Copy line_number#hash_id anchors for use in hashline_edit operations
</tips>
