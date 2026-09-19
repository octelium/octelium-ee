import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  EmptyHint,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { useAccessSummary, useAccessTop } from "@/components/Dashboard/queries";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TopList from "@/components/TopList";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Boxes, EyeOff, Globe, PanelTop, PowerOff, Shield } from "lucide-react";
import { useCoreCreated, useCoreTotals } from "./queries";

const Services = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCoreCreated(periodMinutes, QUERY_PRIORITY.high);
  const access = useAccessSummary(periodMinutes, QUERY_PRIORITY.normal);
  const topServices = useAccessTop(
    "service",
    periodMinutes,
    QUERY_PRIORITY.normal,
  );

  const services = totals.data?.core?.service;
  const namespaces = totals.data?.core?.namespace;
  const policies = totals.data?.core?.policy;
  const newServices = created.data?.core?.service;
  const newPolicies = created.data?.core?.policy;

  const modes = [
    { label: "HTTP", value: n(services?.totalHTTP) },
    { label: "Web", value: n(services?.totalWeb) },
    { label: "TCP", value: n(services?.totalTCP) },
    { label: "SSH", value: n(services?.totalSSH) },
    { label: "Kubernetes", value: n(services?.totalKubernetes) },
    { label: "gRPC", value: n(services?.totalGRPC) },
    { label: "PostgreSQL", value: n(services?.totalPostgres) },
    { label: "MySQL", value: n(services?.totalMysql) },
    { label: "UDP", value: n(services?.totalUDP) },
    { label: "DNS", value: n(services?.totalDNS) },
    { label: "SOCKS5", value: n(services?.totalSOCKS5) },
    { label: "RDP", value: n(services?.totalRDPWeb) },
    { label: "LLM", value: n(services?.totalLLM) },
    { label: "MCP", value: n(services?.totalMCP) },
  ].filter((mode) => mode.value > 0);

  const traffic = Object.entries(access.data?.totalByMode ?? {})
    .map(([label, value]) => ({ label, value: n(value) }))
    .sort((a, b) => b.value - a.value);

  const serviceItems = (topServices.data?.items ?? [])
    .map((item: any) => ({ resource: item.service, count: item.count }))
    .filter((item) => !!item.resource);

  return (
    <Panel
      icon={PanelTop}
      title="Services & exposure"
      description={`What the Cluster publishes and what was actually reached in the last ${rangeLabel}`}
      to="/core/services"
      toLabel="All Services"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Services"
            value={n(services?.totalNumber)}
            icon={PanelTop}
            to="/core/services"
          />
          <MiniStat
            label="Namespaces"
            value={n(namespaces?.totalNumber)}
            icon={Boxes}
            to="/core/namespaces"
          />
          <MiniStat
            label="Public"
            value={n(services?.totalPublic)}
            tone={n(services?.totalPublic) > 0 ? "warning" : "default"}
            icon={Globe}
            to="/core/services?isPublic=true"
          />
          <MiniStat
            label="Anonymous"
            value={n(services?.totalAnonymous)}
            tone={n(services?.totalAnonymous) > 0 ? "critical" : "default"}
            icon={EyeOff}
            to="/core/services?isAnonymous=true"
          />
          <MiniStat
            label="Disabled"
            value={n(services?.totalDisabled)}
            icon={PowerOff}
            to="/core/services?isDisabled=true"
          />
          <MiniStat
            label="Policies"
            value={n(policies?.totalNumber)}
            icon={Shield}
            to="/core/policies"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Services by mode"
            note={`${compact(n(services?.totalNumber))} published`}
          >
            {modes.length > 0 ? (
              <CompositionBar
                segments={modes}
                total={n(services?.totalNumber)}
              />
            ) : (
              <span className="text-micro font-normal text-slate-500">
                No Services are published yet
              </span>
            )}
          </Breakdown>

          <Breakdown
            title={`Traffic by mode · ${rangeLabel}`}
            note={`${compact(n(access.data?.totalNumber))} requests`}
          >
            {traffic.length > 0 ? (
              <CompositionBar
                segments={traffic}
                total={n(access.data?.totalNumber)}
              />
            ) : (
              <span className="text-micro font-normal text-slate-500">
                No Service was reached in the last {rangeLabel}
              </span>
            )}
          </Breakdown>
        </div>

        <Breakdown
          title="Authorization rules"
          note={`${compact(n(policies?.totalRule))} rules across ${compact(n(policies?.totalNumber))} Policies`}
        >
          <CompositionBar
            segments={[
              {
                label: "Allow",
                value: n(policies?.totalRuleAllow),
                color: STATUS_COLORS.good,
              },
              {
                label: "Deny",
                value: n(policies?.totalRuleDenied),
                color: STATUS_COLORS.critical,
              },
            ]}
            total={n(policies?.totalRule)}
          />
          {n(policies?.totalDisabled) > 0 && (
            <span className="text-micro font-semibold text-amber-700">
              {compact(n(policies?.totalDisabled))} Policies are disabled
            </span>
          )}
        </Breakdown>

        <DeltaStatGrid>
          <DeltaStat
            label={`New Services · ${rangeLabel}`}
            value={n(newServices?.totalNumber)}
            prev={n(newServices?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={PanelTop}
            to="/core/services"
          />
          <DeltaStat
            label="New public"
            value={n(newServices?.totalPublic)}
            prev={n(newServices?.previous?.totalPublic)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={Globe}
            to="/core/services?isPublic=true"
          />
          <DeltaStat
            label="New anonymous"
            value={n(newServices?.totalAnonymous)}
            prev={n(newServices?.previous?.totalAnonymous)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={EyeOff}
            to="/core/services?isAnonymous=true"
          />
          <DeltaStat
            label="New Policies"
            value={n(newPolicies?.totalNumber)}
            prev={n(newPolicies?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Shield}
            to="/core/policies"
          />
        </DeltaStatGrid>

        {serviceItems.length > 0 ? (
          <TopList
            title="Most reached Services"
            to="/visibility/accesslogs"
            items={serviceItems}
          />
        ) : (
          !topServices.isLoading && (
            <EmptyHint>
              No Service received a request in the last {rangeLabel}.
            </EmptyHint>
          )
        )}
      </div>
    </Panel>
  );
};

export default Services;
