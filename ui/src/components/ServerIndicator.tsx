import { Button } from "./ui/button";
import { Badge } from "./ui/badge";

interface ServerIndicatorProps {
  serverUrl: string;
  onDisconnect: () => void;
}

export const ServerIndicator = ({ serverUrl, onDisconnect }: ServerIndicatorProps) => {
  return (
    <div className="fixed top-4 right-4 flex items-center space-x-2 bg-background border rounded-lg p-2 shadow-lg">
      <Badge variant="outline" className="text-green-700">
        Connected to: {serverUrl}
      </Badge>
      <Button
        variant="outline"
        size="sm"
        onClick={onDisconnect}
        className="text-xs"
      >
        Disconnect
      </Button>
    </div>
  );
};