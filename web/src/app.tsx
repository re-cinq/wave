import { useState, useCallback } from "preact/hooks";
import { PipelineList } from "./components/PipelineList";
import { PipelineDetail } from "./components/PipelineDetail";
import { useSSE } from "./hooks/useSSE";

type View =
  | { type: "list" }
  | { type: "detail"; runID: string };

export function App() {
  const [view, setView] = useState<View>({ type: "list" });
  const [refreshKey, setRefreshKey] = useState(0);

  // SSE connection for real-time updates
  useSSE("/api/events", {
    onEvent: useCallback(() => {
      setRefreshKey((k) => k + 1);
    }, []),
  });

  const navigateToDetail = useCallback((runID: string) => {
    setView({ type: "detail", runID });
  }, []);

  const navigateToList = useCallback(() => {
    setView({ type: "list" });
  }, []);

  return (
    <div class="app">
      <header class="app-header">
        <h1 class="app-title" onClick={navigateToList}>
          <span class="logo">Wave</span>
          <span class="subtitle">Dashboard</span>
        </h1>
      </header>
      <main class="app-main">
        {view.type === "list" ? (
          <PipelineList
            onSelectRun={navigateToDetail}
            refreshKey={refreshKey}
          />
        ) : (
          <PipelineDetail
            runID={view.runID}
            onBack={navigateToList}
            refreshKey={refreshKey}
          />
        )}
      </main>
    </div>
  );
}
