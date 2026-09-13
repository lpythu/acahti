import { createContext, useContext, useMemo, useState, type ReactNode } from "react"

import { messages, type Locale, type MessageKey } from "./messages"

type Vars = Record<string, string | number>

function format(s: string, vars?: Vars) {
  if (!vars) return s
  let out = s
  for (const [k, v] of Object.entries(vars)) {
    out = out.replaceAll(`{${k}}`, String(v))
  }
  return out
}

const I18n = createContext<{
  locale: Locale
  t: (key: MessageKey, vars?: Vars) => string
  setLocale: (l: Locale) => void
}>({
  locale: "en",
  t: (key, vars) => format(messages.en[key], vars),
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
    }),
    [locale],
  )
  return <I18n.Provider value={value}>{children}</I18n.Provider>
}

export function useT() {
  return useContext(I18n).t
}
