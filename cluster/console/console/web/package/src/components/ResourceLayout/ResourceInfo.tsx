import {
  getRefNameQueryArgStr,
  getResourcePath,
  hasAccessLog,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
} from "@/utils/pb";

import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";

import TimeAgo from "@/components/TimeAgo";
import { Button, Text } from "@mantine/core";
import { Link } from "react-router-dom";
import Label from "../Label";
import ResourceYAML from "../ResourceYAML";

const Value = (props: { children?: React.ReactNode }) => {
  return (
    <div className="font-bold text-sm text-slate-600 flex items-center">
      {props.children}
    </div>
  );
};

const ResourceInfo = (props: { resource: Resource }) => {
  const item = props.resource;

  return (
    <div className="w-full">
      {item.metadata!.picURL.length > 0 && (
        <div className="mb-4">
          <img
            src={item.metadata!.picURL}
            className="w-20 h-20 rounded-full shadow"
          />
        </div>
      )}
      <InfoItem title="Name">
        <span className="flex items-center">
          <Value>
            <CopyText value={item.metadata!.name} />
          </Value>
          {item.metadata?.displayName && (
            <span className="ml-2 text-slate-500">
              {`(${item.metadata?.displayName})`}
            </span>
          )}
        </span>
      </InfoItem>
      <InfoItem title="UID">
        <Text size="xs" fw="bold" c="dimmed">
          <CopyText value={item.metadata!.uid} />
        </Text>
      </InfoItem>

      {item.metadata?.description && (
        <InfoItem title="Description">
          <Text size="sm" fw="bold">
            {item.metadata?.description}
          </Text>
        </InfoItem>
      )}

      {item.metadata?.isSystem && (
        <InfoItem title="System Resource">
          <Text size="sm" fw="bold">
            {`Yes`}
          </Text>
        </InfoItem>
      )}

      {item.metadata?.isUserHidden && (
        <InfoItem title="User Hidden">
          <Text size="sm" fw="bold">
            {`Yes`}
          </Text>
        </InfoItem>
      )}

      {item.metadata?.tags && item.metadata.tags.length > 0 && (
        <InfoItem title="Tags">
          <div className="flex items-center">
            {item.metadata.tags.map((x) => (
              <Label key={x}>{x}</Label>
            ))}
          </div>
        </InfoItem>
      )}

      <InfoItem title="Detailed Info">
        <div className="flex items-center">
          <ResourceYAML item={item} size="xs" />
          <Button
            size={"compact-xs"}
            variant="outline"
            component={Link}
            className="mx-1"
            to={`${getResourcePath(item)}`}
          >
            Details
          </Button>
        </div>
      </InfoItem>

      <InfoItem title="Created">
        <span className="flex items-center">
          <Value>
            <TimeAgo rfc3339={item.metadata?.createdAt} />
          </Value>

          {item.metadata?.updatedAt && (
            <span className="ml-3 text-slate-500">
              Updated <TimeAgo rfc3339={item.metadata?.updatedAt} />
            </span>
          )}
        </span>
      </InfoItem>

      {(hasAccessLog(item) ||
        hasAuthenticationLog(item) ||
        hasAuditLog(item)) && (
        <InfoItem title="Visibility">
          <div className="w-full flex">
            <ResourceVisibilityButtons item={item} />
          </div>
        </InfoItem>
      )}
    </div>
  );
};

export default ResourceInfo;

export const ResourceVisibilityButtons = (props: { item: Resource }) => {
  const { item } = props;
  const qryNameArg = getRefNameQueryArgStr(item);
  return (
    <>
      {hasAccessLog(item) && (
        <Button
          size={"compact-xs"}
          variant="outline"
          component={Link}
          className="mx-1"
          // rightSection={<IconAccessLog />}
          to={`/visibility/accesslogs?${qryNameArg}`}
        >
          Access Logs
        </Button>
      )}

      {hasAuthenticationLog(item) && (
        <Button
          size={"compact-xs"}
          variant="outline"
          component={Link}
          className="mx-1"
          // rightSection={<IconAuthenticationLog />}
          to={`/visibility/authenticationlogs?${qryNameArg}`}
        >
          Authentication Logs
        </Button>
      )}

      {hasAuditLog(item) && (
        <Button
          size={"compact-xs"}
          variant="outline"
          component={Link}
          className="mx-1"
          // rightSection={<MdModeEdit />}
          to={`/visibility/auditlogs?${qryNameArg}`}
        >
          Audit Logs
        </Button>
      )}

      {hasSSHSessionLog(item) && (
        <Button
          size={"compact-xs"}
          variant="outline"
          component={Link}
          className="mx-1"
          // rightSection={<MdModeEdit />}
          to={`/visibility/ssh?${qryNameArg}`}
        >
          SSH Sessions
        </Button>
      )}
    </>
  );
};
