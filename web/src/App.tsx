import { useState } from "react";
import Layout from "./components/Layout";
import LiveView from "./pages/LiveView";
import Settings from "./pages/Settings";

type Page = "live" | "settings";

export default function App() {
  const [page, setPage] = useState<Page>("live");

  return (
    <Layout currentPage={page} onNavigate={setPage}>
      {page === "live" ? <LiveView /> : <Settings />}
    </Layout>
  );
}
