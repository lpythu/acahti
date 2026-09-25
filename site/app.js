function siteBase() {
  // Project Pages live under /acahti/; local/file and custom roots stay relative.
  const parts = location.pathname.split("/").filter(Boolean)
  if (parts[0] === "acahti") return "/acahti/"
  return "./"
}

function rootHost() {
  const el = document.getElementById("host")
  return (el && el.value ? el.value : "https://acahti.example.com").replace(/\/$/, "")
}

function orgName() {
  const el = document.getElementById("org")
  return (el && el.value ? el.value : "acme").trim() || "acme"
}

function demoPrompt(root, org) {
  return [
    `Install ${root}/skill.md`,
    `Install ${root}/demo.md`,
    "",
    `Run the Acahti demo loop on this instance now. Connect MCP with OAuth. Do not paste tokens. Do not change global git config. Create or reuse a disposable demo-* repo under ${org}, push a trivial branch with a smoke pipeline, wait for checks, open a PR, merge when green (or stop with evidence if no Runner). Reply with repo URL, PR number, and pipeline number. I will only watch the Board.`,
  ].join("\n")
}

function refreshPrompts() {
  const root = rootHost()
  const org = orgName()
  const skill = document.getElementById("install-skill")
  const demo = document.getElementById("install-demo")
  const prompt = document.getElementById("demo-prompt")
  if (skill) skill.textContent = `Install ${root}/skill.md`
  if (demo) demo.textContent = `Install ${root}/demo.md`
  if (prompt) prompt.textContent = demoPrompt(root, org)
}

async function copyById(id) {
  refreshPrompts()
  const node = document.getElementById(id)
  const text = node ? node.textContent : ""
  try {
    await navigator.clipboard.writeText(text)
    flash(id)
  } catch {
    window.prompt("Copy this:", text)
  }
}

function flash(id) {
  const block = document.getElementById(id)?.closest(".copy-block")
  const btn = document.querySelector(`[data-copy="${id}"]`)
  if (btn) {
    const prev = btn.textContent
    btn.textContent = "Copied"
    setTimeout(() => {
      btn.textContent = prev
    }, 1200)
  }
  if (block) {
    block.style.borderColor = "#87e4c8"
    setTimeout(() => {
      block.style.borderColor = ""
    }, 1200)
  }
}

function mountChrome(active) {
  const base = siteBase()
  const header = document.getElementById("site-header")
  const footer = document.getElementById("site-footer")
  if (header) {
    header.innerHTML = `
      <a class="brand" href="${base}">
        <img src="${base}assets/acahti.svg" width="28" height="28" alt="" />
        Acahti
      </a>
      <nav class="nav-links">
        <a href="${base}compare.html"${active === "compare" ? ' aria-current="page"' : ""}>Compare</a>
        <a href="${base}security.html"${active === "security" ? ' aria-current="page"' : ""}>Security</a>
        <a href="${base}self-host.html"${active === "self-host" ? ' aria-current="page"' : ""}>Self-host</a>
        <a href="${base}cloud.html"${active === "cloud" ? ' aria-current="page"' : ""}>Cloud</a>
        <a href="${base}pricing.html"${active === "pricing" ? ' aria-current="page"' : ""}>Pricing</a>
        <a href="https://github.com/lpythu/acahti">GitHub</a>
        <a class="btn ghost" href="${base}self-host.html">Install</a>
      </nav>`
  }
  if (footer) {
    footer.innerHTML = `
      <span>Agents ship. You watch.</span>
      <span class="foot-links">
        <a href="${base}pricing.html">Pricing</a>
        ·
        <a href="${base}cloud.html">Cloud waitlist</a>
        ·
        <a href="https://github.com/lpythu/acahti">GitHub</a>
      </span>`
  }
  document.querySelectorAll("[data-base-href]").forEach((el) => {
    const path = el.getAttribute("data-base-href") || ""
    el.setAttribute("href", base + path.replace(/^\.\//, ""))
  })
  document.querySelectorAll("[data-base-src]").forEach((el) => {
    const path = el.getAttribute("data-base-src") || ""
    el.setAttribute("src", base + path.replace(/^\.\//, ""))
  })
}

document.addEventListener("DOMContentLoaded", () => {
  mountChrome(document.body.dataset.page || "")
  document.querySelectorAll("[data-copy]").forEach((btn) => {
    btn.addEventListener("click", () => {
      void copyById(btn.getAttribute("data-copy"))
    })
  })
  ;["host", "org"].forEach((id) => {
    const el = document.getElementById(id)
    if (el) el.addEventListener("input", refreshPrompts)
  })
  refreshPrompts()

  const form = document.getElementById("waitlist-form")
  if (form) {
    form.addEventListener("submit", (e) => {
      e.preventDefault()
      const email = /** @type {HTMLInputElement} */ (form.querySelector('[name="email"]'))?.value?.trim()
      const size = /** @type {HTMLSelectElement} */ (form.querySelector('[name="size"]'))?.value || ""
      const prefer = /** @type {HTMLSelectElement} */ (form.querySelector('[name="prefer"]'))?.value || ""
      const subject = encodeURIComponent("Acahti Cloud waitlist")
      const body = encodeURIComponent(
        `Email: ${email}\nTeam size: ${size}\nPreference: ${prefer}\n\n(Same OSS kernel; export anytime.)`
      )
      window.location.href = `mailto:hello@acahti.dev?subject=${subject}&body=${body}`
    })
  }
})
