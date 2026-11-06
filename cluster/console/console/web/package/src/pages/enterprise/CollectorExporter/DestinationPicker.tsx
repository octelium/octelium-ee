import ProviderPicker from "./ProviderPicker";
import {
  CATEGORY_LABEL,
  ExporterKind,
  ExporterTile,
  PROVIDERS,
  SignalBadges,
} from "./providers";

const DestinationPicker = (props: {
  value?: ExporterKind;
  onChange: (kind: ExporterKind) => void;
}) => (
  <ProviderPicker
    title="Destination"
    description="Where the Collector sends the logs and metrics of its pipelines."
    noun="destination"
    placeholder="Pick the backend that receives the telemetry."
    providers={PROVIDERS}
    categoryLabels={CATEGORY_LABEL}
    value={props.value}
    onChange={props.onChange}
    tile={(kind, size) => <ExporterTile kind={kind} size={size} />}
    badges={(provider, className) => (
      <SignalBadges signals={provider.signals} className={className} />
    )}
    searchText={(provider) => provider.signals.join(" ")}
    empty={(query) => (
      <>
        No destination matches “{query}”. Most observability vendors accept
        OTLP, so try OTLP gRPC or OTLP HTTP.
      </>
    )}
  />
);

export default DestinationPicker;
