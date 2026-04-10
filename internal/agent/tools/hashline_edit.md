Edits files using hash-based line references for precise, efficient code changes.

<prerequisites>
1. Use View tool to understand file contents and note the hash annotations (e.g., 42#VK|...)
2. Each line in View output has format: line_number#hash_id|content
3. Use the line_number#hash_id anchors to specify edit targets
</prerequisites>

<parameters>
1. file_path: Absolute path to file (required)
2. edits: Array of edit operations, each containing:
   - op: "replace" | "insert_after" | "insert_before" (required)
   - pos: Line reference anchor, e.g., "42#VK" (required)
   - end: End line reference for range replace, e.g., "44#NKT" (optional, only for replace)
   - lines: Array of replacement content lines (required)
</parameters>

<operations>

**replace**: Replace one line or a range of lines.
- Single line: set pos only. Replaces that line with lines[].
- Range: set pos and end. Replaces lines from pos to end (inclusive) with lines[].
- To delete lines: use replace with empty lines array.

**insert_after**: Insert new lines after the anchored line. Original line stays unchanged.

**insert_before**: Insert new lines before the anchored line. Original line stays unchanged.
</operations>

<advantages>
- No need to copy exact whitespace/indentation from original code
- Hash anchors uniquely identify lines regardless of duplicate content
- Much less token usage than string-matching edits
- Concurrent-safe: detects if file changed since last read
</advantages>

<critical_requirements>
1. You MUST View the file first to get current hash annotations
2. Hash anchors come from View output — do not guess them
3. If you get a hash mismatch error, the file has changed — View it again
4. Edits are applied bottom-up (higher line numbers first) so earlier edits don't shift later anchors
5. All line references in a single call must be from the same View of the file
</critical_requirements>

<examples>
✅ Replace a single line:
```json
{
  "file_path": "/src/main.go",
  "edits": [{
    "op": "replace",
    "pos": "42#VK",
    "lines": ["func hello(name string) {"]
  }]
}
```

✅ Replace a range of lines:
```json
{
  "file_path": "/src/main.go",
  "edits": [{
    "op": "replace",
    "pos": "42#VK",
    "end": "44#NKT",
    "lines": ["func hello(name string) {", "    return fmt.Sprintf(\"Hello, %s\", name)", "}"]
  }]
}
```

✅ Insert lines after a specific line:
```json
{
  "file_path": "/src/main.go",
  "edits": [{
    "op": "insert_after",
    "pos": "42#VK",
    "lines": ["    log.Println(\"entered hello\")"]
  }]
}
```

✅ Multiple edits in one call:
```json
{
  "file_path": "/src/main.go",
  "edits": [
    {"op": "replace", "pos": "10#QR", "lines": ["var version = \"2.0\""]},
    {"op": "insert_after", "pos": "5#ZW", "lines": ["import \"log\""]}
  ]
}
```

❌ Wrong: Guessing hash values instead of getting them from View:
```json
{"op": "replace", "pos": "42#AA", "lines": ["..."]}
// AA is made up — use the actual hash from View output
```
</examples>

<warnings>
- If hash mismatch: file was modified externally. View the file again for fresh hashes.
- If edit fails: check response for failed edits list, fix the anchors, and retry.
- Partial success: some edits may apply while others fail. Check the response.
</warnings>
