import { createContext } from "react";
import { DriverClient } from "./api";

export const ApiContext = createContext<DriverClient | null>(null);
