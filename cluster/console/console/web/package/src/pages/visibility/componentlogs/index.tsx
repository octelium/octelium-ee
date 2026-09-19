import ComponentLogViewer from "@/components/ComponentLogViewer";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";
import { useLogListReq } from "@/components/AccessLogViewer/listReq";
import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { ComponentSelector } from "@/apis/visibilityv1/visibilityv1";
import { useSearchParams } from "react-router-dom";
import { ScrollText } from "lucide-react";

export default () => {
  const req = useLogListReq();
  const [searchParams, setSearchParams] = useSearchParams();
  const pagination = useLogPageParams();
  const level = req?.level
    ? ComponentLog_Entry_Level[
        req.level.toUpperCase() as keyof typeof ComponentLog_Entry_Level
      ]
    : undefined;
  return (
    <LogPageShell
      title="Component logs"
      description="Runtime messages from Octelium control-plane and data-plane components."
      icon={ScrollText}
    >
      <ComponentLogViewer
        component={
          req?.component
            ? ComponentSelector.create({
                namespace: req.component.namespace,
                type: req.component.type,
                uid: req.component.uid,
              })
            : undefined
        }
        level={typeof level === "number" ? level : undefined}
        onLevelChange={(nextLevel) => {
          const next = new URLSearchParams(searchParams);
          next.delete("common.page");
          if (nextLevel === undefined) next.delete("level");
          else next.set("level", ComponentLog_Entry_Level[nextLevel]);
          setSearchParams(next, {
            replace: true,
            preventScrollReset: true,
          });
        }}
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
        query={req?.common?.query}
      />
    </LogPageShell>
  );
};
