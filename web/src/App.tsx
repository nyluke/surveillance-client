import { useState } from "react";
import Layout from "./components/Layout";
import LiveView from "./pages/LiveView";
import Settings from "./pages/Settings";
import Faces from "./pages/Faces";

export type Page = "live" | "faces" | "settings";

export default function App() {
  const [page, setPage] = useState<Page>("live");

  return (
    <Layout currentPage={page} onNavigate={setPage}>
      {page === "live" && <LiveView />}
      {page === "faces" && <Faces />}
      {page === "settings" && <Settings />}
    </Layout>
  );
}
