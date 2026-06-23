import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, Select, Textarea } from "@mantine/core";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Boxes, Layers, Search, Send } from "lucide-react";
import * as React from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";

import DurationInput from "../../../components/DurationInput";
import { Badge, Card, EmptyState, Eyebrow, Field, Loading } from "../../../ui";
import { getUserClient } from "../../../utils/client";
import ServicePicker from "./ServicePicker";

type Selection =
  | { kind: "service"; name: string; label: string }
  | { kind: "catalog"; name: string; label: string };

const TabButton = (props: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  children: React.ReactNode;
}) => (
  <button
    onClick={props.onClick}
    className={twMerge(
      "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[0.76rem] font-bold transition-colors duration-150",
      props.active
        ? "bg-slate-900 text-white"
        : "text-slate-500 hover:text-slate-900 hover:bg-slate-100",
    )}
  >
    {props.icon}
    {props.children}
  </button>
);

const CatalogCard = (props: {
  title: string;
  subtitle?: string;
  badge?: string;
  selected: boolean;
  onClick: () => void;
}) => (
  <button
    onClick={props.onClick}
    className={twMerge(
      "w-full flex items-center gap-3 text-left rounded-lg border px-3 py-2.5 transition-[border-color,box-shadow,background-color] duration-150",
      props.selected
        ? "border-slate-900 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
        : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
    )}
  >
    <div className="flex-1 min-w-0">
      <div className="text-[0.82rem] font-bold text-slate-800 truncate">
        {props.title}
      </div>
      {props.subtitle && (
        <div className="text-[0.7rem] font-semibold text-slate-400 truncate font-mono">
          {props.subtitle}
        </div>
      )}
    </div>
    {props.badge && <Badge tone="slate">{props.badge}</Badge>}
  </button>
);

