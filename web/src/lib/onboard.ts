export const PLUGIN_REPO = "https://github.com/lpythu/acahti-plugin"

function rootOf(rootURL: string) {
  return rootURL.replace(/\/$/, "")
}

export function onboardYou(rootURL: string, login: string, password: string) {
  const root = rootOf(rootURL)
  const lines = [`Login    ${root}/login`, `User     ${login}`]
  if (password) lines.push(`Password ${password}`)
  return lines.join("\n")
}

export function onboardAgent(rootURL: string) {
  const root = rootOf(rootURL)
  return [`Plugin  ${PLUGIN_REPO}`, `Other   Install ${root}/skill.md`].join("\n")
}
