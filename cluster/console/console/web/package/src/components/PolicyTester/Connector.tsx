import React, { useId } from "react";

interface AnimatedConnectorProps {
  dotSize?: number;
  dotSpacing?: number;
  color?: string;
  speed?: number;
  className?: string;
}

const AnimatedConnector: React.FC<AnimatedConnectorProps> = ({
  dotSize = 3,
  dotSpacing = 14,
  color = "#94a3b8",
  speed = 0.8,
  className = "w-full",
}) => {
  const uid = useId().replace(/:/g, "");
  const anim = `flow-${uid}`;

  return (
    <div className={`flex items-center ${className}`}>
      <svg width="100%" height={dotSize * 3} xmlns="http://www.w3.org/2000/svg">
        <style>{`
          @keyframes ${anim} {
            0%   { stroke-dashoffset: 0; }
            100% { stroke-dashoffset: -${dotSpacing}; }
          }
          .c-${uid} {
            animation: ${anim} ${speed}s linear infinite;
          }
        `}</style>
        <line
          x1="0"
          y1="50%"
          x2="100%"
          y2="50%"
          stroke={color}
          strokeWidth={dotSize}
          strokeLinecap="round"
          strokeDasharray={`1 ${dotSpacing}`}
          className={`c-${uid}`}
        />
      </svg>
    </div>
  );
};

export default AnimatedConnector;
