import ComponentLogViewer from "@/components/ComponentLogViewer";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";
import { useLogListReq } from "@/components/AccessLogViewer/listReq";
import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { ComponentSelector } from "@/apis/visibilityv1/visibilityv1";
import { useSearchParams } from "react-router-dom";

export default () => {
  const req = useLogListReq();
  const [searchParams, setSearchParams] = useSearchParams();
  const pagination = useLogPageParams();
  const levels = [
    ...(req?.level ? [req.level] : []),
    ...(searchParams.get("levels")?.split(",") ?? []),
  ]
    .map(
      (itm) =>
        ComponentLog_Entry_Level[
          itm.trim().toUpperCase() as keyof typeof ComponentLog_Entry_Level
        ],
    )
    .filter(
      (itm, idx, arr): itm is ComponentLog_Entry_Level =>
        typeof itm === "number" && itm > 0 && arr.indexOf(itm) === idx,
    );
  return (
    <LogPageShell title="Component logs">
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
        levels={levels}
        onLevelsChange={(nextLevels) => {
          const next = new URLSearchParams(searchParams);
          next.delete("common.page");
          next.delete("level");
          next.delete("levels");
          if (nextLevels.length === 1) {
            next.set("level", ComponentLog_Entry_Level[nextLevels[0]]);
          } else if (nextLevels.length > 1) {
            next.set(
              "levels",
              nextLevels.map((itm) => ComponentLog_Entry_Level[itm]).join(","),
            );
          }
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
