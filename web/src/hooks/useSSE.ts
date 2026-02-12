import { useEffect, useRef } from "preact/hooks";

interface UseSSEOptions {
  onEvent?: (event: MessageEvent) => void;
  onError?: (error: Event) => void;
}

export function useSSE(url: string, options: UseSSEOptions) {
  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => {
    let source: EventSource | null = null;
    let retryTimeout: ReturnType<typeof setTimeout>;
    let retryDelay = 1000;

    function connect() {
      source = new EventSource(url);

      source.addEventListener("connected", () => {
        retryDelay = 1000; // Reset on successful connection
      });

      source.addEventListener("run_update", (e) => {
        optionsRef.current.onEvent?.(e);
      });

      source.addEventListener("step_update", (e) => {
        optionsRef.current.onEvent?.(e);
      });

      source.addEventListener("progress", (e) => {
        optionsRef.current.onEvent?.(e);
      });

      source.onerror = (e) => {
        optionsRef.current.onError?.(e);
        source?.close();
        // Reconnect with exponential backoff (max 30s)
        retryTimeout = setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 30000);
      };
    }

    connect();

    return () => {
      source?.close();
      clearTimeout(retryTimeout);
    };
  }, [url]);
}
