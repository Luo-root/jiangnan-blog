/// <reference types="vite/client" />

declare module "virtual:vault-index" {
  /** 文件名 → 相对 Obsidian Vault 根的路径（正斜杠） */
  const vaultIndex: Record<string, string>;
  export default vaultIndex;
}

declare module "virtual:vault-tree" {
  /** 一级目录（栏目）→ (栏目内相对路径 → md 原文) */
  const vaultTree: Record<string, Record<string, string>>;
  export default vaultTree;
}
