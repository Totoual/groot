# Groot Agent Skeleton

This package is the first thin agent layer planned for Groot.

Current state:

- defines the v0 agent boundaries
- defines and implements the CLI + JSON contract client the agent will call
- defines the first supported intent kinds
- implements the first thin service methods for:
  - inspect runtime status
  - open and set up an existing repo
- does not implement natural-language intent parsing yet

The intended flow is:

1. parse user intent
2. inspect project/runtime state via Groot CLI + JSON
3. choose the next Groot action
4. execute Groot actions
5. report progress back to the user

This package should stay thin.

It should not become a generic agent framework.
