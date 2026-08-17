const fs = require("fs");
const p = 'D:/Code/Front-end/博客/src/routes/_layout.posts_.$slug.tsx';
let s = fs.readFileSync(p, "utf8");
s = s.replace(/\nconsole\.log.*$/m, "");
fs.writeFileSync(p, s);
console.log("cleaned, len =", s.length);
