---
title: "Markdown 语法速查"
date: "2025-03-20"
tags: ["Markdown", "语法", "教程"]
cover: "https://picsum.photos/seed/markdown/1200/630"
excerpt: "一份覆盖标题、列表、代码、表格、任务、公式的 Markdown 语法参考。"
draft: false
---

## 标题层级

```markdown
# 一级标题
## 二级标题
### 三级标题
#### 四级标题
```

## 文本格式

- **粗体**：`**文字**`
- *斜体*：`*文字*`
- ~~删除线~~：`~~文字~~`
- `行内代码`：反引号包裹

## 列表

### 无序列表

- 第一项
- 第二项
  - 嵌套项
  - 嵌套项

### 有序列表

1. 第一步
2. 第二步
3. 第三步

### 任务列表

- [x] 已完成的任务
- [x] 学习基础语法
- [ ] 未完成的任务
- [ ] 进阶用法

## 代码块

支持多种语言的语法高亮：

```javascript
function fibonacci(n) {
  if (n <= 1) return n;
  return fibonacci(n - 1) + fibonacci(n - 2);
}
console.log(fibonacci(10)); // 55
```

```python
def quicksort(arr):
    if len(arr) <= 1:
        return arr
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    return quicksort(left) + middle + quicksort(right)
```

## 表格

| 语法 | 效果 | 说明 |
|------|------|------|
| `**粗**` | **粗** | 加粗文本 |
| `*斜*` | *斜* | 斜体文本 |
| `` `code` `` | `code` | 行内代码 |
| `~~del~~` | ~~del~~ | 删除线 |

## 引用

> 这是一段引用文本。
>
> 引用可以包含多个段落。

## 分隔线

上方内容

---

下方内容

## 链接与图片

普通链接：[Obsidian 官网](https://obsidian.md)

WikiLink 站内链接：[[Obsidian入门指南]] 和 [[代码高亮的艺术]] 都用到了 Markdown 语法。

图片：

![示例图片](https://picsum.photos/seed/demo-image/800/400)

## 数学公式

行内公式：质能方程 $E = mc^2$

块级公式：

$$\int_{-\infty}^{\infty} e^{-x^2} dx = \sqrt{\pi}$$

更多公式用法见 [[用数学公式美化笔记]]。

## 总结

这份速查覆盖了 90% 的日常写作需求。遇到更复杂的排版需求时，随时回来查阅。
