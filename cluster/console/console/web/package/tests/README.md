From the console package directory, install Chromium and run the browser tests:

```sh
npx playwright install chromium
npm test
```

The tests start their own Vite server and mount `TimeAgo` with 100 timestamps and the Session overview with Cordium metadata under React StrictMode. They check reference links, navigation, partial references, and missing or invalid extensions. The Agent chat is mounted against an in-page fake agent (`fetch` and `EventSource`) to check the conversation list, search, composer, streaming, approvals, renaming and deletion. The navigation test mounts the page header, the command palette and the agent panel against the same kind of fake agent to check the breadcrumb menus, the page slots, the palette search and recents, and the console context sent with the questions. A running cluster is not required.

The Service playground tests mount the resource drawer for public LLM and MCP Services against a fake Service API and fake upstreams (`fetch`) to check the Playground tab, streaming chat completions, request settings, stopping, Policy errors, image attachments, the MCP handshake and session, schema-based tool arguments, resources, prompts and the request history, in both themes at widths from 320px to 1440px.

The sidebar tests mount the console shell to check API and functional navigation, route selection, keyboard menus, saved preferences, the icon rail, mobile drawer focus and dismissal, both themes, and reduced motion at widths from 320px to 1440px.
