import { createFileRoute } from "@tanstack/react-router";
import { BookOpen, Link2, Code2, Sigma, Share2 } from "lucide-react";

export const Route = createFileRoute("/_layout/about")({
  component: About,
});

const FEATURES = [
  { icon: BookOpen, title: "Markdown 原生", desc: "支持 GFM 扩展语法：表格、任务列表、删除线、自动链接" },
  { icon: Link2, title: "双链转换", desc: "Obsidian 的 [[WikiLink]] 自动转为站内可点击链接" },
  { icon: Code2, title: "代码高亮", desc: "支持 180+ 编程语言语法高亮，一键复制代码" },
  { icon: Sigma, title: "数学公式", desc: "KaTeX 渲染行内 $E=mc^2$ 与块级公式" },
  { icon: Share2, title: "知识图谱", desc: "力导向图可视化文章间的引用关系，可交互探索" },
];

function About() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <div className="reveal">
        <h1 className="font-serif text-3xl font-bold">关于</h1>
        <p className="mt-4 leading-relaxed text-muted-foreground">
          静识是一个由 Obsidian 驱动的个人博客系统。它将 Obsidian 笔记的 Markdown 原生体验、双向链接与知识图谱完整保留到公开博客中，让个人知识网络可以被世界看见。
        </p>
      </div>

      <div className="reveal mt-12" data-reveal-delay="100">
        <h2 className="font-serif text-xl font-bold">核心特性</h2>
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          {FEATURES.map((f) => (
            <div key={f.title} className="rounded-xl border border-border bg-card p-5">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <f.icon size={18} />
              </div>
              <h3 className="mt-3 font-medium">{f.title}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>

      <div className="reveal mt-12" data-reveal-delay="200">
        <h2 className="font-serif text-xl font-bold">内容替换</h2>
        <p className="mt-4 leading-relaxed text-muted-foreground">
          本博客当前展示的是示例文章。要替换为你自己的 Obsidian 笔记，只需将 Markdown 文件放入
          <code className="mx-1 rounded bg-secondary px-1.5 py-0.5 text-sm">src/content/posts/</code>
          目录，并在文件头部添加 YAML frontmatter（标题、日期、标签、封面图、摘要），系统会自动解析并生成文章页面、标签索引与知识图谱。
        </p>
        <pre className="mt-4 overflow-x-auto rounded-lg border border-border bg-card p-4 text-sm">
          <code>{`---
title: "文章标题"
date: "2025-01-01"
tags: ["标签1", "标签2"]
cover: "https://example.com/cover.jpg"
excerpt: "一句话摘要"
draft: false
---

正文内容，支持 [[双链]] 和 $E=mc^2$。`}</code>
        </pre>
      </div>

      <div className="reveal mt-12" data-reveal-delay="300">
        <h2 className="font-serif text-xl font-bold">技术说明</h2>
        <p className="mt-4 leading-relaxed text-muted-foreground">
          本系统基于 React + Vite 构建，使用 react-markdown 渲染管线（remark-gfm / remark-math / rehype-katex / rehype-highlight），Fuse.js 提供全文模糊搜索，Canvas 实现力导向知识图谱。所有内容在构建时通过 import.meta.glob 加载，生成可部署的静态站点。
        </p>
      </div>
    </div>
  );
}
