import { Button, Tooltip } from "@mantine/core";
import { LifeBuoy } from "lucide-react";
import { useNavigate } from "react-router";
import {
  getSupportReportPath,
  type SupportReport,
  useSupportAvailable,
} from "./context";

const ReportButton = (props: { report: () => SupportReport }) => {
  const available = useSupportAvailable();
  const navigate = useNavigate();
  if (!available) return null;

  return (
    <Tooltip
      label="Report this entry to Octelium Support"
      withArrow
      openDelay={300}
    >
      <Button
        type="button"
        size="compact-xs"
        variant="outline"
        leftSection={<LifeBuoy size={11} strokeWidth={2.5} />}
        onClick={() => navigate(getSupportReportPath(props.report()))}
        styles={{ root: { fontSize: "0.68rem", fontWeight: 700 } }}
      >
        Report
      </Button>
    </Tooltip>
  );
};

export default ReportButton;
