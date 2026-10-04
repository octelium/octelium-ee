# @octelium/console-agent

The AI agent behind the chat of the Octelium web console. It runs as an ordinary process inside a Cordium Workspace, uses the Workspace's own Octelium identity and exposes an HTTP/JSON + SSE API that is served as the Workspace's default Application.

```text
Web console (chat UI)
   │  HTTP/JSON + SSE (via the Cordium portal, authenticated as the Workspace owner)
   ▼
octelium-console-agent ───────────── Cordium Workspace ──────────────────────────────
   ├─ Pi (agent loop, sessions, compaction, retries, skills)
   ├─ tools
   │   ├─ octelium_api_search / octelium_api_describe / octelium_api_call
   │   │     └─ gRPC over OCTELIUM_AUTH_PROXY_SOCKET (h2c) ──► Octelium APIs
   │   ├─ present_table / present_chart / present_resources / publish_artifact
   │   ├─ read / bash / edit / write / grep / find / ls  (the Workspace is the agent's computer)
   │   └─ web_fetch / web_search
   └─ LLM ──► Octelium LLM Service (via `octelium connect`), or any Pi provider
```

## Design

- **Execution.** The agent is a Node.js process inside the Workspace. Everything that the agent does (shell commands, `octelium`/`octeliumctl`, `psql`, `curl`, `kubectl`, scripts) runs with the Workspace's network and filesystem, so the protected Services are reachable by name through the Workspace's `octelium connect`.
- **Identity.** The Octelium APIs are called through the supervisor's auth proxy socket (`OCTELIUM_AUTH_PROXY_SOCKET`), exactly like the `octelium` CLI does inside a Workspace. Every call is authorized server-side by the Cluster's Policies for the Workspace owner. The model never sees a credential.
- **API discovery.** The agent embeds a protobuf descriptor set of the Octelium APIs (`apis/octelium.binpb`, generated from the `.proto` files with their doc comments). It powers a BM25 search over ~350 methods, schema rendering with the field documentation, and dynamic invocation: the model sends proto3 JSON, the agent validates it against the descriptors (unknown fields and wrong types are rejected with precise errors), encodes it and calls the method over gRPC. A model can only call a method that exists in the catalog. A compact index of all the methods is part of the system prompt so that common calls need no search.
- **Approvals.** Methods are classified by risk (`read`, `write`, `destructive`, `sensitive`). Depending on `approvals.octeliumAPI`, non-read calls pause until the User approves them in the UI (and shell commands too if `approvals.bash` is set). Approvals are a UX safety net, not the security boundary: authorization is always enforced by the Cluster.
- **Model.** By default the model is served by an Octelium LLM Service (`llm.provider: "octelium"`), reached through `octelium connect`, so no provider secret is stored in the Workspace. Any provider supported by Pi (e.g. `anthropic`, `openai`, `google`, `openrouter`) or any compatible endpoint (`custom`) can be used instead.
- **Backend abstraction.** The HTTP protocol knows nothing about Pi: runs go through the `AgentBackend` interface (`src/agent/types.ts`), so other harnesses (e.g. Codex or Claude as delegated sub-agents) can be added without changing the protocol.
- **Persistence.** Conversations, finalized messages, Pi sessions, uploads and artifacts are stored in the data directory. An in-flight message is checkpointed and recovered as `interrupted` if the process dies. Restarting the agent resumes the conversations with their full context.

## Running inside a Cordium Workspace

The enterprise `AgentService` (`octelium.api.main.enterprise.v1.AgentService`) provisions the agent of every User on demand: `InitializeAgent` creates the User's personal `octelium.<user>` Cordium Space, its `default` Template and a primary Workspace that serves the agent as its default Application on port `8080`. The Template installs Node.js upon the first start of a Workspace if the image lacks it (`ON_CREATE`) and starts `npx @octelium/console-agent@<version> serve` on every start (`POST_START`). The Template is generated from the `spec.agent` section of the enterprise ClusterConfig:

