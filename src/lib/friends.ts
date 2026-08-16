import { load as yamlLoad } from "js-yaml";
import vaultTree from "virtual:vault-tree";

const FRIENDS_SECTION = "友链";

export interface Friend {
  slug: string;
  name: string;
  url: string;
  avatar: string;
  desc: string;
}

function parseFrontmatter(raw: string): { meta: Record<string, unknown>; body: string } {
  const fmRegex = /^---\n([\s\S]*?)\n---\n?/;
  const match = raw.match(fmRegex);
  if (!match) return { meta: {}, body: raw };
  try {
    const meta = yamlLoad(match[1]) as Record<string, unknown>;
    return { meta, body: raw.slice(match[0].length) };
  } catch {
    return { meta: {}, body: raw.slice(match[0].length) };
  }
}

function toSlug(rel: string): string {
  return rel.split("/").join("__");
}

function buildFriends(): Friend[] {
  const section = vaultTree[FRIENDS_SECTION] ?? {};
  const friends: Friend[] = [];

  for (const [relPath, raw] of Object.entries(section)) {
    const rel = relPath.replace(/\.md$/, "");
    const { meta } = parseFrontmatter(raw);
    const url = (meta.url as string) || "";
    if (!url) continue;   // 没 url 的不收录
    friends.push({
      slug: toSlug(rel),
      name: (meta.name as string) || rel.split("/").pop() || rel,
      url,
      avatar: (meta.avatar as string) || "",
      desc: (meta.desc as string) || "",
    });
  }

  friends.sort((a, b) => a.name.localeCompare(b.name, "zh-Hans-CN"));
  return friends;
}

const allFriends = buildFriends();

export function getAllFriends(): Friend[] {
  return allFriends;
}