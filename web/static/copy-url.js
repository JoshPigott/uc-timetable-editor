function copyTextFallback(value) {
  const input = document.createElement("textarea");
  input.value = value;
  input.setAttribute("readonly", "");
  input.className = "clipboard-fallback";
  document.body.appendChild(input);
  input.select();
  const copied = document.execCommand("copy");
  input.remove();
  return copied;
}

document.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-copy-url]");
  if (!button) return;

  const url = button.dataset.copyUrl;
  const status = document.getElementById("copy-status");
  if (!url || !status) return;

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
    } else if (!copyTextFallback(url)) {
      throw new Error("Clipboard copy failed");
    }
    status.textContent = "URL copied to clipboard.";
  } catch {
    try {
      if (!copyTextFallback(url)) throw new Error("Clipboard copy failed");
      status.textContent = "URL copied to clipboard.";
    } catch {
      status.textContent = "Copy failed. Please try again or use a supported browser.";
    }
  }
});
