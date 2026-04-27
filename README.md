A customized fork of Crush, always up to date with crush statle.

new features:

- [Hash-Edit](https://blog.can.ac/2026/02/12/the-harness-problem/): cover the builtin edit and multiedit tool, to achieve a better edit performance. Since hash_edit covers two tools, you can disable them in config file:
```json
{
  "$schema": "https://charm.land/crush.json",
  "options": {
    "disabled_tools": ["edit", "multiedit"]
  }
}
```