const NewRequest = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const deepLinkApplied = React.useRef(false);
  const [tab, setTab] = React.useState<"service" | "catalog">("service");
  const [catalogQuery, setCatalogQuery] = React.useState("");
  const [selection, setSelection] = React.useState<Selection | undefined>();

  const [urgency, setUrgency] = React.useState<AccessP.Request_Spec_Urgency>(
    AccessP.Request_Spec_Urgency.NORMAL,
  );
  const [justification, setJustification] = React.useState("");
  const [duration, setDuration] = React.useState<MetaP.Duration>(
    MetaP.Duration.create({ type: { oneofKind: "hours", hours: 1 } as any }),
  );

  const catalogsQry = useQuery({
    queryKey: ["user", "listCatalog"],
    queryFn: async () => {
      const { response } = await getUserClient().listCatalog(
        AccessP.ListUserCatalogOptions.create({}),
      );
      return response;
    },
  });

  const catalogs = (catalogsQry.data?.items ?? []) as AccessP.Catalog[];

  React.useEffect(() => {
    if (deepLinkApplied.current) return;

    const svcName = searchParams.get("serviceRef.name");
    const catName = searchParams.get("catalogRef.name");

    if (svcName) {
      deepLinkApplied.current = true;
      setTab("service");
      setSelection({ kind: "service", name: svcName, label: svcName });
    } else if (catName) {
      if (catalogsQry.isLoading) return;
      deepLinkApplied.current = true;
      const cat = catalogs.find((c) => c.metadata?.name === catName);
      setTab("catalog");
      setSelection({
        kind: "catalog",
        name: catName,
        label: cat?.metadata?.displayName || cat?.metadata?.name || catName,
      });
    }
  }, [searchParams, catalogsQry.isLoading, catalogs]);

  const createMutation = useMutation({
    mutationFn: async () => {
      if (!selection) throw new Error("Select a resource to request access to");

      const resource: AccessP.Request_Spec_Resource =
        selection.kind === "service"
          ? AccessP.Request_Spec_Resource.create({
              type: {
                oneofKind: "serviceRef",
                serviceRef: MetaP.ObjectReference.create({
                  name: selection.name,
                }),
              },
            })
          : AccessP.Request_Spec_Resource.create({
              type: {
                oneofKind: "catalog",
                catalog: {
                  catalogRef: MetaP.ObjectReference.create({
                    name: selection.name,
                  }),
                },
              },
            });

      const req = AccessP.Request.create({
        apiVersion: "access/v1",
        kind: "Request",
        metadata: {},
        spec: {
          urgency,
          justification,
          duration,
          resource,
        },
      });

      const { response } = await getUserClient().createRequest(req);
      return response;
    },
    onSuccess: (response) => {
      toast.success("Access request submitted");
      navigate(`/user/requests/${response.metadata!.name}`);
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to submit request");
    },
  });

  const cq = catalogQuery.toLowerCase().trim();
  const filteredCatalogs = catalogs.filter(
    (c) =>
      !cq ||
      c.metadata?.name.toLowerCase().includes(cq) ||
      c.metadata?.displayName?.toLowerCase().includes(cq),
  );

  return (
    <div className="w-full">
      <div className="mb-6">
        <Eyebrow>Access</Eyebrow>
        <h1 className="text-[1.35rem] font-bold text-slate-900 leading-tight">
          New Request
        </h1>
        <p className="text-[0.82rem] font-medium text-slate-500">
          Request just-in-time access to a Service or an entire Catalog.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-5 items-start">
        <Card className="p-4">
          <div className="flex items-center gap-1 mb-3">
            <TabButton
              active={tab === "service"}
              onClick={() => {
                setTab("service");
                setSelection(undefined);
              }}
              icon={<Boxes size={14} strokeWidth={2.5} />}
            >
              Services
            </TabButton>
            <TabButton
              active={tab === "catalog"}
              onClick={() => {
                setTab("catalog");
                setSelection(undefined);
              }}
              icon={<Layers size={14} strokeWidth={2.5} />}
            >
              Catalogs
            </TabButton>
          </div>

          {tab === "service" ? (
            <ServicePicker
              value={selection?.kind === "service" ? selection.name : undefined}
              onChange={(svc) =>
                setSelection({
                  kind: "service",
                  name: svc.metadata!.name,
                  label: svc.metadata!.displayName || svc.metadata!.name,
                })
              }
            />
          ) : (
            <div className="flex flex-col gap-3">
              <div className="relative">
                <Search
                  size={13}
                  strokeWidth={2.5}
                  className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
                />
                <input
                  value={catalogQuery}
                  onChange={(e) => setCatalogQuery(e.target.value)}
                  placeholder="Search catalogs..."
                  className="w-full pl-8 pr-3 h-8 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 focus:shadow-[0_0_0_2px_rgba(148,163,184,0.2)] transition-all duration-150 placeholder:text-slate-400 placeholder:font-semibold"
                />
              </div>

              {catalogsQry.isLoading ? (
                <Loading />
              ) : filteredCatalogs.length === 0 ? (
                <EmptyState
                  icon={<Layers size={20} strokeWidth={2} />}
                  title="No catalogs available"
                  description="There are no Catalogs you can currently request access to."
                />
              ) : (
                <div className="flex flex-col gap-2 max-h-[460px] overflow-y-auto pr-0.5">
                  {filteredCatalogs.map((c) => {
                    const svcCount =
                      c.spec?.resourceCollection?.service?.services.length ?? 0;
                    return (
                      <CatalogCard
                        key={c.metadata!.uid || c.metadata!.name}
                        title={c.metadata!.displayName || c.metadata!.name}
                        subtitle={c.metadata!.name}
                        badge={
                          svcCount > 0 ? `${svcCount} services` : undefined
                        }
                        selected={
                          selection?.kind === "catalog" &&
                          selection.name === c.metadata!.name
                        }
                        onClick={() =>
                          setSelection({
                            kind: "catalog",
                            name: c.metadata!.name,
                            label: c.metadata!.displayName || c.metadata!.name,
                          })
                        }
                      />
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </Card>

        <Card className="p-5 lg:sticky lg:top-20">
          <div className="flex flex-col gap-4">
            <div>
              <Eyebrow>Request details</Eyebrow>
              {selection ? (
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-[0.85rem] font-bold text-slate-800 truncate">
                    {selection.label}
                  </span>
                  <Badge tone="slate">
                    {selection.kind === "service" ? "Service" : "Catalog"}
                  </Badge>
                </div>
              ) : (
                <p className="mt-2 text-[0.76rem] font-semibold text-slate-400">
                  Select a {tab} on the left to request access.
                </p>
              )}
            </div>

            <Field label="Urgency">
              <Select
                allowDeselect={false}
                data={[
                  {
                    label: "Very Low",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.VERY_LOW
                      ],
                  },
                  {
                    label: "Low",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.LOW
                      ],
                  },
                  {
                    label: "Normal",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.NORMAL
                      ],
                  },
                  {
                    label: "High",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.HIGH
                      ],
                  },
                  {
                    label: "Very High",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.VERY_HIGH
                      ],
                  },
                  {
                    label: "Highest",
                    value:
                      AccessP.Request_Spec_Urgency[
                        AccessP.Request_Spec_Urgency.HIGHEST
                      ],
                  },
                ]}
                value={AccessP.Request_Spec_Urgency[urgency]}
                onChange={(v) =>
                  v && setUrgency(AccessP.Request_Spec_Urgency[v as "NORMAL"])
                }
              />
            </Field>

            <Field
              label="Duration"
              description="How long you need the access for"
            >
              <DurationInput value={duration} onChange={setDuration} />
            </Field>

            <Field
              label="Justification"
              description="Explain why you need this access"
            >
              <Textarea
                autosize
                minRows={3}
                maxRows={6}
                placeholder="I need access to ..."
                value={justification}
                onChange={(e) => setJustification(e.currentTarget.value)}
              />
            </Field>

            <Button
              fullWidth
              variant="filled"
              color="dark"
              leftSection={<Send size={14} strokeWidth={2.5} />}
              disabled={!selection || justification.trim().length === 0}
              loading={createMutation.isPending}
              onClick={() => createMutation.mutate()}
            >
              Submit Request
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
};

export default NewRequest;
