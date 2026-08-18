import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import tailwindcss from "@tailwindcss/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";
import tsConfigPaths from "vite-tsconfig-paths";
import { existsSync, createReadStream, readdirSync, readFileSync, statSync } from "fs";
import { resolve, join, extname, sep } from "path";

const SOURCE_LOCATION_PLUGIN_CANDIDATES = [
  process.env.MEOO_SOURCE_LOCATION_PLUGIN_PATH,
  "/app/sdk/lib/src/plugins/source-location-babel.js",
  resolve(process.cwd(), "node_modules/@ali/oneday-agent-sdk/lib/src/plugins/source-location-babel.js"),
].filter(Boolean) as string[];

const SOURCE_LOCATION_PLUGIN_PATH = SOURCE_LOCATION_PLUGIN_CANDIDATES.find((path) => existsSync(path));

// ---------------------------------------------------------------------------
// Obsidian Vault 直连：把 工作台（D:/Data/工作台）里的附件作为 /vault/* 静态资源服务
// - dev: 中间件直接读磁盘（不走 Vite 资源管线）
// - build: generateBundle 阶段把 Vault 图片 emit 到 dist/vault/
// ---------------------------------------------------------------------------
const VAULT_ROOT = process.env.VAULT_ROOT || "D:/Data/工作台";
const VAULT_IMAGE_EXTS = new Set([".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".avif", ".bmp", ".ico"]);

// 不作为公开博客栏目的一级目录（Workbase 是私有 Agent 工作基座目录）
const EXCLUDED_SECTIONS = new Set([".obsidian", ".trash", "Workbase"]);

const MIME_MAP: Record<string, string> = {
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".svg": "image/svg+xml",
  ".avif": "image/avif",
  ".bmp": "image/bmp",
  ".ico": "image/x-icon",
};

function isInsideVault(abs: string): boolean {
  const root = resolve(VAULT_ROOT).toLowerCase();
  return resolve(abs).toLowerCase().startsWith(root);
}

function walkVaultImages(): string[] {
  const out: string[] = [];
  const walk = (dir: string) => {
    let entries;
    try {
      entries = readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const abs = join(dir, entry.name);
      if (entry.isDirectory()) {
        if (EXCLUDED_SECTIONS.has(entry.name)) continue;
        walk(abs);
      } else if (VAULT_IMAGE_EXTS.has(extname(entry.name).toLowerCase())) {
        out.push(abs);
      }
    }
  };
  walk(VAULT_ROOT);
  return out;
}

/** 文件名 → 相对 Vault 根的路径（正斜杠），同名图片取浅层优先 */
function buildImageIndex(): Record<string, string> {
  const index: Record<string, string> = {};
  for (const abs of walkVaultImages()) {
    const rel = resolve(abs).slice(resolve(VAULT_ROOT).length).replace(/^[\\/]+/, "").split(sep).join("/");
    const name = rel.split("/").pop() || "";
    if (name && !(name in index)) index[name] = rel;
  }
  return index;
}

/**
 * 一级目录（栏目）→ (栏目内相对路径 → md 原文)。
 * - 排除 .obsidian / .trash
 * - 相对路径不含一级目录前缀（slug 不携带栏目名）
 */
function buildVaultTree(): Record<string, Record<string, string>> {
  const tree: Record<string, Record<string, string>> = {};
  const root = resolve(VAULT_ROOT);
  let entries;
  try {
    entries = readdirSync(VAULT_ROOT, { withFileTypes: true });
  } catch {
    return tree;
  }
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const section = entry.name;
    if (EXCLUDED_SECTIONS.has(section)) continue;
    const files: Record<string, string> = {};
    const walk = (dir: string) => {
      let sub;
      try {
        sub = readdirSync(dir, { withFileTypes: true });
      } catch {
        return;
      }
      for (const e of sub) {
        const abs = join(dir, e.name);
        if (e.isDirectory()) {
          if (EXCLUDED_SECTIONS.has(e.name)) continue;
          walk(abs);
        } else if (e.name.toLowerCase().endsWith(".md")) {
          const rel = resolve(abs).slice(root.length).replace(/^[\\/]+/, "").split(sep).join("/");
          files[rel.slice(section.length + 1)] = readFileSync(abs, "utf8");
        }
      }
    };
    walk(join(VAULT_ROOT, section));
    tree[section] = files;
  }
  return tree;
}

/** 虚拟模块：virtual:vault-index（图片索引）+ virtual:vault-tree（栏目树） */
function vaultIndexPlugin(): Plugin {
  return {
    name: "vault-index",
    resolveId(id) {
      if (id === "virtual:vault-index") return "\0vault-index";
      if (id === "virtual:vault-tree") return "\0vault-tree";
    },
    load(id) {
      if (id === "\0vault-index") {
        return `export default ${JSON.stringify(buildImageIndex())};`;
      }
      if (id === "\0vault-tree") {
        return `export default ${JSON.stringify(buildVaultTree())};`;
      }
    },
  };
}

function vaultAssetsPlugin(): Plugin {
  return {
    name: "vault-assets",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use("/vault", (req, res, next) => {
        const rawPath = (req.url || "").split("?")[0];
        let rel: string;
        try {
          rel = decodeURIComponent(rawPath.replace(/^\/+/, ""));
        } catch {
          return next();
        }
        if (!rel || !VAULT_IMAGE_EXTS.has(extname(rel).toLowerCase())) return next();
        const abs = resolve(VAULT_ROOT, rel);
        if (!isInsideVault(abs) || !existsSync(abs) || !statSync(abs).isFile()) {
          res.statusCode = 404;
          res.end("Not Found");
          return;
        }
        res.setHeader("Content-Type", MIME_MAP[extname(rel).toLowerCase()] || "application/octet-stream");
        createReadStream(abs).pipe(res);
      });
    },
  };
}

