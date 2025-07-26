import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { Driver } from "./gen/flowstate/v1/services_connect";

export type DriverClient = ReturnType<typeof createDriverClient>;

export function createDriverClient(baseUrl: string) {
  const transport = createConnectTransport({ baseUrl,  });

  return createClient(Driver, transport);
}
