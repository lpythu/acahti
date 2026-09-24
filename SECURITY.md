# Security

Use GitHub private vulnerability reporting for this repository. Do not place
credentials or exploit details in public issues. Include affected versions and a
minimal synthetic reproduction.

The control-plane operator is trusted. Deploy TLS in front of the gateway, isolate
CI execution from control-plane data, back up persistent state, and scope external
pipeline credentials. Acahti does not provide a hostile-code execution sandbox.
MCP authorization and website login are separate sessions.
