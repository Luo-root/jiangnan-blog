---
title: "Obsidian 入门指南"
date: "2025-01-12"
tags: ["Obsidian", "入门", "笔记"]
cover: "https://picsum.photos/seed/obsidian-start/1200/630"
excerpt: "从零开始认识 Obsidian——本地优先、Markdown 原生、双链驱动的知识管理工具。"
draft: false
---

## 为什么选择 Obsidian

Obsidian 是一款**本地优先**的笔记工具，所有内容以纯 Markdown 文件存储在你的电脑上。这意味着你完全拥有自己的数据，不依赖任何云服务。

它最核心的能力是**双向链接**（Bidirectional Links），让你像构建个人维基百科一样组织知识。在 [[知识管理工作流]] 中，我们会深入探讨如何利用这一特性。

## 核心概念

### 1. Markdown 原生

Obsidian 使用标准 Markdown 语法，你可以参考 [[Markdown语法速查]] 快速上手。所有笔记都是 `.md` 文件，可以用任何文本编辑器打开。

### 2. 双向链接

用 `[[方括号]]` 即可创建指向其他笔记的链接。Obsidian 会自动建立反向引用，让你看到哪些笔记引用了当前笔记。这种思想源自 [[双链笔记的哲学]]。

### 3. 知识图谱

所有笔记和链接会构成一张可视化图谱，帮助你发现知识间的隐藏关联。详见 [[图谱思维与知识网络]]。

## 快速上手清单

- [x] 安装 Obsidian 并创建仓库（Vault）
- [x] 写下第一篇笔记
- [ ] 尝试用 `[[ ]]` 链接到另一篇笔记
- [ ] 打开图谱视图查看关联
- [ ] 安装一个社区插件

## 代码示例：配置文件

Obsidian 的仓库配置存储在 `.obsidian` 目录下：

```json
{
  "app": {
    "theme": "obsidian-git",
    "cssTheme": "Minimal"
  },
  "hotkeys": {
    "editor:toggle-bold": ["Mod+b"],
    "editor:insert-link": ["Mod+k"]
  }
}
```

## 总结

Obsidian 的魅力在于：它既是一个简单的 Markdown 编辑器，又是一个强大的知识网络引擎。下一篇 [[知识管理工作流]] 将展示如何把这些概念落地为日常习惯。
