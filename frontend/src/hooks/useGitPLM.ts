import { useEffect, useState } from "react";
import { api } from "../lib/api";

/** Shared hook: fetches gitplm base_url once and provides a URL builder */
export function useGitPLM() {
  const [baseUrl, setBaseUrl] = useState<string>("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getGitPLMConfig()
      .then((cfg) => setBaseUrl(cfg.base_url || ""))
      .catch(() => setBaseUrl(""))
      .finally(() => setLoading(false));
  }, []);

  const configured = !!baseUrl;

  const buildUrl = (ipn: string) =>
    configured ? `${baseUrl}/parts/${ipn}` : null;

  return { baseUrl, configured, loading, buildUrl };
}
