export type Section =
  | "board"
  | "repos"
  | "pipelines"
  | "packages"
  | "admin"

export function sectionOf(path: string): Section {
  if (path.startsWith("/admin")) return "admin"
  if (path.startsWith("/packages")) return "packages"
  if (path.startsWith("/pipelines")) return "pipelines"
  if (path.startsWith("/repos")) return "repos"
  if (path.startsWith("/board")) return "board"
  return "board"
}

export function repoBase(path: string): string | null {
  const m = path.match(/^\/repos\/([^/]+)\/([^/]+)/)
  return m ? `/repos/${m[1]}/${m[2]}` : null
}

export function packageHref(kind: string, name: string) {
  return `/packages/${kind}/${name}`
}

export function parsePackagePath(path: string): { kind: string; name: string } | null {
  const m = path.match(/^\/packages\/([^/]+)\/(.+)$/)
  return m ? { kind: m[1], name: m[2] } : null
}

export function pipelineHref(owner: string, name: string, n: number | string) {
  return `/repos/${owner}/${name}/pipelines/${n}`
}

export function parsePipelinePath(path: string): { owner: string; name: string; number?: string } | null {
  const underRepo = path.match(/^\/repos\/([^/]+)\/([^/]+)\/pipelines(?:\/([^/]+))?$/)
  if (underRepo) return { owner: underRepo[1], name: underRepo[2], number: underRepo[3] }
  const legacy = path.match(/^\/pipelines\/([^/]+)\/([^/]+)(?:\/([^/]+))?$/)
  return legacy ? { owner: legacy[1], name: legacy[2], number: legacy[3] } : null
}
