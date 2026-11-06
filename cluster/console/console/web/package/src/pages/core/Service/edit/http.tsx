import * as CoreP from "@/apis/corev1/corev1";
import {
  Callout,
  FormSection,
  Grid,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { Button, TagsInput } from "@mantine/core";
import {
  FileJson2,
  KeyRound,
  MessageSquareText,
  RefreshCw,
  SlidersHorizontal,
} from "lucide-react";
import * as React from "react";
import { AuthEditor } from "./auth";
import {
  BodyField,
  CodeField,
  DurationField,
  EnumField,
  joinSummary,
  jsonError,
  NumberField,
  OptionList,
  OptionRow,
  plural,
  Scope,
  StatusField,
  TextField,
  Toggle,
  Update,
} from "./fields";
import {
  CORSRow,
  headerVisibilitySummary,
  HeadersRow,
  PathRow,
  VisibilityHeaders,
  VisibilityRow,
} from "./headers";
import { HTTP_PLUGINS, PluginsSection } from "./plugins";

type HTTP = CoreP.Service_Spec_Config_HTTP;

const BodyMode = CoreP.Service_Spec_Config_HTTP_Body_Mode;

export const TransportCard = (props: {
  holder: { isUpstreamHTTP2: boolean; listenHTTP2: boolean };
  scope: Scope;
  noHTTP2?: boolean;
  update: Update;
  children?: React.ReactNode;
}) => {
  const { holder, update, scope } = props;
  return (
    <FormSection
      icon={SlidersHorizontal}
      title="Transport"
      description="How the Service speaks to clients and to the upstream."
    >
      <ToggleList>
        {!props.noHTTP2 && (
          <Toggle
            of={holder}
            name="isUpstreamHTTP2"
            update={update}
            label="HTTP/2 to the upstream"
          />
        )}
        {!props.noHTTP2 && scope === "default" && (
          <Toggle
            of={holder}
            name="listenHTTP2"
            update={update}
            label="HTTP/2 from clients"
          />
        )}
        {props.children}
      </ToggleList>
    </FormSection>
  );
};

const RetryRow = (props: { value: HTTP; update: Update }) => {
  const { value, update } = props;
  const retry = value.retry;
  return (
    <OptionRow
      icon={RefreshCw}
      title="Retry"
      description="Retry failed upstream requests with an exponential backoff."
      summary={
        retry &&
        joinSummary(
          plural(retry.maxRetries, "retry", "retries"),
          retry.statusCodes.length > 0 && retry.statusCodes.join(", "),
          retry.retryOnServerErrors && "All 5xx",
        )
      }
      isSet={!!retry}
      onAdd={() => {
        value.retry = CoreP.Service_Spec_Config_HTTP_Retry.create();
        update();
      }}
      onRemove={() => {
        value.retry = undefined;
        update();
      }}
    >
      {retry && (
        <>
          <Grid>
            <NumberField
              of={retry}
              name="maxRetries"
              update={update}
              label="Max retries"
              placeholder="3"
              max={100}
            />
            <NumberField
              of={retry}
              name="multiplier"
              update={update}
              label="Backoff multiplier"
              placeholder="2"
              decimals={2}
              step={0.1}
            />
          </Grid>
          <Grid cols={3}>
            <DurationField
              of={retry}
              name="initialInterval"
              update={update}
              label="First backoff"
            />
            <DurationField
              of={retry}
              name="maxInterval"
              update={update}
              label="Longest backoff"
            />
            <DurationField
              of={retry}
              name="maxElapsedTime"
              update={update}
              label="Give up after"
            />
          </Grid>
          <TagsInput
            label="Status codes"
            description="The default is 502, 503 and 504."
            placeholder={retry.statusCodes.length ? undefined : "503"}
            data={["429", "500", "502", "503", "504"]}
            splitChars={[",", " "]}
            acceptValueOnBlur
            value={retry.statusCodes.map(String)}
            onChange={(next) => {
              retry.statusCodes = next
                .map(Number)
                .filter((code) => code >= 200 && code <= 599);
              update();
            }}
          />
          <ToggleList>
            <Toggle
              of={retry}
              name="retryOnServerErrors"
              update={update}
              label="Retry every 5xx response"
            />
          </ToggleList>
        </>
      )}
    </OptionRow>
  );
};

const BodyRow = (props: { value: HTTP; scope: Scope; update: Update }) => {
  const { value, scope, update } = props;
  const body = value.body;
  const validation = body?.validation;
  const schema =
    validation?.type.oneofKind === "jsonSchema" &&
    validation.type.jsonSchema.type.oneofKind === "inline"
      ? validation.type.jsonSchema.type.inline
      : undefined;
  const buffered =
    value.enableRequestBuffering || value.auth?.type.oneofKind === "sigv4";

  return (
    <OptionRow
      icon={FileJson2}
      title="Request body"
      description="Limit the size of the body and validate its content."
      summary={
        body &&
        (joinSummary(
          body.mode === BodyMode.JSON && "JSON",
          !!body.maxRequestSize &&
            `Up to ${body.maxRequestSize.toLocaleString()} bytes`,
          schema !== undefined && "Schema validated",
        ) ||
          "No checks")
      }
      isSet={!!body}
      onAdd={() => {
        value.body = CoreP.Service_Spec_Config_HTTP_Body.create();
        if (scope === "default") value.enableRequestBuffering = true;
        update();
      }}
      onRemove={() => {
        value.body = undefined;
        update();
      }}
    >
      {body && (
        <>
          {!buffered && scope === "default" && (
            <Callout title="These checks need request buffering">
              <span className="flex flex-col items-start gap-2">
                <span>The Service has to read the whole body first.</span>
                <Button
                  type="button"
                  variant="default"
                  size="compact-xs"
                  onClick={() => {
                    value.enableRequestBuffering = true;
                    update();
                  }}
                >
                  Enable request buffering
                </Button>
              </span>
            </Callout>
          )}
          <Grid>
            <NumberField
              of={body}
              name="maxRequestSize"
              update={update}
              label="Largest body"
              suffix=" bytes"
              thousands
            />
            <EnumField
              of={body}
              name="mode"
              update={update}
              label="Format"
              placeholder="Any"
              options={[{ value: BodyMode.JSON, label: "JSON" }]}
            />
          </Grid>
          <ToggleList>
            <div className="px-3.5 py-3">
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs font-semibold text-slate-700">
                  JSON Schema validation
                </span>
                <Button
                  type="button"
                  variant="default"
                  size="compact-xs"
                  onClick={() => {
                    body.validation =
                      schema !== undefined
                        ? undefined
                        : CoreP.Service_Spec_Config_HTTP_Body_Validation.create({
                            type: {
                              oneofKind: "jsonSchema",
                              jsonSchema: {
                                type: { oneofKind: "inline", inline: "" },
                              },
                            },
                          });
                    update();
                  }}
                >
                  {schema !== undefined ? "Remove" : "Add schema"}
                </Button>
              </div>
              {schema !== undefined && (
                <div className="mt-3">
                  <CodeField
                    label="Schema"
                    description="Requests that do not match are rejected with status 400."
                    mode="json"
                    value={schema}
                    error={jsonError(schema)}
                    onChange={(inline) => {
                      body.validation =
                        CoreP.Service_Spec_Config_HTTP_Body_Validation.create({
                          type: {
                            oneofKind: "jsonSchema",
                            jsonSchema: { type: { oneofKind: "inline", inline } },
                          },
                        });
                      update();
                    }}
                  />
                </div>
              )}
            </div>
          </ToggleList>
        </>
      )}
    </OptionRow>
  );
};

const ResponseRow = (props: { value: HTTP; update: Update }) => {
  const { value, update } = props;
  const type = value.response?.type;
  const direct = type?.oneofKind === "direct" ? type.direct : undefined;
  return (
    <OptionRow
      icon={MessageSquareText}
      title="Direct response"
      description="Answer every request here, without reaching the upstream."
      summary={
        direct &&
        (joinSummary(
          !!direct.statusCode && `Status ${direct.statusCode}`,
          direct.contentType,
        ) ||
          "Status 200")
      }
      isSet={!!direct}
      onAdd={() => {
        value.response = CoreP.Service_Spec_Config_HTTP_Response.create({
          type: {
            oneofKind: "direct",
            direct: CoreP.Service_Spec_Config_HTTP_Response_Direct.create({
              type: { oneofKind: "inline", inline: "" },
            }),
          },
        });
        update();
      }}
      onRemove={() => {
        value.response = undefined;
        update();
      }}
    >
      {direct && (
        <>
          <Grid>
            <StatusField
              of={direct}
              name="statusCode"
              update={update}
              placeholder="200"
            />
            <TextField
              of={direct}
              name="contentType"
              update={update}
              label="Content type"
              placeholder="application/json"
            />
          </Grid>
          <BodyField
            label="Body"
            holder={direct}
            update={update}
            placeholder={'{ "status": "ok" }'}
          />
        </>
      )}
    </OptionRow>
  );
};

export const HTTPForm = (props: {
  value: HTTP;
  mode: CoreP.Service_Spec_Mode;
  scope: Scope;
  update: Update;
}) => {
  const { value, mode, scope, update } = props;
  const global = scope === "default";
  const visibility = value.visibility;

  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Upstream authentication"
        description="Credentials that the Service injects, so that Users never see them."
      >
        <AuthEditor holder={value} update={update} />
      </FormSection>

      <OptionList title="Requests and responses">
        <HeadersRow holder={value} update={update} />
        <PathRow holder={value} update={update} />
        <BodyRow value={value} scope={scope} update={update} />
        <ResponseRow value={value} update={update} />
      </OptionList>

      <PluginsSection
        holder={value}
        kind="http"
        definitions={HTTP_PLUGINS}
        scope={scope}
        update={update}
        description="Rate limit, cache, rewrite and extend requests per Condition."
      />

      <OptionList title="Reliability and logging">
        <RetryRow value={value} update={update} />
        {global && <CORSRow holder={value} update={update} />}
        <VisibilityRow
          isSet={!!visibility}
          summary={
            visibility &&
            (joinSummary(
              visibility.enableRequestBody && "Request bodies",
              visibility.enableRequestBodyMap && "Request JSON",
              visibility.enableResponseBody && "Response bodies",
              visibility.enableResponseBodyMap && "Response JSON",
              headerVisibilitySummary(visibility),
            ) ||
              "Nothing extra recorded")
          }
          onAdd={() => {
            value.visibility = CoreP.Service_Spec_Config_HTTP_Visibility.create();
            update();
          }}
          onRemove={() => {
            value.visibility = undefined;
            update();
          }}
        >
          {visibility && (
            <>
              <ToggleList>
                <Toggle
                  of={visibility}
                  name="enableRequestBody"
                  update={update}
                  label="Record request bodies"
                />
                <Toggle
                  of={visibility}
                  name="enableRequestBodyMap"
                  update={update}
                  label="Record request bodies as JSON"
                />
                <Toggle
                  of={visibility}
                  name="enableResponseBody"
                  update={update}
                  label="Record response bodies"
                />
                <Toggle
                  of={visibility}
                  name="enableResponseBodyMap"
                  update={update}
                  label="Record response bodies as JSON"
                />
              </ToggleList>
              <VisibilityHeaders value={visibility} update={update} />
            </>
          )}
        </VisibilityRow>
      </OptionList>

      {(global || mode !== CoreP.Service_Spec_Mode.GRPC) && (
        <TransportCard
          holder={value}
          scope={scope}
          noHTTP2={mode === CoreP.Service_Spec_Mode.GRPC}
          update={update}
        >
          {global && (
            <Toggle
              of={value}
              name="enableRequestBuffering"
              update={update}
              label="Buffer request bodies"
              description="Read the whole body first, so that Policies and body checks can use it."
            />
          )}
        </TransportCard>
      )}
    </>
  );
};
