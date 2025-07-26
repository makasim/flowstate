# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a React frontend for the flowstatesrv project, built with:
- React 18
- TypeScript
- Vite
- TailwindCSS
- Connect RPC (for gRPC-web communication)
- Radix UI components

The UI is a dashboard for monitoring and inspecting state transitions in the flowstate system.

## Development Commands

```bash
# Install dependencies
pnpm install

# Start development server
pnpm dev

# Build for production
pnpm build

# Lint code
pnpm lint

# Preview production build
pnpm preview
```

## Architecture

### Frontend Structure

- **API Client**: The application connects to a flowstatesrv backend using Connect RPC. The API client is created in `src/api.ts` and provided through React context.

- **Main Flow**:
  1. User enters a server URL and connects to it
  2. The application streams state changes using the `watchStates` API
  3. State data is displayed in a table with detailed views available in dialogs

- **Components**:
  - Uses a combination of custom components and Radix UI primitives
  - DataTable component using TanStack Table for displaying state information
  - Dialog components for showing detailed state information

### Backend Integration

- The UI is embedded in the Go backend using Go's embed package
- The frontend build outputs to the `public/` directory, which is then embedded in the Go binary
- This allows the server to serve the frontend as static files directly from the binary

## Important Implementation Details

1. **Server Connection**:
   - The UI stores previously connected servers in local storage
   - Connection to the server is established via Connect RPC

2. **State Streaming**:
   - The UI uses the `watchStates` API to receive state updates in real-time
   - New states are prepended to the existing list

3. **Annotations and Labels**:
   - States have associated annotations and labels displayed as badges
   - Special annotations containing "data:" prefix can be expanded to view detailed information

## Build and Deploy

The application is built as part of the main flowstatesrv build process:

1. The React app is built using Vite
2. The built files are embedded in the Go binary
3. The binary is packaged into a Docker container
4. The container can be pushed to the registry using the Taskfile

## Environment Variables

- `FLOWSTATESRV_ADDR`: Address for the server to listen on (default: 127.0.0.1:8080)
- `CORS_ENABLED`: Set to "true" to enable CORS support