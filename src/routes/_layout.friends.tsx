import { createFileRoute } from "@tanstack/react-router";
import { Github, Mail } from "lucide-react";
import { getAllFriends } from "@/lib/friends";
import { FriendCard } from "@/components/friend-card";
import { ThemeScenery } from "@/components/theme-scenery";
import { DawnClouds, NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/friends")({
  component: FriendsList,
});

function FriendsList() {
  const friends = getAllFriends();

  return (
    <div className="friends-page relative mx-auto max-w-6xl overflow-hidden px-4 py-12 sm:px-6">
      <ThemeScenery className="posts-scenery absolute -inset-x-16 top-0 h-72 opacity-30" />

      {/* 页头 */}
      <div className="reveal relative">
        <div className="dawn-copy flex items-center gap-5">
          <img src="/assets/theme/sun.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-wide">东篱友舍</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">DAWN COMPANIONS · 共 {friends.length} 邻</p>
          </div>
        </div>
        <div className="night-copy flex items-center gap-5">
          <img src="/assets/theme/moon.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-[0.08em]">月下客来</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">NIGHT COMPANIONS · {friends.length} FRIENDS</p>
          </div>
        </div>
        <div className="dawn-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <div className="night-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-night-moon/35 to-transparent" />
        <DawnClouds className="dawn-only mt-3 h-10 w-full max-w-xl text-ink-4/60" />
        <NightWaves className="night-only mt-3 h-9 w-full max-w-xl text-ink-4/60" />
        <p className="mt-6 max-w-2xl text-sm leading-relaxed text-muted-foreground">
          收藏那些读起来像围炉夜话的站点，按名字排序。点击即往他处。
        </p>
      </div>

      {/* 友链卡片墙 */}
      <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {friends.map((friend, i) => (
          <div key={friend.slug} className="reveal" data-reveal-delay={`${Math.min(i * 60, 400)}`}>
            <FriendCard friend={friend} />
          </div>
        ))}
      </div>

      {/* 分隔 */}
      <div className="reveal mt-16 flex items-center gap-4">
        <span className="hairline flex-1" />
        <span className="font-calligraphy text-lg text-ink-3">欢迎交换友链</span>
        <span className="hairline flex-1" />
      </div>

      {/* 本站信息（方便对方直接复制添加） */}
      <section className="reveal mt-8" data-reveal-delay="40">
        <div className="rounded-sm border border-border/60 bg-card/60 p-5">
          <div className="mb-3 flex items-center gap-2">
            <span className="font-serif font-bold tracking-wide">本站信息</span>
            <span className="font-mono text-[10px] text-ink-4">// 复制以下内容到你的友链页</span>
          </div>
          <dl className="grid gap-2 font-mono text-xs sm:grid-cols-[auto_1fr] sm:gap-x-4 sm:gap-y-1.5">
            <dt className="text-ink-4">名称</dt>
            <dd className="text-foreground">遇见江楠</dd>
            <dt className="text-ink-4">地址</dt>
            <dd className="text-foreground">https://jiangnan.dev</dd>
            <dt className="text-ink-4">简介</dt>
            <dd className="text-foreground">朝曦入山，夜隐卷序 — 一座关于代码、写作与山水的小筑。</dd>
            <dt className="text-ink-4">头像</dt>
            <dd className="break-all text-foreground">https://jiangnan.dev/assets/theme/logo.jpg</dd>
          </dl>
        </div>
      </section>

      {/* 提交方式 */}
      <section className="reveal mt-6" data-reveal-delay="80">
        <p className="mb-5 max-w-2xl text-sm leading-relaxed text-muted-foreground">
          想结邻而居？任选一种方式，附上你的站名、地址、一句话简介即可。
        </p>
        <div className="grid gap-3 sm:grid-cols-2">
          <a
            href="https://github.com/Luo-root/jiangnan-blog/issues/new"
            target="_blank"
            rel="noopener noreferrer"
            className="group flex items-center gap-4 rounded-sm border border-border/60 bg-card/60 p-5 transition-all hover:border-primary/40 hover:shadow-md hover:-translate-y-0.5"
          >
            <Github size={22} className="shrink-0 text-foreground transition-colors group-hover:text-primary" />
            <div className="min-w-0">
              <div className="font-serif font-bold tracking-wide group-hover:text-primary">GitHub Issue</div>
              <p className="mt-1 truncate font-mono text-[11px] text-ink-4">Luo-root / 提交新 Issue「友链申请：你的站名」</p>
            </div>
          </a>
          <a
            href="mailto:3029295957@qq.com?subject=%E5%8F%8B%E9%93%BE%E7%94%B3%E8%AF%B7"
            className="group flex items-center gap-4 rounded-sm border border-border/60 bg-card/60 p-5 transition-all hover:border-primary/40 hover:shadow-md hover:-translate-y-0.5"
          >
            <Mail size={22} className="shrink-0 text-foreground transition-colors group-hover:text-primary" />
            <div className="min-w-0">
              <div className="font-serif font-bold tracking-wide group-hover:text-primary">邮件</div>
              <p className="mt-1 truncate font-mono text-[11px] text-ink-4">3029295957@qq.com · 主题「友链申请」</p>
            </div>
          </a>
        </div>
      </section>
    </div>
  );
}