
## 🪴 Groot

A workspace-first runtime layer that keeps your system clean.

Groot is a lightweight control plane that makes environments first-class citizens.

Instead of installing tools globally and allowing them to mutate your system state, Groot scopes toolchains and execution to isolated workspaces.

Delete a workspace → everything related to it is gone.


## Principles

    •	All state lives under a single root directory: ~/.groot
	•	Each workspace has its own isolated $HOME
	•	Toolchains are installed into a shared store
	•	Processes launched via Groot run inside a workspace context
	•	No global mutation of your system

## Runtime Layout

```bash
~/.groot/
  store/          # installed toolchains & binaries
  workspaces/
    acme/
      manifest.yaml
      home/       # workspace-scoped $HOME
      state/
      project/
      logs/
```

## Architecture Overview

```mermaid
flowchart TD
    U[User] --> CLI[groot CLI]
    CLI --> APP[App Runtime Core]

    APP -->|create/read/update| WS[Workspace Folder]
    APP -->|spawn shell| SH[Shell Process]
    APP -->|start services| EX[Execution Backend]

    WS --> MANIFEST[manifest.json<br/>desired state]
    WS --> HOME[home/<br/>isolated $HOME]
    WS --> STATE[state/<br/>runtime metadata]
    WS --> LOGS[logs/]

    APP --> ROOT[~/.groot]
    ROOT --> TOOLCHAINS[toolchains/]
    ROOT --> BIN[bin/]
    ROOT --> CACHE[cache/]
```