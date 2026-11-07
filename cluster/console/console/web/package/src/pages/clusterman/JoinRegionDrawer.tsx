import { JoinClusterRequest } from "@/apis/enterprisev1/enterprisev1";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { getDomain, onError } from "@/utils";
import { getClientCluster } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import {
  Button,
  Drawer,
  SegmentedControl,
  Textarea,
  TextInput,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useMutation } from "@tanstack/react-query";
import { Info, ListChecks, PlusCircle, TriangleAlert, X } from "lucide-react";
import * as React from "react";
import {
  clustermanKeys,
  DEFAULT_REGION,
  isJoinIncomplete,
  PACKAGES,
  packageInfo,
  useClusterVersionInfo,
  useRegions,
} from "./queries";

type Mode = "kubeconfig" | "bearerToken";

type Form = {
  name: string;
  displayName: string;
  mode: Mode;
  secret: string;
  context: string;
  server: string;
  trustedCAs: string;
};

const emptyForm = (): Form => ({
  name: "",
  displayName: "",
  mode: "kubeconfig",
  secret: "",
  context: "",
  server: "",
  trustedCAs: "",
});

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,38}[a-z0-9]$/;

const CERT_END = "-----END CERTIFICATE-----";

const splitCertificates = (value: string) =>
  value
    .split(CERT_END)
    .map((item) => item.trim())
    .filter((item) => item !== "")
    .map((item) => `${item}\n${CERT_END}`);

const validateRegionName = (name: string, existing: string[]) => {
  if (name === "") return "A name is required";
  if (name === DEFAULT_REGION) return "The default Region already exists";
  if (!NAME_PATTERN.test(name) || name.includes("--"))
    return "Use lowercase letters, digits and single hyphens";
  if (existing.includes(name)) return "A Region with this name already exists";
  return undefined;
};

const validateServer = (server: string) => {
  if (server === "") return "The API server URL is required";
  try {
    const url = new URL(server);
    if (url.protocol !== "https:" || url.host === "")
      return "Use an https:// URL";
    if (url.username || url.password)
      return "The URL must not contain credentials";
  } catch {
    return "Invalid URL";
  }
  return undefined;
};

const buildJoinRequest = (form: Form) =>
  JoinClusterRequest.create({
    name: form.name.trim(),
    displayName: form.displayName.trim(),
    kubernetes: {
      type:
        form.mode === "kubeconfig"
          ? {
              oneofKind: "kubeconfig",
              kubeconfig: {
                type: { oneofKind: "fromSecret", fromSecret: form.secret },
                context: form.context.trim(),
              },
            }
          : {
              oneofKind: "bearerToken",
              bearerToken: {
                type: { oneofKind: "fromSecret", fromSecret: form.secret },
                server: form.server.trim(),
                trustedCAs: splitCertificates(form.trustedCAs),
              },
            },
    },
  });

