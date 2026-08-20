import { useEffect, useState } from "react";
import { loadSession } from "./lib/auth";
import { LoginPage } from "./routes/login";
import { Shell } from "./routes/shell";

function pathOf() {
  return location.pathname.replace(/\/+$/, "") || "/";
}

export function App() {
  const [path, setPath] = useState(pathOf);
  useEffect(() => {
    const onPop = () => setPath(pathOf());
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const sess = loadSession();
  const onLogin = sess ? "/workspace/inbox" : "/login";
  if (path === "/" || path === "") {
    history.replaceState(null, "", onLogin);
    return sess ? <Shell path="/workspace/inbox" /> : <LoginPage />;
  }
  if (path === "/login") {
    if (sess) {
      history.replaceState(null, "", "/workspace/inbox");
      return <Shell path="/workspace/inbox" />;
    }
    return <LoginPage />;
  }
  if (!sess) {
    history.replaceState(null, "", "/login");
    return <LoginPage />;
  }
  return <Shell path={path} />;
}
