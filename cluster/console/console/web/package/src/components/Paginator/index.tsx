import { ListResponseMeta } from "@/apis/metav1/metav1";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

import { Pagination } from "@mantine/core";

import { SegmentedControl } from "@mantine/core";

import { ArrowDownWideNarrow, ArrowUpWideNarrow } from "lucide-react";

const Paginator = (props: {
  meta?: ListResponseMeta;
  onPageChange?: (page: number) => void;
}) => {
  const meta = props.meta;
  const navigate = useNavigate();
  let [searchParams, _] = useSearchParams();

  const loc = useLocation();
  if (!meta) {
    return <></>;
  }

  const totalPages = Math.ceil(meta.totalCount / meta.itemsPerPage);

  return (
    <div className="w-full my-8">
      <div className="flex items-center">
        <Pagination
          total={totalPages}
          radius={"xl"}
          value={meta.page + 1}
          withEdges
          color="#111"
          classNames={{
            control: "!shadow-sm !font-semibold !transition-all !duration-500",
          }}
          onChange={(v) => {
            let page = v;
            searchParams.set("common.page", `${page}`);
            navigate(`${loc.pathname}?${searchParams.toString()}`);
            /*
            const i = v;
            if (props.onPageChange) {
              props.onPageChange(i);
            } else if (props.path) {
              navigate(
                `${props.path}${props.path.includes("?") ? "&" : "?"}page=${
                  i - 1
                }`
              );
            }
            */
          }}
        />

        {meta && meta.totalCount > 0 && (
          <div className="w-full flex items-center justify-end flex-1 ml-2">
            <div className="flex-1 mx-2 w-full">
              <span className="font-bold text-slate-700 text-sm">
                {meta.totalCount > 1 ? `${meta.totalCount} Items` : `1 Item`}
              </span>
            </div>
            {/*
            <TextInput
              value={qry}
              onChange={(v) => {
                setQry(v.target.value);
                if (v.target.value.length < 3) {
                  return;
                }
                searchParams.set("common.query", v.target.value);
                navigate(`${loc.pathname}?${searchParams.toString()}`);
              }}
            />
            */}
            <SegmentedControl
              value={searchParams.get("common.orderBy.type") ?? ""}
              classNames={{
                label: "!font-bold",
                control: "!font-bold",
              }}
              onChange={(v) => {
                searchParams.set("common.orderBy.type", v);
                navigate(`${loc.pathname}?${searchParams.toString()}`);
              }}
              data={[
                {
                  label: "Name",
                  value: "NAME",
                },
                {
                  label: "Creation Date",
                  value: "CREATED_AT",
                },
              ]}
            />

            <SegmentedControl
              value={searchParams.get("common.orderBy.mode") ?? ""}
              classNames={{
                label: "!font-bold",
                control: "!font-bold",
              }}
              onChange={(v) => {
                searchParams.set("common.orderBy.mode", v);
                navigate(`${loc.pathname}?${searchParams.toString()}`);
              }}
              data={[
                {
                  label: <ArrowUpWideNarrow />,
                  value: "ASC",
                },
                {
                  label: <ArrowDownWideNarrow />,
                  value: "DESC",
                },
              ]}
            />
          </div>
        )}
      </div>
    </div>
  );
};
export default Paginator;
