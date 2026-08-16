---
title: "用数学公式美化笔记"
date: "2025-04-02"
tags: ["数学", "KaTeX", "公式"]
cover: "https://picsum.photos/seed/math-notes/1200/630"
excerpt: "在 Obsidian 中用 LaTeX 语法书写优雅的数学公式，从行内到矩阵全覆盖。"
draft: false
---

## 为什么在笔记里写公式

数学公式是精确表达逻辑的工具。无论你写的是算法笔记、物理推导还是统计模型，LaTeX 公式都能让你的表达**无歧义**。

本文的 Markdown 基础语法见 [[Markdown语法速查]]。

## 行内公式

用 `$...$` 包裹行内公式。例如，勾股定理 $a^2 + b^2 = c^2$ 可以自然地嵌入段落中。

再比如欧拉公式 $e^{i\pi} + 1 = 0$，它把五个最重要的数学常数联系在了一起。

## 块级公式

用 `$$...$$` 包裹独占一行的公式：

$$\frac{\partial}{\partial t} \rho + \nabla \cdot (\rho \mathbf{v}) = 0$$

这是流体力学的**连续性方程**。

## 常用符号速查

### 分数与根号

$$\frac{a+b}{c-d}, \quad \sqrt{x^2 + y^2}, \quad \sqrt[n]{x}$$

### 求和与积分

$$\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$$

$$\int_0^{\pi} \sin(x) \, dx = 2$$

### 极限

$$\lim_{n \to \infty} \left(1 + \frac{1}{n}\right)^n = e$$

### 矩阵

$$A = \begin{pmatrix} 1 & 2 & 3 \\ 4 & 5 & 6 \\ 7 & 8 & 9 \end{pmatrix}$$

### 多行对齐

$$\begin{aligned} f(x) &= (x+1)^2 \\ &= x^2 + 2x + 1 \end{aligned}$$

## 实战：贝叶斯定理

在机器学习笔记中，贝叶斯定理是高频公式：

$$P(A|B) = \frac{P(B|A) \cdot P(A)}{P(B)}$$

展开后：

$$P(A|B) = \frac{P(B|A) \cdot P(A)}{P(B|A) \cdot P(A) + P(B|\neg A) \cdot P(\neg A)}$$

## 实战：梯度下降

深度学习中的梯度下降更新规则：

$$\theta_{t+1} = \theta_t - \eta \nabla_\theta J(\theta)$$

其中 $\eta$ 是学习率，$J(\theta)$ 是损失函数。动量优化变体：

$$v_t = \gamma v_{t-1} + \eta \nabla_\theta J(\theta)$$
$$\theta_{t+1} = \theta_t - v_t$$

## 条件概率链式法则

$$P(A_1, A_2, \ldots, A_n) = \prod_{i=1}^{n} P\left(A_i \,\middle|\, A_1, \ldots, A_{i-1}\right)$$

## 总结

数学公式让笔记从"描述性"升级为"精确性"。结合 [[代码高亮的艺术]] 中的代码块，你的技术笔记将兼具可读性与严谨性。
