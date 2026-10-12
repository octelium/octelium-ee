import { describeRpcError } from "@/utils/rpcError";

const RpcError = (props: { error: unknown }) => {
  const { message, detail } = describeRpcError(props.error);

  return (
    <div className="flex flex-col gap-2 text-micro font-normal leading-4 text-slate-500">
      <p>{message}</p>
      {detail && (
        <details>
          <summary className="cursor-pointer">Details</summary>
          <p className="mt-2 break-words font-mono">{detail}</p>
        </details>
      )}
    </div>
  );
};

export default RpcError;
