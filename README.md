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

## Hash-Edit 探索结论

[Hash-Edit 的设计初衷](https://blog.can.ac/2026/02/12/the-harness-problem/)是用短哈希锚点（如 `42#VKQR`）替代完整的 `old_string` 文本匹配，让模型无需逐字符复现原始代码即可精确指定编辑位置。

### 核心价值：消除"完美复现"失败模式

普通编辑要求模型**逐字符精确复现** old_string（包括空格、缩进、空行）。模型知道要改什么，但输出时复制不精确——这不是模型能力问题，而是编辑格式的结构性缺陷：

- Claude Code 的 "String to replace not found" 错误频繁到有自己的 GitHub megathread
- Aider 基准：仅换编辑格式就让 GPT-4 Turbo 从 26% → 59%
- JetBrains Diff-XYZ 基准：没有任何单一编辑格式在所有模型上占优
- **没有任何模型**在真实编辑任务上超过 60% pass@1

哈希编辑只需模型复现 2-4 个字符的锚点，而非整段文本。基准测试显示最弱的模型获益最大（Grok Code Fast 1: 6.7% → 68.3%，十倍提升）。

> "Often the model isn't flaky at understanding the task. It's flaky at expressing itself. You're blaming the pilot for the landing gear."

### 已实现的改进

1. **4 字符哈希**（65536 个值）：从 2 字符（256 值）升级，碰撞概率从 500 行文件 ~100% 降至 ~0.3%

2. **自动搜索**：当锚点的行号因之前的编辑而漂移时，工具自动在全文搜索匹配的哈希，模型无需重新 View 文件或计算行号偏移

3. **链式编辑稳定**：结合 4 字符哈希的低碰撞率和自动搜索，"看一次编辑多次"的工作流变得可靠

### 感想

哈希编辑技术从实验结果来看，对于弱模型提升最大，对于强模型收益不佳（token消耗与哈希值引入的模型心智负担）；所以最佳的选择是强模型使用普通编辑，弱模型使用哈希编辑
