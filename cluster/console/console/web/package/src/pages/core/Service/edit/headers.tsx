import * as CoreP from "@/apis/corev1/corev1";
import {
  Callout,
  Grid,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Checkbox, TextInput } from "@mantine/core";
import { Eye, Globe, Heading, Route } from "lucide-react";
import * as React from "react";
import {
  EnumField,
  joinSummary,
  ListField,
  listSummary,
  OptionRow,
  plural,
  RemoveButton,
  Segments,
  TagsField,
  TextField,
  Toggle,
  Update,
} from "./fields";
import { corsOriginError, headerNameError, pathPrefixError } from "./model";

type HeaderSpec = CoreP.Service_Spec_Config_HTTP_Header;
type KeyValue = CoreP.Service_Spec_Config_HTTP_Header_KeyValue;
type Host = CoreP.Service_Spec_Config_HTTP_Header_Host;

const ForwardedMode = CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode;
const AuthorizationMode =
  CoreP.Service_Spec_Config_HTTP_Header_AuthorizationMode;

const HeaderRow = (props: {
  label: string;
  header: KeyValue;
  index: number;
  update: Update;
  onRemove: () => void;
}) => {
  const { header, index, update, label } = props;
  const isEval = header.type.oneofKind === "eval";
  return (
    <div className="flex min-w-0 flex-wrap items-start gap-2">
      <TextInput
        aria-label={`${label} ${index + 1} name`}
        className="min-w-0 flex-[1_1_9rem]"
        placeholder="X-Header"
        value={header.key}
        error={header.key ? headerNameError(header.key) : undefined}
        onChange={(event) => {
          header.key = event.currentTarget.value;
          update();
        }}
      />
      <div className="flex h-9 shrink-0 items-center">
        <Segments
          label={`${label} ${index + 1} source`}
          hideLabel
          value={isEval ? "eval" : "value"}
          options={[
            { value: "value", label: "Value" },
            { value: "eval", label: "CEL" },
          ]}
          onChange={(next) => {
            header.type =
              next === "eval"
                ? { oneofKind: "eval", eval: "" }
                : { oneofKind: "value", value: "" };
            update();
          }}
        />
      </div>
      <TextInput
        aria-label={`${label} ${index + 1} value`}
        className="min-w-0 flex-[2_1_11rem]"
        classNames={isEval ? { input: "font-mono" } : undefined}
        placeholder={isEval ? "ctx.user.metadata.name" : "value"}
        value={
          header.type.oneofKind === "eval"
            ? header.type.eval
            : header.type.oneofKind === "value"
              ? header.type.value
              : ""
        }
        onChange={(event) => {
          const value = event.currentTarget.value;
          header.type = isEval
            ? { oneofKind: "eval", eval: value }
            : { oneofKind: "value", value };
          update();
        }}
      />
      <div className="flex h-9 shrink-0 items-center gap-2">
        <Checkbox
          size="xs"
          label="Append"
          aria-label={`Append ${label} ${index + 1}`}
          checked={header.append}
          onChange={(event) => {
            header.append = event.currentTarget.checked;
            update();
          }}
        />
        <RemoveButton
          label={`Remove ${label} ${index + 1}`}
          onClick={props.onRemove}
        />
      </div>
    </div>
  );
};

const HeaderRows = (props: {
  label: string;
  itemLabel: string;
  description: string;
  headers: KeyValue[];
  update: Update;
}) => {
  const rows = useListKeys();
  const { headers, update } = props;
  return (
    <ListField
      label={props.label}
      description={props.description}
      addLabel="Add header"
      count={headers.length}
      onAdd={() => {
        headers.push(
          CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create({
            type: { oneofKind: "value", value: "" },
          }),
        );
        update();
      }}
    >
      {headers.map((header, index) => (
        <HeaderRow
          key={rows.keyAt("headers", index)}
          label={props.itemLabel}
          header={header}
          index={index}
          update={update}
          onRemove={() => {
            rows.removeAt("headers", index);
            headers.splice(index, 1);
            update();
          }}
        />
      ))}
    </ListField>
  );
};

const hostKind = (host?: Host) =>
  host?.type.oneofKind === "preserve" && host.type.preserve
    ? "preserve"
    : host?.type.oneofKind === "value" || host?.type.oneofKind === "eval"
      ? host.type.oneofKind
      : "default";

