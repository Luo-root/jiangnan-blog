import { readFileSync, writeFileSync } from "fs";
const p = 'D:/Code/Front-end/博客/src/routes/_layout.posts_.$slug.tsx';
let s = readFileSync(p, "utf8");
s = s.replace(/\nconsole\.log.*$/m, "");
writeFileSync(p, s);
console.log("cleaned, len =", s.length);
