import { ActionIcon, Tooltip } from "@mantine/core";
import { RefreshCw } from "lucide-react";

const RefreshButton = (props: {
  onClick: () => void;
  label?: string;
  loading?: boolean;
  disabled?: boolean;
}) => {
  const label = props.label ?? "Refresh";
  return (
    <Tooltip label={label} withArrow>
      <ActionIcon
        type="button"
        variant="default"
        size="sm"
        aria-label={label}
        aria-busy={props.loading || undefined}
        disabled={props.disabled}
        onClick={props.onClick}
      >
        <RefreshCw
          size={12}
          strokeWidth={2.5}
          className={props.loading ? "motion-safe:animate-spin" : ""}
        />
      </ActionIcon>
    </Tooltip>
  );
};

export default RefreshButton;
