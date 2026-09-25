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
