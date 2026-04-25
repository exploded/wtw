// htmx loading shimmer
document.body.addEventListener('htmx:beforeRequest', function(e) {
  e.detail.elt.classList.add('htmx-request');
});
document.body.addEventListener('htmx:afterRequest', function(e) {
  e.detail.elt.classList.remove('htmx-request');
});

// Poster image fallback (replaces inline onerror blocked by CSP)
document.addEventListener('error', function(e) {
  if (e.target.tagName === 'IMG' && e.target.closest('.poster-card__poster, .rec-feature__poster')) {
    e.target.parentNode.classList.add('poster--fallback');
    e.target.remove();
  }
}, true);
