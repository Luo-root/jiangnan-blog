---
title: "代码高亮的艺术"
date: "2025-04-15"
tags: ["代码", "高亮", "编程"]
cover: "https://picsum.photos/seed/code-art/1200/630"
excerpt: "让代码块在笔记中既美观又可读——多语言高亮、复制按钮与排版技巧。"
draft: false
---

## 代码块是技术笔记的灵魂

一段排版良好的代码胜过千言万语。Obsidian 原生支持超过 180 种语言的语法高亮。Markdown 语法基础见 [[Markdown语法速查]]。

## 多语言示例

### TypeScript：类型系统

```typescript
interface TreeNode<T> {
  value: T;
  left: TreeNode<T> | null;
  right: TreeNode<T> | null;
}

function inorderTraversal<T>(root: TreeNode<T> | null): T[] {
  if (!root) return [];
  return [
    ...inorderTraversal(root.left),
    root.value,
    ...inorderTraversal(root.right),
  ];
}
```

### Rust：所有权与借用

```rust
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}

fn main() {
    let s1 = String::from("hello world");
    let s2 = String::from("rust");
    let result = longest(s1.as_str(), s2.as_str());
    println!("最长的是: {}", result);
}
```

### SQL：复杂查询

```sql
WITH ranked_posts AS (
  SELECT
    p.title,
    p.created_at,
    t.name AS tag,
    ROW_NUMBER() OVER (PARTITION BY p.id ORDER BY t.name) AS rn
  FROM posts p
  JOIN post_tags pt ON pt.post_id = p.id
  JOIN tags t ON t.id = pt.tag_id
  WHERE p.published = true
)
SELECT title, created_at, tag
FROM ranked_posts
WHERE rn = 1
ORDER BY created_at DESC
LIMIT 10;
```

### Go：并发协程

```go
func fanOut(input <-chan int, workers int) <-chan int {
    output := make(chan int)
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for n := range input {
                output <- n * n
            }
        }()
    }
    go func() { wg.Wait(); close(output) }()
    return output
}
```

### Bash：脚本技巧

```bash
#!/bin/bash
# 批量将 Obsidian 笔记同步到博客
VAULT="$HOME/Documents/Obsidian/Blog"
DEST="./src/content/posts"

find "$VAULT" -name "*.md" | while read -r file; do
  filename=$(basename "$file")
  cp "$file" "$DEST/$filename"
  echo "已同步: $filename"
done
```

## 行内代码

除了代码块，行内代码 `like this` 用于标注文件名、命令、变量名。比如配置文件 `vite.config.ts` 或命令 `pnpm run dev`。

## 代码与公式结合

技术笔记经常需要代码和公式配合使用。比如实现一个 softmax 函数：

```python
import numpy as np

def softmax(x):
    exp_x = np.exp(x - np.max(x))
    return exp_x / exp_x.sum()

logits = np.array([2.0, 1.0, 0.1])
print(softmax(logits))  # [0.659, 0.242, 0.099]
```

对应的数学定义：

$$\text{softmax}(x_i) = \frac{e^{x_i}}{\sum_{j=1}^{n} e^{x_j}}$$

## 总结

好的代码高亮不只是颜色，而是**信息层次**——关键字、字符串、注释各有视觉权重。配合 [[用数学公式美化笔记]] 中的公式，你的技术笔记将达到专业文档的水准。
