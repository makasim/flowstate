import { useMemo, useState } from "react";
import "./App.css";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { useLocalStorage } from "./hooks/useLocalStorage";
import { StatesPage } from "./StatesPage";
import { ApiContext } from "./ApiContext";
import { createDriverClient } from "./api";
import { ServerIndicator } from "./components/ServerIndicator";

export default function App() {
  const [servers, setServers] = useLocalStorage<string[]>("servers", []);
  const [selectedServer, setSelectedServer] = useLocalStorage<string>("selectedServer", "");
  const [apiURL, setApiURL] = useState(selectedServer);
  const [choosenServer, setChoosenServer] = useState(selectedServer || servers[0] || "");

  const client = useMemo(() => apiURL ? createDriverClient(apiURL) : null, [apiURL]);

  const handleDisconnect = () => {
    setSelectedServer("");
    setApiURL("");
  };

  if (client) {
    return (
      <ApiContext.Provider value={client}>
        <ServerIndicator serverUrl={apiURL} onDisconnect={handleDisconnect} />
        <StatesPage />
      </ApiContext.Provider>
    );
  }

  return (
    <div className="container mx-auto py-10">
      <form
        className="flex w-full max-w-sm items-center space-x-2"
        onSubmit={(e) => {
          e.preventDefault();
          if (!servers.includes(choosenServer)) {
            setServers([choosenServer, ...servers]);
          }
          setSelectedServer(choosenServer);
          setApiURL(choosenServer);
        }}
      >
        <Input
          placeholder="https://flowstate.makasim.com"
          value={choosenServer}
          onChange={(e) => setChoosenServer(e.target.value)}
        />
        <Button type="submit">Subscribe</Button>
      </form>

      <div className="mt-10">
        <h1 className="text-2xl font-bold">Servers</h1>

        <ul>
          {servers.map((server) => (
            <li
              key={server}
              className="flex w-full max-w-sm items-center space-x-2"
            >
              <Button
                onClick={() => setServers(servers.filter((s) => s !== server))}
              >
                x
              </Button>
              <a
                href={server}
                onClick={(e) => {
                  e.preventDefault();
                  setChoosenServer(server);
                  setSelectedServer(server);
                  setApiURL(server);
                }}
              >
                {server}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
