import {
  ComponentSelector,
  CounterOperation_Function,
  GaugeOperation_Function,
  HistogramOperation_Function,
  MetricDescriptor_Kind,
} from "@/apis/visibilityv1/metrics/vmetricsv1";
import MetricChart, {
  counterOp,
  eqFilter,
  gaugeOp,
  histogramOp,
} from "@/components/Charts/MetricChart";

const Metrics = () => {
  return (
    <div>
      <MetricChart
        title="Active requests — api"
        unit="requests"
        metric="req.active"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        component={ComponentSelector.create({ type: "vigil" })}
        filters={[eqFilter("octelium.vigil.svc.name", "api")]}
      />
      <MetricChart
        title="Memory — nocturne"
        unit="bytes"
        metric="process.mem.total"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        component={ComponentSelector.create({ type: "nocturne" })}
      />

      <MetricChart
        title="Heap allocated"
        unit="bytes"
        metric="process.mem.heap_alloc"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={["octelium.component.type"]}
      />
      <MetricChart
        title="Goroutines — octovigil"
        metric="process.goroutines"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        component={ComponentSelector.create({ type: "octovigil" })}
      />

      <MetricChart
        title="Avg authorization latency"
        unit="ms"
        metric="authorization.req.duration"
        operation={histogramOp(HistogramOperation_Function.AVG)}
        component={ComponentSelector.create({ type: "octovigil" })}
      />

      <MetricChart
        title="Request rate"
        unit="requests/s"
        metric="req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["octelium.component.type"]}
      />

      <MetricChart
        title="Request latency"
        unit="ms"
        metric="req.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.95, 0.99],
        )}
      />

      <MetricChart
        title="Active requests"
        unit="requests"
        metric="req.active"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={["octelium.component.type"]}
      />

      <MetricChart
        title="Authorization rate"
        unit="requests/s"
        metric="authorization.req.total"
        operation={counterOp(CounterOperation_Function.RATE)}
        component={ComponentSelector.create({ type: "octovigil" })}
      />

      <MetricChart
        title="Authorization latency"
        unit="us"
        metric="authorization.req.duration"
        operation={histogramOp(
          HistogramOperation_Function.QUANTILE,
          [0.5, 0.75, 0.95, 0.99],
        )}
        component={ComponentSelector.create({ type: "octovigil" })}
      />

      <MetricChart
        title="Memory"
        unit="bytes"
        metric="process.mem.heap_alloc"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={["octelium.component.type"]}
      />

      <MetricChart
        title="CPU usage"
        unit="cores"
        metric="process.cpu.seconds"
        operation={counterOp(CounterOperation_Function.RATE)}
        groupBy={["octelium.component.type"]}
      />

      <MetricChart
        title="Goroutines"
        metric="process.goroutines"
        operation={gaugeOp(GaugeOperation_Function.LAST)}
        groupBy={["octelium.component.type"]}
      />

      <MetricChart
        title="Authorizations per interval"
        unit="requests"
        metric="authorization.req.total"
        kind={MetricDescriptor_Kind.COUNTER}
        operation={counterOp(CounterOperation_Function.INCREASE)}
        component={ComponentSelector.create({ type: "octovigil" })}
      />

      <MetricChart
        title="Authorization rate by replica"
        unit="requests/s"
        metric="authorization.req.total"
        kind={MetricDescriptor_Kind.COUNTER}
        operation={counterOp(CounterOperation_Function.RATE)}
        component={ComponentSelector.create({ type: "octovigil" })}
        groupBy={["octelium.component.name"]}
      />
    </div>
  );
};

export default Metrics;
