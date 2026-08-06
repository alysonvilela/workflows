# 02 — Frontend shell

**What to build:** the frontend stops being the vanilla Vite starter and becomes a React application that talks to the API. Opening the browser shows the product layout and an indicator that the API is alive.

**Blocked by:** 01

**Status:** ready-for-agent

`frontend/` is currently the vanilla template (`src/main.ts`, `counter.ts`) with the `@coss` components dropped on top. React 19, the `@/*` alias in tsconfig and vite, and `jsx: react-jsx` were already added while prototyping — check before redoing them.

- [ ] React entry point replaces `src/main.ts`; the starter (counter, hero, Vite assets) is removed
- [ ] Client-side routing in place, with the product routes declared
- [ ] Base layout — header, content area — built from the existing `@coss` components, without modifying the primitives
- [ ] Hand-written HTTP client, with the base URL coming from `VITE_API_URL`
- [ ] `GET /health` state visible on screen, proving end to end that the frontend reaches the API
- [ ] Route skeletons while loading, not spinners
- [ ] `pnpm-workspace.yaml` at the monorepo root rather than inside `frontend/`
- [ ] All user-facing copy, identifiers and comments in English
