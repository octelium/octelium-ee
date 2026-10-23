import type { Service } from "@/apis/corev1/corev1";
import type { Workspace } from "@/apis/cordiumv1/cordiumv1";
import type { PinnedContext } from "@/components/AgentDock/context";
import type { ConnectPlan, Facts } from "./plan";
import { mainCommand } from "./plan";

const MAX_TEXT = 5000;

const tail = (value: string, max: number) =>
  value.length > max ? `…${value.slice(value.length - max)}` : value;

export const terminalContext = (args: {
  service: Service;
  plan: ConnectPlan;
  facts: Facts;
  workspace: Workspace;
  terminalId?: string;
  output: string;
  selection?: string;
  issue?: string;
}): PinnedContext => {
  const name = args.service.metadata?.name ?? "";
  const selection = args.selection?.trim();
  return {
    label: selection
      ? `Terminal selection · ${name}`
      : `${args.plan.label} session · ${name}`,
    resources: [
      {
        apiVersion: "core/v1",
        kind: "Service",
        name,
        uid: args.service.metadata?.uid,
      },
    ],
    items: [
      {
        kind: "TerminalSession",
        id: args.terminalId,
        title: `${args.plan.label} session to ${name}`,
        data: {
          service: name,
          address: args.plan.address,
          tool: args.plan.tool,
          command: mainCommand(args.plan, args.facts),
          workspace: args.workspace.metadata?.name,
          terminal: args.terminalId,
          ...(args.facts.kubeconfig
            ? { kubeconfig: args.facts.kubeconfig }
            : {}),
          ...(args.issue ? { issue: args.issue } : {}),
          ...(selection
            ? { selection: tail(selection, MAX_TEXT) }
            : { output: tail(args.output, MAX_TEXT) }),
        },
      },
    ],
  };
};
