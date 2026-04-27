# WebUI Capturer

Headless browser operator. Drives the chromedp adapter to capture full-page screenshots of WebUI routes cited in an audit issue.

## Role

This persona is metadata-only — the BrowserAdapter (chromedp) is the executor. Steps using this persona declare `adapter: browser` and pass a JSON array of `BrowserCommand` entries as the prompt source. The persona record exists so audit-issue's tool-scoping pre-flight has a name to validate against.

## Rules

- The prompt body MUST be a JSON array of `BrowserCommand` (action, url, selector, format, timeout_ms, wait_for).
- Capture at full viewport — let the parent step set the viewport size if needed via the Wave manifest's browser config.
- Save each screenshot under `.agents/output/screenshots/<slug>.png` where `<slug>` is a kebab-case derivation of the route path.
- Emit `webui-evidence.json` matching `wave://contracts/evidence` with `axis: "webui"` and one item per captured route.

## Constraints

- NEVER drive the browser to URLs outside the host listed in `issue_context.cited_routes`.
- NEVER write outside `.agents/output/`.
- If chromedp is unavailable in the current sandbox, write an `error` field into the evidence artifact and emit an empty `items[]` rather than failing the step — the parent pipeline will route the missing axis through the synthesize step's degradation path.
