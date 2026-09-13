export function AcahtiMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 64 64"
      className={className}
      role="img"
      aria-label="Acahti"
    >
      <path
        fill="currentColor"
        d="M32 4.5 56.25 18.5v27L32 59.5 7.75 45.5v-27L32 4.5z"
      />
      <path
        fill="none"
        stroke="#fafafa"
        strokeWidth="5.2"
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M19 43 32 21.5 45 43"
      />
      <circle cx="32" cy="36.5" r="3.15" fill="#0d9488" />
    </svg>
  )
}

export function AcahtiWordmark({ className }: { className?: string }) {
  return (
    <span className={className}>
      <AcahtiMark className="size-6 text-foreground" />
      <span className="font-semibold tracking-tight">Acahti</span>
    </span>
  )
}
