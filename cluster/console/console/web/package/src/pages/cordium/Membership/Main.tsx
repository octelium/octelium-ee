import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { twMerge } from "@/lib/utils";
import {
  getRoleMeta,
  gitProviderRefOf,
  spaceRefOf,
  userRefOf,
  workspaceRefOf,
} from "../utils";
import { BadgeCheck, Boxes, GitBranch, UserRound } from "lucide-react";

export const MainInfo = (props: {
  item: CordiumP.Membership;
}): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const meta = getRoleMeta(item.spec?.role);
  const userRef = userRefOf(status?.userRef);
  const spaceRef = spaceRefOf(status?.spaceRef);
  const gitProviderStates = Object.entries(status?.gitProviderStateMap ?? {});

  return {
    status: { label: meta.label, tone: meta.tone },
    groupOrder: [{ title: "Git providers", icon: GitBranch }],
    items: [
      {
        label: "Role",
        primary: true,
        icon: BadgeCheck,
        value: (
          <span className={twMerge("font-semibold", meta.className)}>
            {meta.label}
          </span>
        ),
      },
      ...(userRef
        ? [
            {
              label: "User",
              primary: true,
              icon: UserRound,
              value: <ResourceListLabel itemRef={userRef} />,
            },
          ]
        : []),
      ...(spaceRef
        ? [
            {
              label: "Space",
              primary: true,
              icon: Boxes,
              value: <ResourceListLabel itemRef={spaceRef} />,
            },
          ]
        : []),
      ...(status?.userInfo?.displayName
        ? [
            {
              label: "Display name",
              primary: true,
              icon: UserRound,
              value: (
                <span className="font-semibold text-slate-700">
                  {status.userInfo.displayName}
                </span>
              ),
            },
          ]
        : []),
      ...gitProviderStates.map(([key, state]) => {
        const gitProviderRef = gitProviderRefOf(state.gitProviderRef);
        const workspaceRef = workspaceRefOf(state.workspaceRef);

        return {
          label: key,
          group: "Git providers",
          span: "full" as const,
          value: (
            <div className="flex flex-wrap items-center gap-1.5">
              {gitProviderRef && <ResourceListLabel itemRef={gitProviderRef} />}
              {workspaceRef && <ResourceListLabel itemRef={workspaceRef} />}
              {state.createdAt && (
                <Label size="sm" outlined>
                  Authorized <TimeAgo rfc3339={state.createdAt} />
                </Label>
              )}
            </div>
          ),
        };
      }),
    ],
  };
};
