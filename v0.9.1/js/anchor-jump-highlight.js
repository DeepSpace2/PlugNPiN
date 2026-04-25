function isElementInsideTable(element) {
  let currentElement = element;

  while (currentElement && currentElement !== document.body) {
    if (currentElement.tagName === 'TR') {
      return currentElement;
    }
    currentElement = currentElement.parentElement;
  }

  return undefined;
}

document.addEventListener('DOMContentLoaded', function() {
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function(e) {
      
      const targetId = this.getAttribute('href').substring(1);
      let targetElement = document.getElementById(targetId);

      let trElement = isElementInsideTable(targetElement);
      if (trElement !== undefined) {
        targetElement = trElement;
      }

      if (targetElement) {
        document.querySelectorAll('.highlight-target').forEach(el => {
          el.classList.remove('highlight-target');
        });

        targetElement.classList.add('highlight-target');

        setTimeout(() => {
          targetElement.classList.remove('highlight-target');
        }, 2000);
      }
    });
  });
});
