import { Navigate } from "react-router-dom";
import PageWrap from "../PageWrap";

import { ResourceMainInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";
import InfoItem from "../InfoItem";
import ResourceInfo from "./ResourceInfo";
import { useContextResource } from "./utils";

const ResourceItemMainPage = (props: {
  mainComponent?: (props: { item: Resource }) => React.ReactNode;
  mainItemsGetter?: (props: { item: Resource }) => ResourceMainInfo;
}) => {
  const ctx = useContextResource();

  if (ctx?.isError) {
    return <Navigate to={`/`} />;
  }

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx?.data && (
        <div className="w-full">
          <div className="w-full">
            <ResourceInfo resource={ctx.data} />
          </div>

          {props.mainItemsGetter
            ? props.mainItemsGetter({ item: ctx.data }).items?.map((x) => {
                return (
                  <div>
                    <InfoItem title={x.label}>{x.value}</InfoItem>
                  </div>
                );
              })
            : props.mainComponent && props.mainComponent({ item: ctx.data })}
        </div>
      )}
    </PageWrap>
  );
};

export default ResourceItemMainPage;
