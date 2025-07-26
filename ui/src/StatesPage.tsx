import "./App.css";
import { DataTable } from "./components/data-table";
import React, { useEffect, useState } from "react";
import { State } from "./gen/flowstate/v1/messages_pb";

import { ColumnDef } from "@tanstack/react-table";
import { createDriverClient } from "./api";
import { Badge } from "./components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from "./components/ui/dialog";
import { ApiContext } from "./ApiContext";
import { AnnotationDetails } from "./AnnotationDetails";
import { GetStatesCommand, Command } from "./gen/flowstate/v1/messages_pb";

type StateData = {
  id: string;
  stateId: string;
  rev: bigint;
  transition: string;
  annotations: Record<string, string>;
  labels: Record<string, string>;
  state: State;
};

const columns: ColumnDef<StateData>[] = [
  { accessorKey: "stateId", header: "ID" },
  { accessorKey: "rev", header: "REV" },
  { accessorKey: "transition", header: "Transtion" },
  {
    accessorKey: "annotations",
    header: "Annotations",
    cell: ({ row }) =>
      Object.entries(row.original.annotations || {}).map(([key, value]) => (
        <div key={key} className="text-left">
          <Badge variant="outline">
            <span className="text-green-700">{key}:&nbsp;</span>
            <span className="text-purple-700">{String(value)}</span>
          </Badge>
        </div>
      )),
  },
  {
    accessorKey: "data",
    header: "Data",
    cell: ({ row }) =>
      Object.entries(row.original.annotations || {})
        .filter(([key]) => key.startsWith("flowstate.data."))
        .map(([key, value]) => {
          const alias = key.slice(15); // Remove "flowstate.data." prefix
          const [dataId, dataRev] = value.split(":");
          return [alias, dataId, dataRev];
        })
        .map(([alias, dataId, dataRev]) => (
          <Dialog modal key={`${dataId}:${dataRev}`}>
            <DialogTrigger className="text-slate">
              <svg
                className="w-6 h-6 text-white"
                aria-hidden="true"
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke="currentColor"
                  stroke-width="2"
                  d="M21 12c0 1.2-4.03 6-9 6s-9-4.8-9-6c0-1.2 4.03-6 9-6s9 4.8 9 6Z"
                />
                <path
                  stroke="currentColor"
                  stroke-width="2"
                  d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
                />
              </svg>
            </DialogTrigger>

            <DialogContent>
              <DialogTitle className="pb-4 sticky top-0 bg-background">
                flowstate.data.{alias}: "{dataId}:{dataRev}"
              </DialogTitle>
              <DialogDescription>
                <AnnotationDetails alias={alias} state={row.original.state} />
              </DialogDescription>
            </DialogContent>
          </Dialog>
        )),
  },
  {
    accessorKey: "labels",
    header: "Labels",
    cell: ({ row }) =>
      Object.entries(row.original.labels || {}).map(([key, value]) => (
        <div key={key} className="text-left">
          <Badge variant="outline">
            <span className="text-green-700">{key}:&nbsp;</span>
            <span className="text-purple-700">{String(value)}</span>
          </Badge>
        </div>
      )),
  },
  {
    accessorKey: "state",
    header: "State",
    cell: ({ row }) => (
      <Dialog modal>
        <DialogTrigger className="text-slate-100">Show State</DialogTrigger>
        <DialogContent>
          <DialogTitle>State: {row.original.state.id}</DialogTitle>
          <DialogDescription>
            <pre className="text-left">
              {JSON.stringify(row.original.state.toJson(), null, 2)}
            </pre>
          </DialogDescription>
        </DialogContent>
      </Dialog>
    ),
  },
];

type DriverClient = ReturnType<typeof createDriverClient>;

export const StatesPage = () => {
  const [states, setStates] = useState<State[]>([]);
  const client = React.useContext(ApiContext);

  useEffect(() => {
    if (!client) return;

    const abortController = new AbortController();

    listenToStates(client, abortController.signal).catch((error) =>
      console.log("Listening error", error)
    );

    return () => {
      abortController.abort();
    };
  }, [client]);

  const listenToStates = async (client: DriverClient, signal: AbortSignal) => {
    let sinceRev = BigInt(0);
    let intervalId: NodeJS.Timeout;

    const pollStates = async () => {
      if (signal.aborted) return;

      const getStatesCommand = new GetStatesCommand({
        limit: BigInt(100),
        latestOnly: false,
        sinceRev: sinceRev,
      });

      const anyCommand = new Command({
        getStates: getStatesCommand
      });

      try {
        const anyCommandResp = await client.getStates(anyCommand, { signal });
        const getStatesCommand = anyCommandResp.getStates
        if (getStatesCommand && getStatesCommand.result) {
          const getStatesResult = getStatesCommand.result;
          if (getStatesResult.states.length > 0) {
            setStates(currentStates => {
              const newStates = [...currentStates, ...getStatesResult.states];
              return newStates.sort((a, b) => Number(b.rev - a.rev));
            });

            const maxRev = getStatesResult.states.reduce((max, state) =>
              state.rev > max ? state.rev : max, sinceRev);
            sinceRev = maxRev;
          }
        }
      } catch (error) {
        if (!signal.aborted) {
          console.log("Command error", error);
        }
      }
    };

    await pollStates();
    
    intervalId = setInterval(pollStates, 1000);
    
    signal.addEventListener('abort', () => {
      clearInterval(intervalId);
    });
  };

  function formatTransition({ from, to }: { from: string; to: string }) {
    if (!from && !to) return "";
    if (!from) return ` -> ${to}`;
    if (!to) return `${from} -> `;
    if (from === to) return from;

    return `${from} -> ${to}`;
  }

  const data = states.map((state) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { id, rev, transition, annotations, labels } = state.toJson() as any;
    return {
      id: `${id}#${rev}`,
      stateId: id,
      rev,
      transition: transition ? formatTransition(transition) : "",
      annotations,
      labels,
      state,
    };
  });

  return (
    <div className="container mx-auto py-10">
      <DataTable columns={columns} data={data} />
    </div>
  );
};
