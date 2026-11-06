import * as CoreP from "@/apis/corev1/corev1";
import { Callout } from "@/pages/enterprise/CollectorExporter/fields";
import { Button } from "@mantine/core";
import {
  Layers,
  Server,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import * as React from "react";
import { AccessForm } from "./edit/access";
import { ProtocolForm, UpstreamForm } from "./edit/ConfigForms";
import { DynamicForm } from "./edit/dynamic";
import {
  cloneForEdit,
  collectIssues,
  createConfig,
  ModeStash,
  SectionID,
  switchMode,
} from "./edit/model";
import ModePicker from "./edit/ModePicker";
import { getModeIcon, getModeInfo } from "./edit/modes";
import SectionTabs, { SectionTab } from "./edit/SectionTabs";
import { SettingsForm } from "./edit/settings";

const lastSections = new Map<string, SectionID>();

const Edit = (props: {
  item: CoreP.Service;
  onUpdate: (item: CoreP.Service) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(() => cloneForEdit(item));
  const stash = React.useRef<ModeStash>({});
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;
  const initialMode = React.useRef(req.spec?.mode);
  const [section, setSection] = React.useState<SectionID>(
    () => lastSections.get(itemKey) ?? "upstream",
  );

  React.useEffect(() => {
    setReq(cloneForEdit(item));
    stash.current = {};
    initialMode.current = item.spec?.mode;
  }, [itemKey]);

  const updateReq = () => {
    const next = CoreP.Service.clone(req);
    setReq(next);
    onUpdate(CoreP.Service.clone(next));
  };

  const spec = req.spec;
  if (!spec) return null;
  const info = getModeInfo(spec.mode);
  const config = spec.config;
  const issues = collectIssues(spec);
  const dynamicCount =
    (spec.dynamicConfig?.configs.length ?? 0) +
    (spec.dynamicConfig?.rules.length ?? 0);

  const tabs: SectionTab<SectionID>[] = [
    {
      id: "upstream",
      label: "Upstream",
      icon: Server,
      issues: issues.upstream,
    },
    ...(info?.tab
      ? [
          {
            id: "protocol" as const,
            label: info.tab,
            icon: getModeIcon(spec.mode),
            issues: issues.protocol,
          },
        ]
      : []),
    {
      id: "access",
      label: "Access",
      icon: ShieldCheck,
      count:
        (spec.authorization?.policies.length ?? 0) +
        (spec.authorization?.inlinePolicies.length ?? 0),
      issues: issues.access,
    },
    {
      id: "dynamic",
      label: "Dynamic",
      icon: Layers,
      count: dynamicCount,
      issues: issues.dynamic,
    },
    {
      id: "settings",
      label: "Settings",
      icon: Settings2,
      issues: issues.settings,
    },
  ];

  const active = tabs.some((tab) => tab.id === section) ? section : "upstream";

  const openSection = (id: SectionID) => {
    lastSections.set(itemKey, id);
    setSection(id);
  };

  const needsConfig = !config && (active === "upstream" || active === "protocol");

  return (
    <div className="@container flex min-w-0 flex-col gap-4">
      <ModePicker
        value={spec.mode}
        onChange={(mode) => {
          switchMode(spec, mode, stash.current);
          updateReq();
        }}
      />

      {!!item.metadata?.uid &&
        !!initialMode.current &&
        spec.mode !== initialMode.current && (
          <Callout tone="info" title="The mode changes when you save">
            The settings of the previous mode stay available here until then,
            so you can switch back.
          </Callout>
        )}

      {spec.isDisabled && (
        <Callout title="This Service is disabled">
          It keeps its configuration, but it does not accept requests.
        </Callout>
      )}

      {info ? (
        <>
          <SectionTabs tabs={tabs} value={active} onChange={openSection} />
          <div
            key={active}
            id={`service-panel-${active}`}
            role="tabpanel"
            aria-labelledby={`service-tab-${active}`}
            className="flex min-w-0 flex-col gap-4"
          >
            {needsConfig && (
              <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-slate-300 px-6 py-10 text-center">
                <p className="max-w-md text-body font-normal text-slate-600">
                  This Service has no main Configuration, so its upstream comes
                  from the dynamic rules.
                </p>
                <Button
                  type="button"
                  variant="default"
                  size="xs"
                  onClick={() => {
                    spec.config = createConfig(info);
                    updateReq();
                  }}
                >
                  Add a main Configuration
                </Button>
              </div>
            )}
            {active === "upstream" && config && (
              <UpstreamForm config={config} info={info} update={updateReq} />
            )}
            {active === "protocol" && config && (
              <ProtocolForm
                config={config}
                info={info}
                scope="default"
                update={updateReq}
              />
            )}
            {active === "access" && (
              <AccessForm spec={spec} info={info} update={updateReq} />
            )}
            {active === "dynamic" && (
              <DynamicForm spec={spec} info={info} update={updateReq} />
            )}
            {active === "settings" && (
              <SettingsForm spec={spec} info={info} update={updateReq} />
            )}
          </div>
        </>
      ) : (
        <div className="rounded-xl border border-dashed border-slate-300 px-6 py-10 text-center text-body font-normal text-slate-600">
          Choose a mode to see the settings that apply to it.
        </div>
      )}
    </div>
  );
};

export default Edit;
