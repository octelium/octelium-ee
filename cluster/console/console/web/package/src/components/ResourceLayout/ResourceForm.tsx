import ContainerGen from "@/components/ContainerGen";
import MetadataEdit from "@/components/MetadataEdit";
import ResourceEditor from "@/components/ResourceEditor";
import {
  consumeNavigationApproval,
  useRegisterDirtyForm,
  validateResourceName,
} from "@/utils/forms";
import {
  cloneResource,
  Resource,
  resourceFingerprint,
  resourceFromYAML,
  resourceToYAML,
} from "@/utils/pb";
import { Button, Modal, SegmentedControl } from "@mantine/core";
import { AlertTriangle, FileCode, Loader2, Settings, X } from "lucide-react";
import * as React from "react";
import { useBlocker } from "react-router-dom";

type SpecComponent = React.ComponentType<{
  item: Resource;
  onUpdate: (item: Resource) => void;
}>;

export interface ResourceFormProps {
  item: Resource;
  specComponent?: SpecComponent;
  dataComponent?: SpecComponent;
  noMetadata?: boolean;
  readOnly?: boolean;
  submitIcon: React.ReactNode;
  submitLabel: string;
  submitPendingLabel: string;
  isPending: boolean;
  isError: boolean;
  errorLabel: string;
  onSubmit: (item: Resource, yaml: string, isYAML: boolean) => void;
  onCancel: () => void;
  requireDirty?: boolean;
}

const YAML_PARSE_DELAY = 300;

const FormSkeleton = () => (
  <div className="flex animate-pulse flex-col gap-3">
    <div className="h-9 w-full rounded-lg bg-slate-100" />
    <div className="h-9 w-2/3 rounded-lg bg-slate-100" />
    <div className="h-24 w-full rounded-lg bg-slate-100" />
  </div>
);

const DiscardModal = (props: {
  opened: boolean;
  onStay: () => void;
  onDiscard: () => void;
}) => (
  <Modal
    opened={props.opened}
    onClose={props.onStay}
    centered
    withCloseButton={false}
    padding={0}
    size="sm"
  >
    <div className="flex flex-col">
      <div className="flex items-center gap-2.5 border-b border-slate-200 bg-slate-50/70 px-5 py-3.5">
        <AlertTriangle size={15} className="shrink-0 text-amber-500" />
        <span className="text-body font-semibold text-slate-900">
          Discard unsaved changes?
        </span>
      </div>
      <p className="px-5 py-4 text-body font-normal text-slate-600">
        This form has changes that have not been saved. Leaving now discards
        them.
      </p>
      <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-5 py-3">
        <Button variant="default" size="sm" onClick={props.onStay}>
          Keep editing
        </Button>
        <Button color="red" size="sm" onClick={props.onDiscard}>
          Discard changes
        </Button>
      </div>
    </div>
  </Modal>
);