const HostHeader = (props: { header: HeaderSpec; update: Update }) => {
  const { header, update } = props;
  const type = header.host?.type;
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Segments
        label="Host header"
        value={hostKind(header.host)}
        options={[
          { value: "default", label: "Upstream host" },
          { value: "preserve", label: "Service FQDN" },
          { value: "value", label: "Fixed" },
          { value: "eval", label: "CEL" },
        ]}
        onChange={(next) => {
          header.host =
            next === "default"
              ? undefined
              : CoreP.Service_Spec_Config_HTTP_Header_Host.create({
                  type:
                    next === "preserve"
                      ? { oneofKind: "preserve", preserve: true }
                      : next === "value"
                        ? { oneofKind: "value", value: "" }
                        : { oneofKind: "eval", eval: "" },
                });
          update();
        }}
      />
      {type?.oneofKind === "value" && (
        <TextInput
          aria-label="Host header value"
          placeholder="upstream.example.com"
          value={type.value}
          onChange={(event) => {
            type.value = event.currentTarget.value;
            update();
          }}
        />
      )}
      {type?.oneofKind === "eval" && (
        <TextInput
          aria-label="Host header CEL"
          classNames={{ input: "font-mono" }}
          placeholder={'ctx.service.metadata.name + ".internal"'}
          value={type.eval}
          onChange={(event) => {
            type.eval = event.currentTarget.value;
            update();
          }}
        />
      )}
    </div>
  );
};

const headerSummary = (header: HeaderSpec) =>
  joinSummary(
    header.addRequestHeaders.length > 0 &&
      `${header.addRequestHeaders.length} added to requests`,
    header.removeRequestHeaders.length > 0 &&
      `${header.removeRequestHeaders.length} removed from requests`,
    header.addResponseHeaders.length > 0 &&
      `${header.addResponseHeaders.length} added to responses`,
    header.removeResponseHeaders.length > 0 &&
      `${header.removeResponseHeaders.length} removed from responses`,
    hostKind(header.host) !== "default" && "Host rewritten",
  ) || "Default behavior";

export const HeadersRow = (props: {
  holder: { header?: HeaderSpec };
  update: Update;
}) => {
  const { holder, update } = props;
  const header = holder.header;
  return (
    <OptionRow
      icon={Heading}
      title="Headers"
      description="Add, remove or rewrite request and response headers."
      summary={header && headerSummary(header)}
      isSet={!!header}
      onAdd={() => {
        holder.header = CoreP.Service_Spec_Config_HTTP_Header.create();
        update();
      }}
      onRemove={() => {
        holder.header = undefined;
        update();
      }}
    >
      {header && (
        <>
          <HeaderRows
            label="Add to requests"
            itemLabel="Request header"
            description="Set before the request reaches the upstream."
            headers={header.addRequestHeaders}
            update={update}
          />
          <TagsField
            of={header}
            name="removeRequestHeaders"
            update={update}
            label="Remove from requests"
            placeholder="X-Debug"
          />
          <HeaderRows
            label="Add to responses"
            itemLabel="Response header"
            description="Set before the response reaches the client."
            headers={header.addResponseHeaders}
            update={update}
          />
          <TagsField
            of={header}
            name="removeResponseHeaders"
            update={update}
            label="Remove from responses"
            placeholder="X-Powered-By"
          />
          <HostHeader header={header} update={update} />
          <Grid>
            <EnumField
              of={header}
              name="forwardedMode"
              update={update}
              label="Forwarded header"
              placeholder="Drop"
              options={[
                { value: ForwardedMode.DROP, label: "Drop" },
                { value: ForwardedMode.OBFUSCATE, label: "Obfuscate" },
                { value: ForwardedMode.TRANSPARENT, label: "Pass through" },
              ]}
            />
            <EnumField
              of={header}
              name="authorizationMode"
              update={update}
              label="Authorization header"
              placeholder="Delete unless anonymous"
              options={[
                { value: AuthorizationMode.DELETE, label: "Delete" },
                { value: AuthorizationMode.PASS, label: "Pass through" },
              ]}
            />
          </Grid>
        </>
      )}
    </OptionRow>
  );
};

export const PathRow = (props: {
  holder: { path?: CoreP.Service_Spec_Config_HTTP_Path };
  update: Update;
}) => {
  const { holder, update } = props;
  const path = holder.path;
  return (
    <OptionRow
      icon={Route}
      title="Path rewrite"
      description="Change the path prefix before forwarding the request."
      summary={
        path &&
        (joinSummary(
          path.removePrefix && `Remove ${path.removePrefix}`,
          path.addPrefix && `Add ${path.addPrefix}`,
        ) ||
          "No change")
      }
      isSet={!!path}
      onAdd={() => {
        holder.path = CoreP.Service_Spec_Config_HTTP_Path.create();
        update();
      }}
      onRemove={() => {
        holder.path = undefined;
        update();
      }}
    >
      {path && (
        <Grid>
          <TextField
            of={path}
            name="removePrefix"
            update={update}
            label="Remove prefix"
            placeholder="/api"
            mono
            error={pathPrefixError}
          />
          <TextField
            of={path}
            name="addPrefix"
            update={update}
            label="Add prefix"
            placeholder="/v2"
            mono
            error={pathPrefixError}
          />
        </Grid>
      )}
    </OptionRow>
  );
};

