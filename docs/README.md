# Documentation

Deep reference for the Mooring Line Management system. For a quick start, see the
[root README](../README.md).

| Doc | What's in it |
|---|---|
| [architecture.md](architecture.md) | System design and build sequencing: stack choices, onboard/shore split, sync model, data model, slice plan. The canonical "why". |
| [authentication.md](authentication.md) | OIDC Backend-for-Frontend auth: login flow, provider/endpoints, every auth env var, groups→permissions, registering the redirect URI, troubleshooting. |
| [oidc-integration-lessons.md](oidc-integration-lessons.md) | Postmortem: why the OIDC integration needed several debugging round trips despite a working reference, and a pre-flight checklist to prevent them next time. |
| [configuration.md](configuration.md) | All environment variables, the onboard vs shore dev topology (ports), and the Make targets. |
