import { Stats } from "@/apis/visibilityv1/llm/vllmv1";
import { Trend } from "@/components/Dashboard/components";
import { formatCount } from "@/utils/format";
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

const StatsOverview = (props: {
  stats?: Stats;
  previous?: Stats;
  rangeLabel: string;
}) => {
  const stats = props.stats;
  const requests = stats?.requests;
  const previous = props.previous;
  const trend = (cur: number, prev: number, upIsGood: boolean) =>
    previous ? (
      <Trend
        cur={cur}
        prev={prev}
        upIsGood={upIsGood}
        rangeLabel={props.rangeLabel}
      />
    ) : undefined;
  const total = num(requests?.total);
  const denied = num(requests?.denied);
  const failed = num(requests?.failed);
  const cacheServed =
    num(requests?.cacheExactHit) + num(requests?.cacheSemanticHit);

  return (
    <StatGrid>
      <StatCard
        label="Requests"
        value={formatCount(total)}
        trend={trend(total, num(previous?.requests?.total), true)}
        hint={`${formatCount(num(requests?.streamed))} streamed`}
        icon={Sparkles}
      />
      <StatCard
        label="Denied"
        value={formatCount(denied)}
        trend={trend(denied, num(previous?.requests?.denied), false)}
        hint={formatPercent(ratio(denied, total))}
        tone={denied > 0 ? "danger" : "default"}
        icon={Ban}
      />
      <StatCard
        label="Failed"
        value={formatCount(failed)}
        trend={trend(failed, num(previous?.requests?.failed), false)}
        hint={`${formatCount(num(requests?.serverError))} upstream errors`}
        tone={failed > 0 ? "warning" : "default"}
        icon={ServerCrash}
      />
      <StatCard
        label="Total tokens"
        value={formatTokens(num(stats?.tokens?.total))}
        trend={trend(
          num(stats?.tokens?.total),
          num(previous?.tokens?.total),
          true,
        )}
        hint={`${formatTokens(num(stats?.tokens?.input))} in · ${formatTokens(num(stats?.tokens?.output))} out`}
        icon={Coins}
      />
      <StatCard
        label="Reasoning tokens"
        value={formatTokens(num(stats?.tokens?.reasoningOutput))}
        hint={`${formatCount(num(requests?.withManagedReasoning))} managed requests`}
        icon={Braces}
      />
      <StatCard
        label="Discarded tokens"
        value={formatTokens(num(stats?.tokens?.discarded))}
        hint={`${formatCount(num(requests?.discardedInference))} discarded inferences`}
        tone={num(stats?.tokens?.discarded) > 0 ? "warning" : "default"}
        icon={DatabaseZap}
      />
      <StatCard
        label="Guardrail denials"
        value={formatCount(num(requests?.guardrailDenied))}
        trend={trend(
          num(requests?.guardrailDenied),
          num(previous?.requests?.guardrailDenied),
          false,
        )}
        hint={`${formatCount(num(requests?.guardrailModified))} modified · ${formatCount(num(requests?.guardrailError))} errors`}
        tone={num(requests?.guardrailDenied) > 0 ? "danger" : "default"}
        icon={ShieldAlert}
      />
      <StatCard
        label="Cache hit rate"
        value={formatPercent(cacheHitRate(stats))}
        hint={`${formatCount(cacheServed)} served from cache`}
        tone={cacheServed > 0 ? "positive" : "default"}
        icon={DatabaseZap}
      />
      <StatCard
        label="P95 latency"
        value={formatMs(stats?.latency?.p95Ms ?? 0)}
        trend={trend(
          stats?.latency?.p95Ms ?? 0,
          previous?.latency?.p95Ms ?? 0,
          false,
        )}
        hint={`avg ${formatMs(stats?.latency?.avgMs ?? 0)}`}
        icon={Timer}
      />
      <StatCard
        label="P95 TTFT"
        value={formatMs(stats?.timeToFirstToken?.p95Ms ?? 0)}
        trend={trend(
          stats?.timeToFirstToken?.p95Ms ?? 0,
          previous?.timeToFirstToken?.p95Ms ?? 0,
          false,
        )}
        hint={`avg ${formatMs(stats?.timeToFirstToken?.avgMs ?? 0)}`}
        icon={Hourglass}
      />
      <StatCard
        label="Tool calls"
        value={formatCount(num(stats?.toolCalls))}
        hint={`${formatCount(num(requests?.withToolCalls))} requests with tool calls`}
        icon={Wrench}
      />
      <StatCard
        label="Token quota denials"
        value={formatCount(num(requests?.tokenRateLimitDenied))}
        trend={trend(
          num(requests?.tokenRateLimitDenied),
          num(previous?.requests?.tokenRateLimitDenied),
          false,
        )}
        hint={`${formatCount(num(requests?.tokenRateLimitAllowed))} allowed`}
        tone={num(requests?.tokenRateLimitDenied) > 0 ? "warning" : "default"}
        icon={Gauge}
      />
      <StatCard
        label="Stream events"
        value={formatTokens(num(stats?.streamEvents))}
        hint={`${formatCount(num(requests?.streamed))} streamed requests`}
        icon={Radio}
      />
    </StatGrid>
  );
};

export default StatsOverview;
