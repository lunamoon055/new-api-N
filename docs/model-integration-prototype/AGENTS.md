# Prototype Instructions

Selected visual target: the first displayed Product Design concept, with a left model directory, editorial integration guide, and a dark right code panel. Deliver a standalone HTML export in addition to the Vite preview. Keep existing New API and QuantumNous identity.

User revision, 2026-10-11: remove the three-step quick-start section and its obsolete navigation links. Provide one-click copying of all current models' integration documentation, including authentication and code examples. Read the site's live model-square list only; never display models from a bundled snapshot or from the documentation mapping alone. Auto-refresh and clear outdated entries on failure. Export to `docs/model-integration.html` and `web/default/public/docs/model-integration.html`, mounted at the site's `/docs` entry.

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.
