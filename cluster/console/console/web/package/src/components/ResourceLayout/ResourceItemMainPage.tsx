import { Navigate } from "react-router-dom";
import PageWrap from "../PageWrap";

import { Resource } from "@/utils/pb";
import ResourceInfo from "./ResourceInfo";
import { useContextResource } from "./utils";

const ResourceItemMainPage = (props: {
  mainComponent?: (props: { item: Resource }) => React.ReactNode;
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

          {props.mainComponent && props.mainComponent({ item: ctx.data })}
        </div>
      )}
    </PageWrap>
  );
};

export default ResourceItemMainPage;
