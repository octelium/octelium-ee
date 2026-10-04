import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getLimitItems, getSpaceTypeLabel, userRefOf } from "../utils";

export const MainInfo = (props: { item: CordiumP.Space }): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec;
  const status = item.status;
  const userRef = userRefOf(status?.userRef);
  const name = item.metadata!.name;
  const sshDisabled = !!spec?.authorization?.disableSSH;

  const related = [
    { label: "Workspaces", path: "workspaces" },
    { label: "Templates", path: "templates" },
    { label: "Memberships", path: "memberships" },
    { label: "Git providers", path: "gitproviders" },
    { label: "Secrets", path: "secrets" },
  ];

  return {
    status: {
      label: getSpaceTypeLabel(status?.type),
      tone:
        status?.type === CordiumP.Space_Status_Type.ORGANIZATION
          ? "info"
          : "neutral",
    },
    groupOrder: ["Space details", "Limits", "Runtime", "Related resources"],
    items: [
      {
        label: "Type",
        primary: true,
        value: <Label>{getSpaceTypeLabel(status?.type)}</Label>,
      },
      {
        label: "SSH",
        primary: true,
        value: (
          <Label tone={sshDisabled ? "warning" : "success"}>
            {sshDisabled ? "Disabled" : "Enabled"}
          </Label>
        ),
        hint: "Whether the Space members can access its Workspaces over SSH.",
      },
      ...(userRef
        ? [
            {
              label: "Owner",
              value: <ResourceListLabel itemRef={userRef} />,
            },
          ]
        : []),

      ...getLimitItems(spec?.limit?.defaultLimit).map((limit) => ({
        label: `Default ${limit.label}`,
        group: "Limits",
        value: (
          <span className="font-semibold text-slate-700">{limit.value}</span>
        ),
      })),
      ...getLimitItems(spec?.limit?.maxLimit).map((limit) => ({
        label: `Max ${limit.label}`,
        group: "Limits",
        value: (
          <span className="font-semibold text-slate-700">{limit.value}</span>
        ),
      })),

      ...((spec?.runtime?.envVars.length ?? 0) > 0
        ? [
            {
              label: "Environment variables",
              group: "Runtime",
              value: (
                <div className="flex flex-wrap gap-1.5">
                  {spec!.runtime!.envVars.map((envVar) => (
                    <Label key={envVar.key} size="sm" outlined>
                      {envVar.key}
                    </Label>
                  ))}
                </div>
              ),
            },
          ]
        : []),
      ...((spec?.runtime?.tasks.length ?? 0) > 0
        ? [
            {
              label: "Lifecycle tasks",
              group: "Runtime",
              value: (
                <span className="font-semibold text-slate-700">
                  {spec!.runtime!.tasks.length}
                </span>
              ),
            },
          ]
        : []),

      ...related.map((entry) => ({
        label: entry.label,
        group: "Related resources",
        value: (
          <ResourceListLabel
            to={`/cordium/${entry.path}?spaceRef.name=${name}`}
          >
            View {entry.label.toLowerCase()}
          </ResourceListLabel>
        ),
      })),
    ],
  };
};
