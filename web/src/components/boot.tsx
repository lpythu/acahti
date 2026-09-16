import { Component, type ReactNode } from "react"

export function BootScreen({ label = "Acahti" }: { label?: string }) {
  return (
    <div className="bg-background text-muted-foreground flex min-h-svh items-center justify-center text-sm">
      {label}
    </div>
  )
}

export class ErrorBoundary extends Component<{ children: ReactNode }, { err: Error | null }> {
  state: { err: Error | null } = { err: null }

  static getDerivedStateFromError(err: Error) {
    return { err }
  }

  render() {
    if (!this.state.err) return this.props.children
    return (
      <div className="bg-background flex min-h-svh flex-col items-center justify-center gap-3 p-6">
        <p className="text-sm text-destructive">{this.state.err.message}</p>
        <button type="button" className="text-sm underline" onClick={() => window.location.reload()}>
          Reload
        </button>
      </div>
    )
  }
}
