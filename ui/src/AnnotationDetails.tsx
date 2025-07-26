import { useContext, useEffect, useState } from "react";
import { ApiContext } from "./ApiContext";
import {Command, GetDataCommand, Data, DataRef, State, StateCtx, StateCtxRef} from "./gen/flowstate/v1/messages_pb";

export const AnnotationDetails = ({ alias, state }: { alias: string; state: State }) => {
  const [info, setInfo] = useState<Command | null>(null);
  const client = useContext(ApiContext);

  useEffect(() => {
    if (!client) return;

    const command = new Command({
      stateCtxs: [
        new StateCtx({current: state, committed: state})
      ],
      datas: [
        new Data()
      ],
      getData: new GetDataCommand({
        alias: alias,
        stateRef: new StateCtxRef({ idx: BigInt(0) }),
        dataRef: new DataRef({ idx: BigInt(0) })
      })
    });
    
    client.getData(command).then(setInfo);
  }, [client]);

  if (!info) return "Loading...";

  return (
    <>
      {info.datas.map((d) => {
        if (d.binary) return <span>{d.b}</span>;

        try {
          return (
            <pre className="text-left">
              {JSON.stringify(JSON.parse(d.b), null, 2)}
            </pre>
          );
        } catch {
          return <span>{d.b}</span>;
        }
      })}
    </>
  );
};
