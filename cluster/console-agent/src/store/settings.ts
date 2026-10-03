import * as fs from "node:fs";
import * as path from "node:path";
import { writeFileAtomic } from "./conversations.ts";

export interface AgentSettings {
  model?: {
    provider: string;
    id: string;
  };
  thinkingLevel?: string;
}

const isObject = (arg: unknown): arg is Record<string, unknown> =>
  typeof arg === "object" && arg !== null && !Array.isArray(arg);

export class SettingsStore {
  private filePath: string;

  constructor(dataDir: string) {
    this.filePath = path.join(dataDir, "settings.json");
  }

  read(): AgentSettings {
    let parsed: unknown;
    try {
      parsed = JSON.parse(fs.readFileSync(this.filePath, "utf8"));
    } catch {
      return {};
    }
    if (!isObject(parsed)) {
      return {};
    }

    const ret: AgentSettings = {};
    const model = parsed.model;
    if (
      isObject(model) &&
      typeof model.provider === "string" &&
      typeof model.id === "string"
    ) {
      ret.model = { provider: model.provider, id: model.id };
    }
    if (typeof parsed.thinkingLevel === "string") {
      ret.thinkingLevel = parsed.thinkingLevel;
    }
    return ret;
  }

  write(settings: AgentSettings) {
    fs.mkdirSync(path.dirname(this.filePath), { recursive: true, mode: 0o700 });
    writeFileAtomic(this.filePath, JSON.stringify(settings, null, 2));
  }

  update(patch: Partial<AgentSettings>): AgentSettings {
    const ret = { ...this.read(), ...patch };
    for (const key of Object.keys(ret) as (keyof AgentSettings)[]) {
      if (ret[key] === undefined) {
        delete ret[key];
      }
    }
    this.write(ret);
    return ret;
  }
}
