const EXT: Record<string, string> = {
  bash: "bash",
  c: "c",
  css: "css",
  go: "go",
  h: "c",
  html: "html",
  ini: "ini",
  java: "java",
  js: "javascript",
  json: "json",
  jsx: "jsx",
  lock: "json",
  md: "markdown",
  mdx: "markdown",
  py: "python",
  rs: "rust",
  sh: "bash",
  sql: "sql",
  toml: "toml",
  ts: "typescript",
  tsx: "tsx",
  yaml: "yaml",
  yml: "yaml",
}

export function langOf(name: string) {
  const base = name.split("/").pop() || name
  if (base === "Dockerfile" || base === "dockerfile") return "docker"
  if (base === "Makefile") return "make"
  const ext = base.includes(".") ? base.split(".").pop() || "" : ""
  return EXT[ext.toLowerCase()] || "text"
}
