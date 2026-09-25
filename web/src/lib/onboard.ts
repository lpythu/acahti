export const PLUGIN_REPO = "https://github.com/lpythu/acahti-plugin"

function rootOf(rootURL: string) {
  return rootURL.replace(/\/$/, "")
}

export function onboardYou(rootURL: string, login: string, password: string) {
  const root = rootOf(rootURL)
  return [`Login    ${root}/login`, `User     ${login}`, `Password ${password}`].join("\n")
}

export function onboardAgent(rootURL: string, org = "org") {
  const root = rootOf(rootURL)
  return [
    `Install ${root}/skill.md`,
    `Install ${root}/demo.md`,
    "",
    `Run the Acahti demo loop on this instance now. Connect MCP with OAuth. Do not paste tokens. Do not change global git config. Create or reuse a disposable demo-* repo under ${org}, push a trivial branch with a smoke pipeline, wait for checks, open a PR, merge when green (or stop with evidence if no Runner). Reply with repo URL, PR number, and pipeline number. I will only watch the Board.`,
  ].join("\n")
}

export function randomPassword() {
  const b = new Uint8Array(12)
  crypto.getRandomValues(b)
  return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("")
}