const JoinRegionDrawer = (props: { opened: boolean; onClose: () => void }) => {
  const versions = useClusterVersionInfo();
  const regions = useRegions();
  const [form, setForm] = React.useState<Form>(emptyForm());
  const [touched, setTouched] = React.useState(false);

  const reset = () => {
    setForm(emptyForm());
    setTouched(false);
  };

  const existing = (regions.data?.items ?? [])
    .filter((item) => !isJoinIncomplete(item))
    .map((item) => item.metadata?.name ?? "");

  const nameError = validateRegionName(form.name.trim(), existing);
  const serverError =
    form.mode === "bearerToken"
      ? validateServer(form.server.trim())
      : undefined;
  const secretError = form.secret === "" ? "A Secret is required" : undefined;
  const invalid = !!nameError || !!serverError || !!secretError;

  const plan = PACKAGES.map((item) => ({
    key: item.key,
    label: item.label,
    version: packageInfo(versions.data, item.key)?.currentVersion,
  })).filter((item) => !!item.version);

  const update = (value: Partial<Form>) =>
    setForm((current) => ({ ...current, ...value }));

  const mutation = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCluster().joinCluster(
        buildJoinRequest(form),
      );
      return response;
    },
    onSuccess: () => {
      notifications.show({
        color: "green",
        message: `Joining the Region ${form.name.trim()} has started`,
      });
      invalidateKey(clustermanKeys.config);
      invalidateKey(clustermanKeys.regions);
      props.onClose();
    },
    onError,
  });

  const handleClose = () => {
    if (mutation.isPending) return;
    props.onClose();
  };

  const handleSubmit = () => {
    setTouched(true);
    if (invalid) return;
    mutation.mutate();
  };

  return (
    <Drawer
      opened={props.opened}
      onClose={handleClose}
      onExitTransitionEnd={reset}
      position="right"
      size="min(640px, 100vw)"
      padding={0}
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      transitionProps={{
        transition: "slide-left",
        duration: 260,
        exitDuration: 220,
      }}
      title={
        <div className="flex min-w-0 flex-col">
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            Cluster
          </span>
          <span className="truncate text-sm font-bold text-slate-900">
            Join a Region to {getDomain()}
          </span>
        </div>
      }
      styles={{
        header: {
          borderBottom: "1px solid var(--color-slate-200)",
          minHeight: "56px",
          paddingInline: "16px",
        },
        body: {
          height: "calc(100dvh - 56px)",
          padding: 0,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "var(--color-slate-50)",
        },
      }}
    >
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4">
        <div className="flex items-start gap-2.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
          <Info
            size={14}
            strokeWidth={2.4}
            className="mt-0.5 shrink-0 text-slate-500"
          />
          <div className="flex flex-col gap-1.5 text-xs font-normal leading-5 text-slate-600">
            <p>
              The Kubernetes cluster becomes a new Region that shares the
              PostgreSQL and Redis storage of the Cluster. Before joining it:
            </p>
            <ul className="list-disc space-y-0.5 pl-4">
              <li>
                Label its nodes with{" "}
                <code className="font-mono text-micro">
                  octelium.com/node-mode-controlplane
                </code>{" "}
                and{" "}
                <code className="font-mono text-micro">
                  octelium.com/node-mode-dataplane
                </code>
                .
              </li>
              <li>
                Make sure that it can reach the PostgreSQL and Redis addresses
                that the Cluster uses.
              </li>
              <li>
                Store its credentials in a Secret. They must be able to manage
                every Kubernetes resource.
              </li>
            </ul>
          </div>
        </div>

        <section className="flex flex-col gap-3">
          <h3 className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            Region
          </h3>
          <TextInput
            label="Name"
            required
            placeholder="e.g. us-east-1"
            description="The name cannot be changed later."
            value={form.name}
            error={touched || form.name !== "" ? nameError : undefined}
            onChange={(event) => update({ name: event.currentTarget.value })}
          />
          <TextInput
            label="Display name"
            placeholder="e.g. US East"
            value={form.displayName}
            onChange={(event) =>
              update({ displayName: event.currentTarget.value })
            }
          />
        </section>

        <section className="flex flex-col gap-3">
          <h3 className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            Kubernetes connection
          </h3>
          <SegmentedControl
            size="xs"
            fullWidth
            value={form.mode}
            onChange={(value) => update({ mode: value as Mode })}
            data={[
              { value: "kubeconfig", label: "Kubeconfig" },
              { value: "bearerToken", label: "Bearer token" },
            ]}
          />
          <SelectResource
            api="enterprise"
            kind="Secret"
            required
            label={
              form.mode === "kubeconfig" ? "Kubeconfig Secret" : "Token Secret"
            }
            description={
              form.mode === "kubeconfig"
                ? "Secret containing a kubeconfig with embedded credentials and certificates."
                : "Secret containing the bearer token (e.g. a ServiceAccount token)."
            }
            defaultValue={form.secret || undefined}
            onChange={(secret) =>
              update({ secret: secret?.metadata?.name ?? "" })
            }
          />
          {touched && secretError && (
            <span className="text-xs font-normal text-red-600">
              {secretError}
            </span>
          )}

          {form.mode === "kubeconfig" ? (
            <TextInput
              label="Context"
              placeholder="The current context of the kubeconfig"
              value={form.context}
              onChange={(event) =>
                update({ context: event.currentTarget.value })
              }
            />
          ) : (
            <>
              <TextInput
                label="API server URL"
                required
                placeholder="https://203.0.113.10:6443"
                value={form.server}
                error={touched || form.server !== "" ? serverError : undefined}
                onChange={(event) =>
                  update({ server: event.currentTarget.value })
                }
              />
              <Textarea
                label="Trusted CA certificates"
                description="PEM-encoded certificates that verify the API server. The system roots are used if empty."
                placeholder="-----BEGIN CERTIFICATE-----"
                autosize
                minRows={3}
                maxRows={8}
                value={form.trustedCAs}
                onChange={(event) =>
                  update({ trustedCAs: event.currentTarget.value })
                }
                styles={{
                  input: { fontFamily: "var(--font-mono)", fontSize: "0.72rem" },
                }}
              />
            </>
          )}
        </section>

        <section className="flex flex-col gap-2.5">
          <h3 className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
            <ListChecks size={12} strokeWidth={2.3} />
            Installed packages
          </h3>
          {plan.length === 0 ? (
            <div className="flex min-h-16 items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white px-4 text-center">
              <p className="text-xs font-normal text-slate-500">
                The current package versions are not known yet.
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
              {plan.map((item) => (
                <div
                  key={item.key}
                  className="flex min-w-0 items-center justify-between gap-3 py-1"
                >
                  <span className="truncate text-body font-semibold text-slate-700">
                    {item.label}
                  </span>
                  <span className="shrink-0 text-body font-semibold tabular-nums text-slate-700">
                    {item.version}
                  </span>
                </div>
              ))}
            </div>
          )}
          <div className="flex items-start gap-2.5 rounded-xl border border-amber-200 bg-amber-50/60 px-3.5 py-3">
            <TriangleAlert
              size={14}
              strokeWidth={2.4}
              className="mt-0.5 shrink-0 text-amber-600"
            />
            <p className="text-xs font-normal leading-5 text-amber-800">
              The new Region runs the same versions as the default Region.
              Upgrades are not possible until the join completes.
            </p>
          </div>
        </section>
      </div>

      <footer className="flex items-center justify-end gap-2 border-t border-slate-200 bg-white px-4 py-3.5">
        <Button
          variant="default"
          size="sm"
          leftSection={<X size={13} strokeWidth={2.5} />}
          disabled={mutation.isPending}
          onClick={handleClose}
        >
          Cancel
        </Button>

        <Button
          variant="filled"
          color="ink"
          size="sm"
          leftSection={<PlusCircle size={13} strokeWidth={2.5} />}
          disabled={touched && invalid}
          loading={mutation.isPending}
          onClick={handleSubmit}
        >
          {mutation.isPending ? "Starting…" : "Join Region"}
        </Button>
      </footer>
    </Drawer>
  );
};

export default JoinRegionDrawer;
