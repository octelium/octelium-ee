From the console package directory, install Chromium and run the browser tests:

```sh
npx playwright install chromium
npm test
```

The tests start their own Vite server and mount `TimeAgo` with 100 timestamps and the Session overview with Cordium metadata under React StrictMode. They check reference links, navigation, partial references, and missing or invalid extensions. The Agent chat is mounted against an in-page fake agent (`fetch` and `EventSource`) to check the conversation list, search, composer, streaming, approvals, renaming and deletion. The navigation test mounts the page header, the command palette and the agent panel against the same kind of fake agent to check the breadcrumb menus, the page slots, the palette search and recents, and the console context sent with the questions. A running cluster is not required.

The Service playground tests mount the resource drawer for public LLM and MCP Services against a fake Service API and fake upstreams (`fetch`) to check the Playground tab, streaming chat completions, request settings, stopping, Policy errors, image attachments, the MCP handshake and session, schema-based tool arguments, resources, prompts and the request history, in both themes at widths from 320px to 1440px.

The Service connect tests mount the resource drawer against fake Service, agent and Cordium Workspace APIs whose terminals run a small simulated shell. They check the Connect tab per Service mode, the hidden setup terminal that installs the missing CLI (including failures and skipping it), the commands that are run or only typed for SSH, PostgreSQL, Kubernetes, embedded SSH and inferred TCP clients, keeping the session across the drawer tabs and removing it when the drawer closes, continuing in Cordium, starting the agent Workspace, asking the agent about the output, a selection or a failed connection, and the layout in both themes at desktop and phone widths.

The sidebar tests mount the console shell to check API and functional navigation, route selection, keyboard menus, saved preferences, the icon rail, mobile drawer focus and dismissal, both themes, and reduced motion at widths from 320px to 1440px.

The resource overview tests mount the resource drawer for Sessions, Services, Users, Credentials, Devices, Certificates, SecretStores, DirectoryProviders, access and Cordium resources against fake core, enterprise, access and Cordium APIs. They check that every overview fits the drawer in both themes at desktop and phone widths, the summary and key facts shown above the details, the order of the detail sections, the resource actions in the header and their shared style, the SecretStore synchronization and the confirmed Device binding reset requests, the Device posture checks, the Policy rules and the write-only Secret values.

The CollectorExporter tests mount the exporter form on its own and in the resource drawer against a fake enterprise API to check that the form fits the drawer in both themes and at phone width, the searchable destination picker, switching destinations and authentication methods without losing their settings, optional credentials, the Advanced options section, the transport security mapping, the Elasticsearch and Kafka specifics and saving from the drawer.
