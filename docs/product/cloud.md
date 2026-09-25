# Acahti Cloud

Cloud is **managed hosting of the same open-source control plane** — not a feature-gated fork.

## Dual track

| Track | Who | Price |
|---|---|---|
| **Acahti OSS** | Teams that need locality, air-gap, or full ops control | Software free (Apache-2.0) |
| **Acahti Cloud** | Teams that want zero forge ops | Subscription |

Open source and Cloud stay **feature-isomorphic**. Cloud sells uptime, backups, shared CI capacity, and support.

## Pricing (proposed)

| Plan | Price | Includes | Limits (initial) |
|---|---|---|---|
| **Hobby** | $0 | 1 seat, community support | 1 private repo, 500 CI min/mo, 1 GB |
| **Team** | **$29 / seat / month** (≈ $24 annual) | Private repos, MCP, packages, Board | 3 000 CI min / seat / mo, 10 GB / seat; CI overage ≈ $0.008 / min |
| **Business** | **$79 / seat / month** | SSO/SAML (roadmap), audit export, attach dedicated Runner, 99.9% SLA | Higher CI / storage pools |
| **Enterprise** | Contact | VPC / data residency, security reviews, custom pipes | Contract |

### What we meter

- **Seats** = human members
- **CI minutes** and **storage**

We do **not** meter MCP tool calls — that would punish the correct agent-driven workflow.

### Self-host support (optional)

Paid support subscription for OSS operators (per node or per seat). Not a license key; the software remains free.

## Waitlist

Email **hello@acahti.dev** with team size and region preference, or open a GitHub Discussion.

Promise: export Git and migrate back to self-host anytime. Same skill / MCP contract.