| Field | Description |
| --- | --- |
| `isDisabled` | Disables the `AgentService` |
| `llm.service`, `llm.model` | The default LLM Service and model (i.e. `llm.service` and `llm.model` below) |
| `version` | The `@octelium/console-agent` version |
| `image`, `limit` | The Workspace image and compute resources |
| `config` | Any additional configuration (same structure as `config.json`), passed via `OCTELIUM_CONSOLE_AGENT_CONFIG_JSON` |

The Workspaces themselves are started, stopped, watched and deleted via the Cordium `MainService` (the web console does so from its `/agent` page). The agent can also run in any other Workspace whose image has Node.js 22.19+ and `git`, for example with such a Template:

```yaml
spec:
  runtime:
    envVars:
      - key: OCTELIUM_CONSOLE_AGENT_CONFIG_JSON
        value: '{"llm":{"service":"llm.default","model":"gpt-5.1"}}'
    tasks:
      - name: console-agent
        type: POST_START
        isBackground: true
        run: npx --yes @octelium/console-agent@0.1.0 serve
  applications:
    - name: agent
      displayName: Octelium Agent
      port: 8080
      isDefault: true
```

`npx … doctor` checks the configuration, the Octelium API access and the LLM access.

## Configuration

The configuration is read, in increasing precedence, from the defaults, a JSON file (`--config`, `OCTELIUM_CONSOLE_AGENT_CONFIG` or `<dataDir>/config.json`), the `OCTELIUM_CONSOLE_AGENT_CONFIG_JSON` environment variable (convenient for Workspace env vars and Secrets) and the CLI flags. Unknown fields are rejected. See `config.example.json`.

