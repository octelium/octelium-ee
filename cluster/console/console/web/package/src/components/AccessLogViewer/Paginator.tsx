import { ListResponseMeta } from "@/apis/metav1/metav1";

import { Pagination } from "@mantine/core";

const Paginator = (props: {
  listResponseMeta?: ListResponseMeta;
  onPageChange: (page: number) => void;
}) => {
  const meta = props.listResponseMeta;
  if (!meta) {
    return <></>;
  }

  const totalPages = Math.ceil(meta.totalCount / meta.itemsPerPage);

  return (
    <div className="w-full">
      <div className="flex items-center justify-center">
        <Pagination
          total={totalPages}
          radius={"xl"}
          value={meta.page + 1}
          withEdges
          color="#111"
          onChange={(v) => {
            props.onPageChange(v);
          }}
        />
      </div>
    </div>
  );
};
export default Paginator;
