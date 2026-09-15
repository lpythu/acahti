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
