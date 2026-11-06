import * as CoreP from "@/apis/corev1/corev1";
import {
  Callout,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { Button } from "@mantine/core";
import { LockKeyhole } from "lucide-react";
import {
  joinSummary,
  Note,
  OptionList,
  OptionRow,
  PEMList,
  plural,
  SecretField,
  Toggle,
  Update,
} from "./fields";
import { Config, parseUpstreamURL } from "./model";
import { UpstreamTLS } from "./modes";

const clientCertificateSecret = (tls: CoreP.Service_Spec_Config_TLS) =>
  tls.clientCertificate?.type.oneofKind === "fromSecret"
    ? tls.clientCertificate.type.fromSecret
    : "";

const summary = (tls: CoreP.Service_Spec_Config_TLS) =>
  joinSummary(
    tls.insecureSkipVerify && "Verification skipped",
    tls.trustedCAs.length > 0 && plural(tls.trustedCAs.length, "trusted CA"),
    tls.appendToSystemPool && "System CAs included",
    clientCertificateSecret(tls) &&
      `Client certificate ${clientCertificateSecret(tls)}`,
  ) || "System trusted CAs";

const plaintextUpstream = (config: Config) => {
  const type = config.upstream?.type;
  const url =
    type?.oneofKind === "url"
      ? type.url
      : type?.oneofKind === "loadbalance"
        ? type.loadbalance.endpoints[0]?.url
        : undefined;
  const scheme = url ? parseUpstreamURL(url)?.scheme : undefined;
  return !!scheme && ["http", "h2c", "ws", "grpc"].includes(scheme);
};

const LegacyCertificate = (props: { config: Config; update: Update }) => {
  const legacy = props.config.clientCertificate;
  if (!legacy) return null;
  return (
    <Callout title="This Configuration uses the older clientCertificate field">
      <span className="flex flex-col items-start gap-2">
        <span>
          It takes priority over every other setting here. Move it to the TLS
          settings to keep the certificate and manage it in one place.
        </span>
        <Button
          type="button"
          variant="default"
          size="compact-xs"
          onClick={() => {
            const tls =
              props.config.tls ?? CoreP.Service_Spec_Config_TLS.create();
            tls.trustedCAs = [...tls.trustedCAs, ...legacy.trustedCAs];
            if (legacy.type.oneofKind === "fromSecret") {
              tls.clientCertificate =
                CoreP.Service_Spec_Config_TLS_ClientCertificate.create({
                  type: {
                    oneofKind: "fromSecret",
                    fromSecret: legacy.type.fromSecret,
                  },
                });
            }
            props.config.tls = tls;
            props.config.clientCertificate = undefined;
            props.update();
          }}
        >
          Move to TLS settings
        </Button>
      </span>
    </Callout>
  );
};

export const TLSRow = (props: {
  config: Config;
  support: UpstreamTLS;
  update: Update;
}) => {
  const { config, update } = props;
  const tls = config.tls;

  return (
    <OptionRow
      icon={LockKeyhole}
      title="Upstream TLS"
      description="Trusted CAs and the mTLS client certificate used with the upstream."
      summary={tls && summary(tls)}
      isSet={!!tls}
      onAdd={() => {
        config.tls = CoreP.Service_Spec_Config_TLS.create();
        update();
      }}
      onRemove={() => {
        config.tls = undefined;
        update();
      }}
    >
      {tls && (
        <>
          {plaintextUpstream(config) && (
            <Note>
              These settings apply to TLS upstreams only, and this upstream URL
              uses a plaintext scheme.
            </Note>
          )}
          {props.support === "full" && (
            <ToggleList>
              <Toggle
                of={tls}
                name="insecureSkipVerify"
                update={update}
                label="Skip certificate verification"
                description="Accept any upstream certificate. Use this only for troubleshooting."
              />
            </ToggleList>
          )}
          {!tls.insecureSkipVerify && (
            <>
              <PEMList
                label="Trusted CAs"
                description="PEM-encoded root CAs. They replace the system CAs unless you keep them below."
                values={tls.trustedCAs}
                update={update}
              />
              {tls.trustedCAs.length > 0 && (
                <ToggleList>
                  <Toggle
                    of={tls}
                    name="appendToSystemPool"
                    update={update}
                    label="Keep the system CAs"
                    description="Trust these CAs in addition to the system CAs."
                  />
                </ToggleList>
              )}
            </>
          )}
          <SecretField
            label="Client certificate"
            description="Secret with the PEM certificate chain and private key, for upstreams that require mTLS."
            value={clientCertificateSecret(tls)}
            onChange={(name) => {
              tls.clientCertificate = name
                ? CoreP.Service_Spec_Config_TLS_ClientCertificate.create({
                    type: { oneofKind: "fromSecret", fromSecret: name },
                  })
                : undefined;
              update();
            }}
          />
        </>
      )}
    </OptionRow>
  );
};

export const TLSSection = (props: {
  config: Config;
  support: UpstreamTLS;
  update: Update;
}) => (
  <div className="flex min-w-0 flex-col gap-3">
    <LegacyCertificate config={props.config} update={props.update} />
    <OptionList title="Connection security">
      <TLSRow
        config={props.config}
        support={props.support}
        update={props.update}
      />
    </OptionList>
  </div>
);
