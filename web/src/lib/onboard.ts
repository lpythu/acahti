export function onboardNote(rootURL: string, login: string, password: string) {
  const root = rootURL.replace(/\/$/, "")
  return [
    `Give this to Cursor, Codex, or any coding agent.`,
    ``,
    `Install ${root}/skill.md`,
    `Login    ${root}/login`,
    `User     ${login}`,
    `Password ${password}`,
  ].join("\n")
}

export function randomPassword() {
  const b = new Uint8Array(12)
  crypto.getRandomValues(b)
  return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("")
}
