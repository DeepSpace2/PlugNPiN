/* Subscribe to the document$ observable provided by MkDocs Material */
document$.subscribe(function() {
  const codeElements = document.querySelectorAll("table td:first-child code");

  codeElements.forEach(code => {
    if (code.closest(".copy-to-clipboard-wrapper")) return;

    const td = code.closest("td");
    if (!td) return;

    td.classList.add("copy-to-clipboard-td");

    const wrapper = document.createElement("span");
    wrapper.classList.add("copy-to-clipboard-wrapper");
    
    code.parentNode.insertBefore(wrapper, code);
    wrapper.appendChild(code);
    
    const button = document.createElement("button");
    button.type = "button";
    button.setAttribute("aria-label", "Copy to clipboard");
    button.setAttribute("title", "Copy to clipboard");
    button.classList.add("copy-to-clipboard-button");
    button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19 21H8V7h11m0-2H8a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2m-3-4H4a2 2 0 0 0-2 2v14h2V3h12V1Z"/></svg>';
    
    wrapper.appendChild(button);
    
    button.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();

      const textToCopy = code.textContent.trim();
      
      const fallbackCopy = () => {
        const textArea = document.createElement("textarea");
        textArea.value = textToCopy;
        textArea.style.position = "fixed";
        textArea.style.left = "-9999px";
        textArea.style.top = "0";
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        try {
          document.execCommand('copy');
          return true;
        } catch (err) {
          console.error('Fallback copy failed', err);
          return false;
        }
        document.body.removeChild(textArea);
      };

      if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(textToCopy)
          .then(() => showSuccess(button))
          .catch(() => {
            if (fallbackCopy()) showSuccess(button);
          });
      } else if (fallbackCopy()) {
        showSuccess(button);
      }
    });
  });

  function showSuccess(button) {
    const originalHTML = button.innerHTML;
    
    button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M21 7 9 19l-5.5-5.5 1.41-1.41L9 16.17 19.59 5.59 21 7Z"/></svg>';
    button.classList.add("copy-success");
    
    setTimeout(() => {
      button.innerHTML = originalHTML;
      button.classList.remove("copy-success");
    }, 1500);
  }
});
