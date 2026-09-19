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
        d="M18 41c3-14 8-21 14-21s11 7 14 21c-5 5-23 5-28 0z"
      />
      <circle cx="32" cy="33" r="3.15" fill="#0d9488" />
    </svg>
  )
}