function vaultAssetsBuildPlugin(): Plugin {
  return {
    name: "vault-assets-build",
    apply: "build",
    generateBundle() {
      for (const abs of walkVaultImages()) {
        const rel = resolve(abs).slice(resolve(VAULT_ROOT).length).replace(/^[\\/]+/, "").split(sep).join("/");
        this.emitFile({ type: "asset", fileName: `vault/${rel}`, source: readFileSync(abs) });
      }
    },
  };
}

/**
 * React + Vite 构建配置
 *
 * 硬约束：
 * - dev server 必须监听 3015 + strictPort（沙箱只开放一个代理端口）
 * - outDir 'dist' / assetsDir 'assets' — 归一化产物目录
 */
export default defineConfig({
  plugins: [
    tailwindcss(),
    TanStackRouterVite(),
    viteReact({
      babel: {
        plugins: SOURCE_LOCATION_PLUGIN_PATH
          ? [[SOURCE_LOCATION_PLUGIN_PATH, { projectRoot: process.cwd() }]]
          : [],
      },
    }),
    tsConfigPaths(),
    vaultIndexPlugin(),
    vaultAssetsPlugin(),
    vaultAssetsBuildPlugin(),
  ],
  server: {
    host: "0.0.0.0",
    port: 3015,
    strictPort: true,
    allowedHosts: true,
    fs: {
      // 允许 Vite 读取项目外的 Obsidian Vault（import.meta.glob 直扫）
      allow: [resolve(process.cwd()), VAULT_ROOT],
    },
    // HMR 默认关闭：沙箱预览 iframe 下 HMR 的整页 reload 会放大任何 transform error
    // 如需热更，改为: hmr: { clientPort: 443, protocol: 'wss' }
    hmr: false,
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: "assets/[name]-[hash].js",
        chunkFileNames: "assets/[name]-[hash].js",
        assetFileNames: "assets/[name]-[hash][extname]",
      },
    },
  },
});
