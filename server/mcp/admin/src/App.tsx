import { useEffect, useState } from "react";
import { ToastProvider } from "./components/toast";
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
  let page;
  if (path === "/" || path === "") {
    history.replaceState(null, "", onLogin);
    page = sess ? <Shell path="/workspace/inbox" /> : <LoginPage />;
  } else if (path === "/login") {
    if (sess) {
      history.replaceState(null, "", "/workspace/inbox");
      page = <Shell path="/workspace/inbox" />;
    } else {
      page = <LoginPage />;
    }
  } else if (!sess) {
    history.replaceState(null, "", "/login");
    page = <LoginPage />;
  } else {
    page = <Shell path={path} />;
  }
  return <ToastProvider>{page}</ToastProvider>;
}
