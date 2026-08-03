import { useId } from "react";

interface AnimatedConnectorProps {
  dotSize?: number;
  dotSpacing?: number;
  color?: string;
  speed?: number;
  className?: string;
  orientation?: "horizontal" | "vertical";
}

const AnimatedConnector = ({
  dotSize = 3,
  dotSpacing = 14,
  color = "#64748b",
  speed = 1.4,
  className = "w-full",
  orientation = "horizontal",
}: AnimatedConnectorProps) => {
  const uid = useId().replace(/:/g, "");
  const animationName = `connector-flow-${uid}`;
  const horizontal = orientation === "horizontal";

  return (
    <div
      className={`flex items-center justify-center ${className}`}
      aria-hidden="true"
    >
      <svg
        width={horizontal ? "100%" : dotSize * 6}
        height={horizontal ? dotSize * 6 : "100%"}
        xmlns="http://www.w3.org/2000/svg"
        focusable="false"
      >
        <style>{`
          @keyframes ${animationName} {
            from { stroke-dashoffset: 0; }
            to { stroke-dashoffset: -${dotSpacing}; }
          }
          .connector-${uid} {
            animation: ${animationName} ${speed}s linear infinite;
          }
          @media (prefers-reduced-motion: reduce) {
            .connector-${uid} { animation: none; }
          }
        `}</style>
        <line
          x1={horizontal ? dotSize : "50%"}
          y1={horizontal ? "50%" : dotSize}
          x2={horizontal ? `calc(100% - ${dotSize}px)` : "50%"}
          y2={horizontal ? "50%" : `calc(100% - ${dotSize}px)`}
          stroke="#e2e8f0"
          strokeWidth={1.5}
          strokeLinecap="round"
        />
        <line
          x1={horizontal ? dotSize : "50%"}
          y1={horizontal ? "50%" : dotSize}
          x2={horizontal ? `calc(100% - ${dotSize}px)` : "50%"}
          y2={horizontal ? "50%" : `calc(100% - ${dotSize}px)`}
          stroke={color}
          strokeWidth={dotSize}
          strokeLinecap="round"
          strokeDasharray={`1 ${dotSpacing - 1}`}
          className={`connector-${uid}`}
        />
        <circle
          cx={horizontal ? dotSize : "50%"}
          cy={horizontal ? "50%" : dotSize}
          r={dotSize}
          fill="#ffffff"
          stroke={color}
          strokeWidth={1.5}
        />
        <circle
          cx={horizontal ? `calc(100% - ${dotSize}px)` : "50%"}
          cy={horizontal ? "50%" : `calc(100% - ${dotSize}px)`}
          r={dotSize + 1}
          fill={color}
          stroke="#ffffff"
          strokeWidth={2}
        />
      </svg>
    </div>
  );
};

export default AnimatedConnector;
