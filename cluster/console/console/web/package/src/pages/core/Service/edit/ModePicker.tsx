import * as CoreP from "@/apis/corev1/corev1";
import ProviderPicker from "@/pages/enterprise/CollectorExporter/ProviderPicker";
import {
  MODE_CATEGORY_LABEL,
  ModeInfo,
  ModeName,
  MODES,
  ModeTile,
} from "./modes";

const ModePicker = (props: {
  value: CoreP.Service_Spec_Mode;
  onChange: (mode: CoreP.Service_Spec_Mode) => void;
}) => (
  <ProviderPicker<ModeName, ModeInfo["category"], ModeInfo>
    title="Mode"
    description="The application-layer protocol that this Service understands."
    noun="mode"
    placeholder="Pick the protocol of the upstream that this Service protects."
    providers={MODES}
    categoryLabels={MODE_CATEGORY_LABEL}
    value={MODES.find((item) => item.mode === props.value)?.kind}
    onChange={(kind) => props.onChange(CoreP.Service_Spec_Mode[kind])}
    tile={(kind, size) => (
      <ModeTile
        mode={kind ? CoreP.Service_Spec_Mode[kind] : undefined}
        size={size}
      />
    )}
    empty={(query) => <>No mode matches “{query}”.</>}
  />
);

export default ModePicker;
