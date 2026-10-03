import * as fs from "node:fs";
import type { AddressInfo } from "node:net";
import * as path from "node:path";
import type { Api, Model } from "@earendil-works/pi-ai";
import type { ModelRuntime } from "@earendil-works/pi-coding-agent";
import { PiBackend, type Identity } from "./agent/pi/backend.ts";
import type { ToolContextProvider } from "./agent/types.ts";
import type { Config } from "./config.ts";
import { logger as baseLogger, type Logger } from "./log.ts";
import { APICatalog } from "./octelium/catalog.ts";
import { OcteliumClient } from "./octelium/client.ts";
import { RunManager } from "./runs/manager.ts";
import { AgentServer } from "./server/server.ts";
import { syncSkills } from "./skills.ts";
import { ConversationStore } from "./store/conversations.ts";
import { FileStore } from "./store/files.ts";
import { getVersion } from "./version.ts";

export interface AppOptions {
  config: Config;
  logger?: Logger;
  catalog?: APICatalog;
  modelRuntime?: ModelRuntime;
  model?: Model<Api>;
  skipSkills?: boolean;
  env?: Record<string, string | undefined>;
}

export interface App {
  config: Config;
  catalog: APICatalog;
  octelium: OcteliumClient;
  store: ConversationStore;
  files: FileStore;
  runs: RunManager;
  backend: PiBackend;
  server: AgentServer;
  identity: Identity;
  start(): Promise<AddressInfo>;
  close(): Promise<void>;
}

interface GetStatusResponse {
  domain?: string;
  user?: {
    metadata?: { name?: string; uid?: string; displayName?: string };
  };
}

export const fetchIdentity = async (
  octelium: OcteliumClient,
  logger: Logger,
  env: Record<string, string | undefined> = process.env,
): Promise<Identity> => {
  const identity: Identity = {};
  if (env.CORDIUM_NAME || env.CORDIUM_HOSTNAME) {
    identity.workspace = {
      name: env.CORDIUM_NAME,
      hostname: env.CORDIUM_HOSTNAME,
    };
  }
  if (!octelium.enabled) {
    return identity;
  }
  try {
    const status = await octelium.call<GetStatusResponse>(
      "user.v1.MainService/GetStatus",
      {},
      { timeoutMs: 10000 },
    );
    const metadata = status.user?.metadata;
    if (metadata) {
      identity.user = {
        name: metadata.name,
        uid: metadata.uid,
        displayName: metadata.displayName,
      };
    }
  } catch (err) {
    logger.warn("Could not get the User's status", { error: err as Error });
  }
  return identity;
};

export const createApp = async (opts: AppOptions): Promise<App> => {
  const { config } = opts;
  const logger = opts.logger ?? baseLogger;
  const version = getVersion();

  fs.mkdirSync(config.dataDir, { recursive: true, mode: 0o700 });
  fs.mkdirSync(config.workDir, { recursive: true });

  const store = new ConversationStore(
    path.join(config.dataDir, "conversations"),
  );
  const recovered = store.init();
  if (recovered.length > 0) {
    logger.info("Recovered interrupted messages", { count: recovered.length });
  }

  const files = new FileStore(config.dataDir);
  files.init();

  const catalog = opts.catalog ?? APICatalog.load();
  const { client: octelium, reason } = OcteliumClient.fromConfig(
    catalog,
    config.octelium,
    `octelium-console-agent/${version}`,
  );
  const issues: string[] = [];
  if (reason) {
    issues.push(reason);
    logger.warn("The Octelium API is not available", { reason });
  } else {
    logger.info("Using the Octelium API", {
      mode: octelium.mode,
      domain: config.octelium.domain,
    });
  }

  const identity = await fetchIdentity(octelium, logger, opts.env);
  const skillPaths = opts.skipSkills ? [] : await syncSkills(config, logger);

  let runs: RunManager | undefined;
  const runtime: ToolContextProvider = {
    current: (conversationId) => runs?.current(conversationId),
  };

  const backend = new PiBackend({
    config,
    catalog,
    octelium,
    octeliumUnavailableReason: reason,
    runtime,
    store,
    files,
    logger,
    skillPaths,
    identity,
    modelRuntime: opts.modelRuntime,
    model: opts.model,
  });
  await backend.init();

  runs = new RunManager({
    store,
    files,
    backend,
    logger,
    maxConcurrentRuns: config.agent.maxConcurrentRuns,
    generateTitles: config.agent.generateTitles,
  });

  const server = new AgentServer({
    config,
    store,
    files,
    runs,
    backend,
    octelium,
    identity,
    issues,
    version,
    logger,
  });

  const runManager = runs;
  return {
    config,
    catalog,
    octelium,
    store,
    files,
    runs: runManager,
    backend,
    server,
    identity,
    start: () => server.listen(config.server.host, config.server.port),
    close: async () => {
      await runManager.shutdown();
      await backend.dispose();
      await server.close();
    },
  };
};
