import React, { useId } from "react";

interface AnimatedConnectorProps {
  dotSize?: number;
  dotSpacing?: number;
  color?: string;
  speed?: number;
  className?: string;
}

export const AnimatedConnector: React.FC<AnimatedConnectorProps> = ({
  dotSize = 4,
  dotSpacing = 16,
  color = "#94a3b8",
  speed = 1,
  className = "w-full",
}) => {
  const uniqueId = useId().replace(/:/g, "");
  const animationName = `flow-${uniqueId}`;

  return (
    <div className={`flex items-center ${className}`}>
      <svg width="100%" height={dotSize} xmlns="http://www.w3.org/2000/svg">
        <style>
          {`
            @keyframes ${animationName} {
              0% { stroke-dashoffset: 0; }
              /* Negative offset moves the pattern from left to right */
              100% { stroke-dashoffset: -${dotSpacing}; } 
            }
            .animate-connector-${uniqueId} {
              animation: ${animationName} ${speed}s linear infinite;
            }
          `}
        </style>

        <line
          x1="0"
          y1={dotSize / 2}
          x2="100%"
          y2={dotSize / 2}
          stroke={color}
          strokeWidth={dotSize}
          strokeLinecap="round"
          strokeDasharray={`0 ${dotSpacing}`}
          className={`animate-connector-${uniqueId}`}
        />
      </svg>
    </div>
  );
};

export default AnimatedConnector;
