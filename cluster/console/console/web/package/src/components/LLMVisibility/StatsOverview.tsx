import { Stats } from "@/apis/visibilityv1/llm/vllmv1";
import {
  Ban,
  Braces,
  Coins,
  DatabaseZap,
  Gauge,
  Hourglass,
  Radio,
  ServerCrash,
  ShieldAlert,
  Sparkles,
  Timer,
  Wrench,
} from "lucide-react";
import { StatCard, StatGrid } from "./Primitives";
import {
  cacheHitRate,
  formatMs,
  formatPercent,
  formatTokens,
  num,
  ratio,
} from "./utils";

const StatsOverview = (props: { stats?: Stats }) => {
  const stats = props.stats;
  const requests = stats?.requests;
  const total = num(requests?.total);
  const denied = num(requests?.denied);
  const failed = num(requests?.failed);
  const cacheServed =
    num(requests?.cacheExactHit) + num(requests?.cacheSemanticHit);

  return (
    <StatGrid>
      <StatCard
        label="Requests"
        value={total.toLocaleString()}
        hint={`${num(requests?.streamed).toLocaleString()} streamed`}
        icon={Sparkles}
      />
      <StatCard
        label="Denied"
        value={denied.toLocaleString()}
        hint={formatPercent(ratio(denied, total))}
        tone={denied > 0 ? "danger" : "default"}
        icon={Ban}
      />
      <StatCard
        label="Failed"
        value={failed.toLocaleString()}
        hint={`${num(requests?.serverError).toLocaleString()} upstream errors`}
        tone={failed > 0 ? "warning" : "default"}
        icon={ServerCrash}
      />
      <StatCard
        label="Total tokens"
        value={formatTokens(num(stats?.tokens?.total))}
        hint={`${formatTokens(num(stats?.tokens?.input))} in · ${formatTokens(num(stats?.tokens?.output))} out`}
        icon={Coins}
      />
      <StatCard
        label="Reasoning tokens"
        value={formatTokens(num(stats?.tokens?.reasoningOutput))}
        hint={`${num(requests?.withManagedReasoning).toLocaleString()} managed requests`}
        icon={Braces}
      />
      <StatCard
        label="Discarded tokens"
        value={formatTokens(num(stats?.tokens?.discarded))}
        hint={`${num(requests?.discardedInference).toLocaleString()} discarded inferences`}
        tone={num(stats?.tokens?.discarded) > 0 ? "warning" : "default"}
        icon={DatabaseZap}
      />
      <StatCard
        label="Guardrail denials"
        value={num(requests?.guardrailDenied).toLocaleString()}
        hint={`${num(requests?.guardrailModified).toLocaleString()} modified · ${num(requests?.guardrailError).toLocaleString()} errors`}
        tone={num(requests?.guardrailDenied) > 0 ? "danger" : "default"}
        icon={ShieldAlert}
      />
      <StatCard
        label="Cache hit rate"
        value={formatPercent(cacheHitRate(stats))}
        hint={`${cacheServed.toLocaleString()} served from cache`}
        tone={cacheServed > 0 ? "positive" : "default"}
        icon={DatabaseZap}
      />
      <StatCard
        label="P95 latency"
        value={formatMs(stats?.latency?.p95Ms ?? 0)}
        hint={`avg ${formatMs(stats?.latency?.avgMs ?? 0)}`}
        icon={Timer}
      />
      <StatCard
        label="P95 TTFT"
        value={formatMs(stats?.timeToFirstToken?.p95Ms ?? 0)}
        hint={`avg ${formatMs(stats?.timeToFirstToken?.avgMs ?? 0)}`}
        icon={Hourglass}
      />
      <StatCard
        label="Tool calls"
        value={num(stats?.toolCalls).toLocaleString()}
        hint={`${num(stats?.distinctToolsCalled).toLocaleString()} distinct tools`}
        icon={Wrench}
      />
      <StatCard
        label="Token quota denials"
        value={num(requests?.tokenRateLimitDenied).toLocaleString()}
        hint={`${num(requests?.tokenRateLimitAllowed).toLocaleString()} allowed`}
        tone={num(requests?.tokenRateLimitDenied) > 0 ? "warning" : "default"}
        icon={Gauge}
      />
      <StatCard
        label="Stream events"
        value={formatTokens(num(stats?.streamEvents))}
        hint={`${num(requests?.streamed).toLocaleString()} streamed requests`}
        icon={Radio}
      />
    </StatGrid>
  );
};

export default StatsOverview;
