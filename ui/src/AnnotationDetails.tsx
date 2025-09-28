import { useContext, useEffect, useState } from "react";
import { ApiContext } from "./ApiContext";
import {Command, GetDataCommand, State, StateCtx, StateCtxRef} from "./gen/flowstate/v1/messages_pb";

export const AnnotationDetails = ({ alias, state }: { alias: string; state: State }) => {
  const [info, setInfo] = useState<Command | null>(null);
  const client = useContext(ApiContext);

  useEffect(() => {
    if (!client) return;

    const command = new Command({
      stateCtxs: [
        new StateCtx({current: state, committed: state})
      ],
      getData: new GetDataCommand({
        alias: alias,
        stateRef: new StateCtxRef({ idx: BigInt(0) })
      })
    });
    
    client.getData(command).then(setInfo);
  }, [client]);

  if (!info) return "Loading...";

  const data = info.stateCtxs[0].datas[alias];

  if (!data) return "No data found for alias";

  if (data.annotations["binary"]) {
    return <span>{data.blob}</span>;
  }

  try {
    return (
      <pre className="text-left">
        {JSON.stringify(JSON.parse(data.blob), null, 2)}
      </pre>
    );
  } catch {
    return <span>{data.blob}</span>;
  }
};
