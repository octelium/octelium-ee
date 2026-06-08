import { Service } from "@/apis/corev1/corev1";
import {
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import PageWrap from "@/components/PageWrap";
import MetricChart, {
  counterOp,
  eqFilter,
  gaugeOp,
  histogramOp,
} from "../Charts/MetricChart";
import { useContextResource } from "./utils";

export const ServiceMetrics = (props: { resource: Service }) => {
  const { resource } = props;
  if (resource.apiVersion !== `core/v1`) {
    return <></>;
  }

  const svcFilters = [
    eqFilter("octelium.vigil.svc.name", resource.metadata!.name),
    eqFilter(
      "octelium.vigil.svc.namespace.name",
      resource.status!.namespaceRef!.name,
    ),
  ];
  const vigil = ComponentSelector.create({ type: "vigil" });

  return (
    <div>
      <div className="w-full">
        <div className="w-full">
          <div>
            <MetricChart
              title="Request rate"
              unit="requests/s"
              metric="req.total"
              operation={counterOp(CounterOperation_Function.RATE)}
              component={vigil}
              filters={svcFilters}
            />

            <MetricChart
              title="Active requests"
              unit="requests"
              metric="req.active"
              operation={gaugeOp(GaugeOperation_Function.LAST)}
              component={vigil}
              filters={svcFilters}
            />

            <MetricChart
              title="Latency"
              unit="ms"
              metric="req.duration"
              operation={histogramOp(
                HistogramOperation_Function.QUANTILE,
                [0.5, 0.95, 0.99],
              )}
              component={vigil}
              filters={svcFilters}
            />
          </div>
          <div>
            <MetricChart
              title="Request rate by Service"
              unit="requests/s"
              metric="req.total"
              operation={counterOp(CounterOperation_Function.RATE)}
              component={vigil}
              groupBy={["octelium.vigil.svc.name"]}
              limitSeries={12}
            />

            <MetricChart
              title="Active requests by mode"
              unit="requests"
              metric="req.active"
              operation={gaugeOp(GaugeOperation_Function.LAST)}
              component={vigil}
              groupBy={["octelium.vigil.svc.mode"]}
            />

            <MetricChart
              title="Request rate by namespace"
              unit="requests/s"
              metric="req.total"
              operation={counterOp(CounterOperation_Function.RATE)}
              component={vigil}
              groupBy={["octelium.vigil.svc.namespace.name"]}
            />

            <MetricChart
              title="Request rate by region"
              unit="requests/s"
              metric="req.total"
              operation={counterOp(CounterOperation_Function.RATE)}
              component={vigil}
              groupBy={["octelium.vigil.svc.region.name"]}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

const ServiceMetricsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ServiceMetrics resource={ctx.data as Service} />}
    </PageWrap>
  );
};

export default ServiceMetricsPage;
