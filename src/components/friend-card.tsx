import { useState } from "react";
import { ExternalLink } from "lucide-react";
import type { Friend } from "@/lib/friends";

/** 国风墨色（和文章/项目卡共用） */
const INK_COVERS = [
  "from-[#3d5a73] to-[#1f3245]",
  "from-[#3c423c] to-[#1c211c]",
  "from-[#6e4a36] to-[#38241a]",
  "from-[#7d6428] to-[#42351a]",
  "from-[#503c52] to-[#2a1f2c]",
  "from-[#3f5d4e] to-[#1f3027]",
];

function FriendAvatar({ name, avatar }: { name: string; avatar: string }) {
  const [failed, setFailed] = useState(false);
  if (!avatar || failed) {
    const hash = name.charCodeAt(0) % INK_COVERS.length;
    return (
      <div className={`flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-full bg-gradient-to-br ${INK_COVERS[hash]}`}>
        <span className="font-serif text-xl font-bold text-white/90 drop-shadow">
          {name.charAt(0) || "·"}
        </span>
      </div>
    );
  }
  return (
    <img
      src={avatar}
      alt={name}
      loading="lazy"
      onError={() => setFailed(true)}
      className="h-12 w-12 shrink-0 rounded-full object-cover ring-1 ring-border/60"
    />
  );
}

/**
 * 友链卡片
 * 整卡 = 外链（target="_blank"），无需 stopPropagation
 */
export function FriendCard({ friend }: { friend: Friend }) {
  return (
    <a
      href={friend.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group flex items-start gap-4 rounded-2xl border border-border/40 bg-card/60 p-5 shadow-[0_6px_22px_-16px_rgb(43_124_166_/_0.22),0_1px_0_0_rgb(143_194_223_/_0.08)_inset] transition-all hover:border-primary/40 hover:shadow-[0_18px_40px_-22px_rgb(43_124_166_/_0.4),0_0_0_1px_rgb(43_124_166_/_0.2)] hover:-translate-y-1"
    >
      <FriendAvatar name={friend.name} avatar={friend.avatar} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <h3 className="truncate font-serif text-base font-bold tracking-wide text-foreground transition-colors group-hover:text-primary">
            {friend.name}
          </h3>
          <ExternalLink
            size={12}
            className="shrink-0 text-muted-foreground opacity-0 transition-all group-hover:translate-x-0.5 group-hover:opacity-100"
          />
        </div>
        {friend.desc && (
          <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{friend.desc}</p>
        )}
        <p className="mt-1 truncate font-mono text-[11px] text-ink-4">{friend.url}</p>
      </div>
    </a>
  );
}