export const CORSRow = (props: {
  holder: { cors?: CoreP.Service_Spec_Config_HTTP_CORS };
  update: Update;
  children?: React.ReactNode;
}) => {
  const { holder, update } = props;
  const cors = holder.cors;
  const originError = cors?.allowOriginStringMatch
    .map(corsOriginError)
    .find(Boolean);
  return (
    <OptionRow
      icon={Globe}
      title="CORS"
      description="Let browser applications on other origins call this Service."
      summary={
        cors &&
        (joinSummary(
          cors.allowOriginStringMatch.length > 0 &&
            listSummary(cors.allowOriginStringMatch, 2),
          cors.allowClusterServices && "Cluster Services",
          cors.allowCredentials && "Credentials",
        ) ||
          "No origins allowed")
      }
      isSet={!!cors}
      onAdd={() => {
        holder.cors = CoreP.Service_Spec_Config_HTTP_CORS.create();
        update();
      }}
      onRemove={() => {
        holder.cors = undefined;
        update();
      }}
    >
      {cors && (
        <>
          <TagsField
            of={cors}
            name="allowOriginStringMatch"
            update={update}
            label="Allowed origins"
            description="Exact origins, or * for every origin."
            placeholder="https://app.example.com"
            error={() => originError}
          />
          <ToggleList>
            <Toggle
              of={cors}
              name="allowClusterServices"
              update={update}
              label="Allow Cluster Services"
              description="Trust every Service of this Cluster, including the console playgrounds."
            />
            <Toggle
              of={cors}
              name="allowCredentials"
              update={update}
              label="Allow credentials"
              description="Send access-control-allow-credentials for cookies and HTTP authentication."
            />
          </ToggleList>
          {cors.allowClusterServices && (
            <Callout title="This trusts every Service in the Cluster">
              A browser application served by any of them can read this
              Service's responses on behalf of the browsing User.
            </Callout>
          )}
          <Grid>
            <TextField
              of={cors}
              name="allowMethods"
              update={update}
              label="Allowed methods"
              placeholder="GET, POST, OPTIONS"
            />
            <TextField
              of={cors}
              name="allowHeaders"
              update={update}
              label="Allowed headers"
              placeholder="Content-Type, Authorization"
            />
            <TextField
              of={cors}
              name="exposeHeaders"
              update={update}
              label="Exposed headers"
              placeholder="X-Request-ID"
            />
            <TextField
              of={cors}
              name="maxAge"
              update={update}
              label="Max age"
              placeholder="86400"
            />
          </Grid>
        </>
      )}
    </OptionRow>
  );
};

type HeaderVisibility = {
  includeRequestHeaders: string[];
  includeResponseHeaders: string[];
  includeAllRequestHeaders: boolean;
  includeAllResponseHeaders: boolean;
  excludeRequestHeaders: string[];
  excludeResponseHeaders: string[];
};

export const headerVisibilitySummary = (value: HeaderVisibility) => {
  const count =
    value.includeRequestHeaders.length + value.includeResponseHeaders.length;
  return joinSummary(
    value.includeAllRequestHeaders && "All request headers",
    value.includeAllResponseHeaders && "All response headers",
    count > 0 && plural(count, "header"),
  );
};

export const VisibilityHeaders = <T extends HeaderVisibility>(props: {
  value: T;
  update: Update;
}) => {
  const value: HeaderVisibility = props.value;
  const update = props.update;
  return (
    <Grid>
      <div className="flex min-w-0 flex-col gap-3">
        <ToggleList>
          <Toggle
            of={value}
            name="includeAllRequestHeaders"
            update={update}
            label="All request headers"
          />
        </ToggleList>
        {!value.includeAllRequestHeaders && (
          <TagsField
            of={value}
            name="includeRequestHeaders"
            update={update}
            label="Request headers to record"
            placeholder="X-Request-ID"
          />
        )}
        <TagsField
          of={value}
          name="excludeRequestHeaders"
          update={update}
          label="Request headers to skip"
          placeholder="Cookie"
        />
      </div>
      <div className="flex min-w-0 flex-col gap-3">
        <ToggleList>
          <Toggle
            of={value}
            name="includeAllResponseHeaders"
            update={update}
            label="All response headers"
          />
        </ToggleList>
        {!value.includeAllResponseHeaders && (
          <TagsField
            of={value}
            name="includeResponseHeaders"
            update={update}
            label="Response headers to record"
            placeholder="X-Request-ID"
          />
        )}
        <TagsField
          of={value}
          name="excludeResponseHeaders"
          update={update}
          label="Response headers to skip"
          placeholder="Set-Cookie"
        />
      </div>
    </Grid>
  );
};

export const VisibilityRow = (props: {
  summary?: string;
  isSet: boolean;
  onAdd: () => void;
  onRemove: () => void;
  children: React.ReactNode;
}) => (
  <OptionRow
    icon={Eye}
    title="Visibility"
    description="Choose what the AccessLogs record about each request."
    summary={props.summary}
    isSet={props.isSet}
    onAdd={props.onAdd}
    onRemove={props.onRemove}
  >
    {props.children}
  </OptionRow>
);
