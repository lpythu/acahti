export function onboardNote(rootURL: string, login: string, password: string) {
  const root = rootURL.replace(/\/$/, "")
  const lines = [
    `Give this to Cursor, Codex, or any coding agent.`,
    ``,
    `Install ${root}/skill.md`,
    `Login    ${root}/login`,
    `User     ${login}`,
  ]
  if (password) lines.push(`Password ${password}`)
  return lines.join("\n")
}
