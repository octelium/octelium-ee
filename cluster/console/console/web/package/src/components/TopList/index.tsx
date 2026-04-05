import { formatNumber } from "@/utils";
import {
  getRefNameQueryArgStr,
  getResourcePath,
  getResourceRef,
  printResourceNameWithDisplay,
  Resource,
} from "@/utils/pb";
import { Link } from "react-router-dom";
import { ResourceHoverCard } from "../ResourceList";

const TopListItem = (props: { item: Resource }) => {
  const { item } = props;
  const x = item;
  return (
    <div className="w-full my-2">
      <ResourceHoverCard itemRef={getResourceRef(x)}>
        <Link
          className="font-bold text-slate-600 hover:text-black transition-all duration-700"
          to={getResourcePath(x)}
        >
          {printResourceNameWithDisplay(x)}
        </Link>
      </ResourceHoverCard>
    </div>
  );
};

const TopList = (props: {
  title: string;
  items?: {
    resource: Resource;
    count: number;
  }[];
  to?: string;
}) => {
  const { items } = props;
  const topCount = items?.at(0)?.count ?? 100;
  if (!items || items.length < 1) {
    return <></>;
  }

  return (
    <div>
      <div className="mb-4 font-bold text-xl">{props.title}</div>
      <div className="ml-2">
        {items.map((x) => (
          <div
            className="w-full flex items-center"
            key={x.resource.metadata?.uid}
          >
            <div className="w-full flex text-sm font-bold mb-1 items-center">
              <div className="flex text-black min-w-[100px]">
                <div className="w-full">
                  {/**
                   <Progress.Root radius={`xs`} size="xl" autoContrast>
                    <Progress.Section value={(x.count * 100) / topCount}>
                      <Progress.Label className="text-white">
                        {formatNumber(x.count)}
                      </Progress.Label>
                    </Progress.Section>
                  </Progress.Root>
                   **/}

                  {props.to ? (
                    <Link
                      to={`${props.to}?${getRefNameQueryArgStr(x.resource)}`}
                    >
                      <ProgressBar
                        progress={(x.count * 100) / topCount}
                        label={formatNumber(x.count)}
                        bgColor={`bg-gray-500`}
                        barColor={`bg-gray-900`}
                        height={`h-4`}
                      />
                    </Link>
                  ) : (
                    <ProgressBar
                      progress={(x.count * 100) / topCount}
                      label={formatNumber(x.count)}
                      bgColor={`bg-gray-500`}
                      barColor={`bg-gray-900`}
                      height={`h-4`}
                    />
                  )}
                </div>
              </div>
              <div className="ml-2">
                <TopListItem item={x.resource} />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

interface ProgressBarProps {
  progress: number;
  label?: string;
  barColor?: string;
  bgColor?: string;
  height?: string;
}

const ProgressBar: React.FC<ProgressBarProps> = ({
  progress,
  label,
  barColor = "bg-blue-600",
  bgColor = "bg-gray-200",
  height = "h-6",
}) => {
  const clampedProgress = Math.min(100, Math.max(0, progress));

  return (
    <div
      className={`w-full ${bgColor} rounded-full relative shadow transition-all duration-500 overflow-hidden ${height}`}
    >
      <div
        className={`${barColor} hover:bg-black transition-all duration-500 h-full ease-out`}
        style={{ width: `${clampedProgress}%` }}
      />

      <div className="absolute inset-0 flex items-center justify-center">
        <span className="text-xs font-bold text-white">
          {label || `${clampedProgress}%`}
        </span>
      </div>
    </div>
  );
};

export default TopList;
