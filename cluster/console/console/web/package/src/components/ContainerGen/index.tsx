import { Title } from "@mantine/core";
import * as React from "react";

const ContainerGen = (props: {
  children?: React.ReactNode;
  title?: React.ReactNode;
}) => {
  return (
    <div className="w-full border-[1px] rounded-md border-gray-300 shadow-sm p-4">
      {props.title && (
        <Title className="mb-6" order={4}>
          {props.title}
        </Title>
      )}
      <div className="w-full">{props.children}</div>
    </div>
  );
};

export default ContainerGen;
