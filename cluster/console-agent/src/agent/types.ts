import type {
  APIRisk,
  ApprovalPreview,
  Conversation,
  ErrorInfo,
  FileInfo,
  ModelInfo,
  RunActivity,
  Usage,
} from "../protocol/index.ts";
import type { MessageBuilder } from "../runs/builder.ts";
import type { ResourceCache } from "./resources.ts";

export interface ApprovalRequest {
  toolCallId: string;
  title: string;
  description?: string;
  risk: APIRisk;
  preview?: ApprovalPreview;
}

export interface ApprovalResult {
  approved: boolean;
  reason?: string;
}

export interface ToolRunContext {
  runId: string;
  conversationId: string;
  builder: MessageBuilder;
  resources: ResourceCache;
  resultsDir: string;
  requestApproval(
    req: ApprovalRequest,
    signal?: AbortSignal,
  ): Promise<ApprovalResult>;
}

export interface ToolContextProvider {
  current(conversationId: string): ToolRunContext | undefined;
}

export interface BackendRunInput {
  text: string;
  attachments: FileInfo[];
}

export interface BackendRunContext {
  runId: string;
  conversation: Conversation;
  builder: MessageBuilder;
  signal: AbortSignal;
  setActivity(activity: RunActivity | undefined): void;
}

export interface BackendRunResult {
  status: "completed" | "failed";
  error?: ErrorInfo;
  usage?: Usage;
}

export interface AgentBackend {
  readonly model?: ModelInfo;
  readonly thinkingLevel?: string;
  run(
    input: BackendRunInput,
    ctx: BackendRunContext,
  ): Promise<BackendRunResult>;
  generateTitle(
    text: string,
    signal?: AbortSignal,
  ): Promise<string | undefined>;
  forgetConversation(conversationId: string): void;
  dispose(): Promise<void>;
}