| Field | Default | Description |
| --- | --- | --- |
| `dataDir` | `~/.octelium-console-agent` | Conversations, sessions, uploads, artifacts, skills |
| `workDir` | `/workspace` if it exists, else `$HOME` | The agent's working directory |
| `logLevel` | `info` | `debug`, `info`, `warn`, `error` |
| `server.host`, `server.port` | `0.0.0.0`, `8080` | Listen address (`8080` is the portal's default Application port) |
| `server.allowedOrigins` | `https://console.octelium.<domain>`, `https://$CORDIUM_HOSTNAME` | Accepted `Origin` headers (plus the request's own origin) |
| `server.cors` | `false` | Emit CORS headers. Keep it off when the Cluster ingress already handles CORS |
| `server.authToken` | | Optional bearer token (or `?access_token=`) required for `/v1/*` |
| `server.maxUploadBytes` | 100 MiB | Upload limit |
| `octelium.mode` | `auto` | `proxy` (auth proxy socket), `direct` (`OCTELIUM_DOMAIN` + `OCTELIUM_ACCESS_TOKEN`, for development), `disabled` |
| `octelium.domain` | `OCTELIUM_DOMAIN` | Cluster domain |
| `octelium.timeoutSeconds` | `30` | Per-call deadline |
| `llm.provider` | `octelium` | `octelium`, `custom` or any Pi provider ID |
| `llm.service` | | The LLM Service (`name` or `name.namespace`). Its URL is resolved via the User API |
| `llm.baseUrl` | | Overrides the endpoint (required for `custom`) |
| `llm.api` | `openai-completions` | Pi API of the endpoint, e.g. `openai-responses`, `anthropic-messages`, `google-generative-ai` |
| `llm.model` | | Model ID. Discovered from the endpoint's model list if unset (`octelium`/`custom`) |
| `llm.apiKey`, `llm.apiKeyEnv` | | Provider credentials. Not needed with an Octelium LLM Service |
| `llm.thinkingLevel` | `medium` | `off` … `max` |
| `llm.contextWindow`, `llm.maxTokens`, `llm.reasoning`, `llm.input` | from Pi's catalog | Model metadata for unknown model IDs |
| `llm.loginProviders` | `anthropic`, `openai` | The Pi providers that the Users can sign in to with their own subscriptions (Claude Pro/Max, ChatGPT) |
| `agent.tools` | all | Pi built-in tools: `read`, `bash`, `edit`, `write`, `grep`, `find`, `ls` (`grep`/`find` are dropped if `rg`/`fd` are missing) |
| `agent.systemPromptAppend` | | Extra instructions |
| `agent.loadContextFiles` | `false` | Load `AGENTS.md`-style files from the working directory |
| `agent.maxConcurrentRuns` | `4` | Across conversations |
| `agent.sessionIdleTimeoutSeconds` | `1800` | Idle Pi sessions are disposed (and transparently reopened) |
| `agent.compaction`, `agent.maxRetries` | `true`, `3` | Context compaction and LLM retries |
| `agent.generateTitles` | `true` | Generate conversation titles with the LLM |
| `approvals.octeliumAPI` | `write` | `never`, `destructive` or `write` (writes, deletions and sensitive calls) |
| `approvals.bash` | `false` | Require an approval for every shell command |
| `skills.repositories` | `octelium/octelium-skills@main` | Agent Skills repositories, cloned/updated at startup (best effort) |
| `skills.paths` | | Local skill directories |
| `webSearch` | | `{"provider": "brave" \| "tavily" \| "searxng", "baseUrl", "apiKey" \| "apiKeyEnv"}`. `baseUrl` can point to an Octelium Service for secretless access |

### Models and subscriptions

The model can be switched at runtime (`PUT /v1/model`). The selection is saved in `<dataDir>/settings.json` and takes precedence over `llm.*` as long as its provider stays configured.

A User can also sign in with their own Claude or ChatGPT subscription from the web console. The OAuth flows run inside the agent (`/v1/auth/logins`): the console shows the provider's sign-in link and then sends back what the provider shows after signing in (the code, or the URL of the page it redirected to, which never needs to load since it points to the Workspace's `localhost`). The tokens are stored and refreshed by Pi in `<dataDir>/pi/auth.json` inside the Workspace and never leave it. Once signed in, the agent switches to the provider's default model.

## HTTP API

All the types are exported from `@octelium/console-agent/protocol` (`src/protocol/index.ts`), which has no runtime dependencies and is meant to be imported by the web console.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness |
| `GET` | `/v1/info` | Version, protocol version, model, identity, capabilities, issues |
| `GET` | `/v1/conversations` | List the conversations |
| `POST` | `/v1/conversations` | Create a conversation, optionally starting a run: `{"title"?, "input"?: {"text", "attachments"?}}` |
| `GET` | `/v1/conversations/{id}` | The conversation, its messages and the snapshot of its active run |
| `PATCH` | `/v1/conversations/{id}` | Rename: `{"title"}` |
| `DELETE` | `/v1/conversations/{id}` | Cancel its run and delete it |
| `POST` | `/v1/conversations/{id}/runs` | Start a run: `{"input": {"text", "attachments"?}}` (`409` if a run is active) |
| `GET` | `/v1/runs/{id}` | Run snapshot (run + in-progress assistant message) |
| `GET` | `/v1/runs/{id}/events` | SSE stream of the run's events |
| `POST` | `/v1/runs/{id}/cancel` | Cancel the run |
| `POST` | `/v1/runs/{id}/approvals/{approvalId}` | `{"decision": "approve" \| "reject", "reason"?}` |
| `POST` | `/v1/files?name=<file name>` | Upload a file (raw body). Returns a `FileInfo` whose `id` can be attached to a run |
| `GET` | `/v1/artifacts/{id}` | Artifact metadata |
| `GET` | `/v1/artifacts/{id}/content` | Download an artifact (`attachment`, sandboxed CSP; `?inline=1` for safe types) |
| `GET` | `/v1/models` | The current model, the thinking level and the available models |
| `PUT` | `/v1/model` | Switch the model: `{"provider", "id", "thinkingLevel"?}` |
| `GET` | `/v1/auth/providers` | The subscription providers and whether they are signed in |
| `DELETE` | `/v1/auth/providers/{id}` | Sign out |
| `POST` | `/v1/auth/logins` | Start signing in: `{"provider", "selectModel"?}`. Returns a `LoginSession` |
| `GET` | `/v1/auth/logins/{id}` | Poll a `LoginSession` (`events` such as `auth_url`, the pending `prompt`, `status`) |
| `POST` | `/v1/auth/logins/{id}/prompts/{promptId}` | Answer the pending prompt: `{"value"}` |
| `DELETE` | `/v1/auth/logins/{id}` | Cancel signing in |

Errors are `{"error": {"code", "message"}}`. `POST`/`PUT`/`PATCH` bodies must be `application/json`.

### Messages and blocks

A run appends one user message and one assistant message to the conversation. A message is an ordered list of typed blocks:

| Block | Content |
| --- | --- |
| `markdown` | GitHub-flavored Markdown (streamed). Resources are linked as `octelium://resource/<apiVersion>/<Kind>/<name>` (see `formatResourceURI`/`parseResourceURI`) |
| `thinking` | The model's reasoning (streamed), possibly `redacted` |
| `tool` | A tool call: `name`, human `title`, `status` (`pending`, `running`, `awaiting_approval`, `completed`, `failed`, `rejected`, `cancelled`), `input`, truncated `output` and structured `details` (`octelium_api` with the method, risk, request, response and the referenced resources; `command`; `file` with diffs) |
| `approval` | A pending/decided approval with its risk and a preview of the request or command |
| `table` | Columns (with types and units) and rows |
| `chart` | `line`, `area`, `bar`, `pie` or `scatter` with an x key, series and rows |
| `resources` | Resource references with their console URIs and, when known, a snapshot |
| `artifact` | A downloadable file published by the agent |
| `notice`, `error` | Informational notices (e.g. context compaction) and errors |

Clients must ignore unknown block types and unknown fields, which is how new block types are added without breaking older consoles. `PROTOCOL_VERSION` is bumped for incompatible changes only.

### Events

`GET /v1/runs/{id}/events` streams `AgentEvent`s as SSE (`id` is the event's `seq`, `event` its `type`). The events of a run are replayed from the start, or after `Last-Event-ID` / `?after=<seq>`, so a client can reconnect or attach to a run in progress after loading `GET /v1/conversations/{id}`. The stream ends after a terminal run event; a finished run with nothing left to replay returns `204`, which stops `EventSource` from reconnecting.

| Event | Payload |
| --- | --- |
| `run.started`, `run.updated`, `run.completed`, `run.failed`, `run.cancelled` | `run` (status, `activity` such as retrying/compacting, usage, error) |
| `message.created`, `message.completed` | `message` (the completed message is authoritative) |
| `block.created`, `block.updated` | `messageId`, `block` (full block) |
| `block.delta` | `messageId`, `blockId`, `text` to append to a `markdown`/`thinking` block |
| `conversation.updated` | `conversation` (e.g. its generated title) |

## Requirements on the Cluster side

- The Workspace owner must be authorized to access the LLM Service, which is reached through the Workspace's `octelium connect`.
- The web console calls the agent at the Workspace's hostname through the Cordium portal. Since that is a different origin than the console's, the portal Service allows exactly the console's origin (`https://console.octelium.<domain>`) with credentials. `allowClusterServices` is deliberately not used since it would let any Workspace Application in the Cluster read the agents of other Users.
- Do not share the agent's Application with other Users: the agent acts with the Workspace owner's identity.

## Development

```bash
npm ci
npm test              # node:test, no network needed (faux LLM + mock gRPC server)
npm run typecheck
npm run build         # dist/
npm run dev           # node src/cli.ts serve
node src/cli.ts api search "workspace snapshot"
node src/cli.ts api describe core.v1.MainService/CreateService
```

Outside a Workspace, set `OCTELIUM_DOMAIN` and `OCTELIUM_ACCESS_TOKEN` to use the direct mode.

`apis/octelium.binpb` is regenerated by `make gen-api` (or `make gen-api-console-agent` after `make cp-pb`).
