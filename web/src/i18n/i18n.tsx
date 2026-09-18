import { Fragment, createContext, useContext, useMemo, useState, type ReactNode } from "react"

import { messages, type Locale, type MessageKey } from "./messages"

type Vars = Record<string, string | number>
type RichVars = Record<string, ReactNode>

function format(s: string, vars?: Vars) {
  if (!vars) return s
  let out = s
  for (const [k, v] of Object.entries(vars)) {
    out = out.replaceAll(`{${k}}`, String(v))
  }
  return out
}

function formatRich(s: string, vars?: RichVars): ReactNode {
  if (!vars) return s
  const parts = s.split(/(\{[A-Za-z0-9_]+\})/g)
  if (parts.length === 1) return s
  return parts.map((part, i) => {
    const m = /^\{([A-Za-z0-9_]+)\}$/.exec(part)
    if (!m) return part
    return <Fragment key={i}>{vars[m[1]] ?? part}</Fragment>
  })
}

const I18n = createContext<{
  locale: Locale
  t: (key: MessageKey, vars?: Vars) => string
  tr: (key: MessageKey, vars?: RichVars) => ReactNode
  setLocale: (l: Locale) => void
}>({
  locale: "en",
  t: (key, vars) => format(messages.en[key], vars),
  tr: (key, vars) => formatRich(messages.en[key], vars),
  setLocale: () => {},
})

function detect(): Locale {
  const lang = navigator.language.toLowerCase()
  return lang.startsWith("zh") ? "zh" : "en"
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(detect)
  const value = useMemo(
    () => ({
      locale,
      setLocale,
      t: (key: MessageKey, vars?: Vars) => format(messages[locale][key], vars),
      tr: (key: MessageKey, vars?: RichVars) => formatRich(messages[locale][key], vars),
    }),
    [locale],
  )
  return <I18n.Provider value={value}>{children}</I18n.Provider>
}

export function useT() {
  return useContext(I18n).t
}

export function useTr() {
  return useContext(I18n).tr
}

export function useLocale() {
  return useContext(I18n).locale
}