const ResourceForm = (props: ResourceFormProps) => {
  const [req, setReq] = React.useState<Resource>(() =>
    cloneResource(props.item),
  );
  const [curYAML, setCurYAML] = React.useState(() =>
    resourceToYAML(props.item),
  );
  const [activeTab, setActiveTab] = React.useState<"main" | "yaml">("main");
  const [yamlError, setYamlError] = React.useState<string | null>(null);
  const [source, setSource] = React.useState<{
    item: Resource;
    version: number;
  }>(() => ({ item: cloneResource(props.item), version: 0 }));

  const lastItem = React.useRef(resourceFingerprint(props.item));

  const reseed = React.useCallback((item: Resource) => {
    setSource((prev) => ({ item, version: prev.version + 1 }));
  }, []);

  const baseline = React.useMemo(
    () => resourceFingerprint(props.item),
    [props.item],
  );

  React.useEffect(() => {
    const incoming = resourceFingerprint(props.item);
    if (incoming === lastItem.current) return;
    lastItem.current = incoming;
    const cloned = cloneResource(props.item);
    setReq(cloned);
    setCurYAML(resourceToYAML(cloned));
    setYamlError(null);
    reseed(cloneResource(props.item));
  }, [props.item, reseed]);

  const current = React.useMemo(() => resourceFingerprint(req), [req]);
  const isDirty = current !== baseline;
  const nameError = validateResourceName(req.metadata?.name ?? "");
  const canSubmit =
    !props.readOnly &&
    !props.isPending &&
    !nameError &&
    (activeTab !== "yaml" || yamlError === null) &&
    (!props.requireDirty || isDirty);

  useRegisterDirtyForm(isDirty && !props.readOnly);

  const submitting = React.useRef(false);
  const prevPending = React.useRef(props.isPending);

  React.useEffect(() => {
    if (prevPending.current && !props.isPending) {
      submitting.current = false;
    }
    prevPending.current = props.isPending;
  }, [props.isPending]);

  const blocker = useBlocker(({ currentLocation, nextLocation }) => {
    if (submitting.current || consumeNavigationApproval()) return false;
    return (
      isDirty &&
      !props.readOnly &&
      currentLocation.pathname !== nextLocation.pathname
    );
  });

  const yamlTimer = React.useRef<number | undefined>(undefined);

  React.useEffect(
    () => () => {
      if (yamlTimer.current) window.clearTimeout(yamlTimer.current);
    },
    [],
  );

  const handleYAMLChange = (value: string) => {
    setCurYAML(value);
    if (yamlTimer.current) window.clearTimeout(yamlTimer.current);
    yamlTimer.current = window.setTimeout(() => {
      const parsed = resourceFromYAML(value);
      setYamlError(parsed ? null : "Invalid YAML — cannot parse resource");
      if (parsed) {
        setReq(cloneResource(parsed));
        reseed(cloneResource(parsed));
      }
    }, YAML_PARSE_DELAY);
  };

  const handleTabChange = (value: string) => {
    const nextTab = value as "main" | "yaml";
    if (nextTab === "yaml") {
      setCurYAML(resourceToYAML(req));
      setYamlError(null);
    } else {
      const parsed = resourceFromYAML(curYAML);
      if (!parsed) {
        setYamlError("Invalid YAML — cannot parse resource");
        return;
      }
      setReq(cloneResource(parsed));
      reseed(cloneResource(parsed));
    }
    setActiveTab(nextTab);
  };

  const updateSpec = (item: Resource) => {
    const next = cloneResource(req);
    next.spec = item.spec;
    if (item.kind.endsWith("Secret")) {
      const withData = next as Resource & { data?: unknown };
      withData.data = (item as Resource & { data?: unknown }).data;
    }
    setReq(next);
  };

  const updateData = (item: Resource) => {
    const next = cloneResource(req) as Resource & { data?: unknown };
    next.data = (item as Resource & { data?: unknown }).data;
    setReq(next);
  };

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="flex items-center">
        <SegmentedControl
          value={activeTab}
          onChange={handleTabChange}
          disabled={props.readOnly}
          data={[
            {
              value: "main",
              label: (
                <span className="flex items-center gap-1.5 px-1">
                  <Settings size={13} strokeWidth={2.25} />
                  Configuration
                </span>
              ),
            },
            {
              value: "yaml",
              label: (
                <span className="flex items-center gap-1.5 px-1">
                  <FileCode size={13} strokeWidth={2.25} />
                  YAML
                </span>
              ),
            },
          ]}
        />
      </div>

      {activeTab === "main" ? (
        <div className="flex flex-col gap-6">
          {!props.noMetadata && req.metadata && (
            <ContainerGen title="Metadata">
              <fieldset
                disabled={props.readOnly}
                className={props.readOnly ? "opacity-60" : undefined}
              >
                <MetadataEdit
                  isUpdateMode={props.requireDirty}
                  item={req}
                  error={nameError}
                  onUpdate={(md) => {
                    const next = cloneResource(req);
                    next.metadata = md;
                    setReq(next);
                  }}
                />
              </fieldset>
            </ContainerGen>
          )}

          {props.specComponent && (
            <ContainerGen title="Spec">
              <fieldset
                disabled={props.readOnly}
                className={props.readOnly ? "opacity-60" : undefined}
              >
                <React.Suspense fallback={<FormSkeleton />}>
                  {React.createElement(props.specComponent, {
                    key: source.version,
                    item: source.item,
                    onUpdate: updateSpec,
                  })}
                </React.Suspense>
              </fieldset>
            </ContainerGen>
          )}

          {props.dataComponent && (
            <ContainerGen title="Data">
              <fieldset
                disabled={props.readOnly}
                className={props.readOnly ? "opacity-60" : undefined}
              >
                <React.Suspense fallback={<FormSkeleton />}>
                  {React.createElement(props.dataComponent, {
                    key: source.version,
                    item: source.item,
                    onUpdate: updateData,
                  })}
                </React.Suspense>
              </fieldset>
            </ContainerGen>
          )}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <ResourceEditor
            item={req}
            value={curYAML}
            onResourceChange={(item) => {
              setReq(cloneResource(item));
              reseed(cloneResource(item));
            }}
            onChange={handleYAMLChange}
          />

          {yamlError && (
            <div className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2">
              <X
                size={12}
                className="shrink-0 text-red-500"
                strokeWidth={2.5}
              />
              <span className="text-xs font-medium text-red-700">
                {yamlError}
              </span>
            </div>
          )}
        </div>
      )}

      <div className="sticky bottom-0 z-30 rounded-xl border border-slate-200 bg-white/95 shadow-raised backdrop-blur supports-[backdrop-filter]:bg-white/85">
        <div className="flex w-full flex-wrap items-center gap-3 px-4 py-3">
          <div className="flex min-w-0 flex-1 flex-col">
            {props.isError ? (
              <span className="text-xs font-medium text-red-600">
                {props.errorLabel}
              </span>
            ) : nameError ? (
              <span className="text-xs font-medium text-red-600">
                {nameError}
              </span>
            ) : (
              <span className="text-xs font-normal text-slate-500">
                {props.readOnly
                  ? "This resource is read-only"
                  : isDirty
                    ? "Unsaved changes"
                    : "No changes"}
              </span>
            )}
          </div>

          <Button
            variant="default"
            leftSection={<X size={13} strokeWidth={2.25} />}
            disabled={props.isPending}
            onClick={props.onCancel}
          >
            Cancel
          </Button>

          <Button
            variant="filled"
            color="ink"
            leftSection={
              props.isPending ? (
                <Loader2
                  size={13}
                  className="animate-spin"
                  strokeWidth={2.25}
                />
              ) : (
                props.submitIcon
              )
            }
            disabled={!canSubmit}
            loading={props.isPending}
            onClick={() => {
              submitting.current = true;
              props.onSubmit(req, curYAML, activeTab === "yaml");
            }}
          >
            {props.isPending ? props.submitPendingLabel : props.submitLabel}
          </Button>
        </div>
      </div>

      <DiscardModal
        opened={blocker.state === "blocked"}
        onStay={() => blocker.reset?.()}
        onDiscard={() => blocker.proceed?.()}
      />
    </div>
  );
};

export default ResourceForm;
