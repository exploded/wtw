// htmx loading shimmer
document.body.addEventListener('htmx:beforeRequest', function(e) {
  e.detail.elt.classList.add('htmx-request');
});
document.body.addEventListener('htmx:afterRequest', function(e) {
  e.detail.elt.classList.remove('htmx-request');
});

// Update catalog stats on rating change
document.body.addEventListener('statsUpdate', function(e) {
  var old = e.detail.old;
  var neu = e.detail.new;
  var statIds = ['favourite', 'liked', 'disliked'];

  // Decrement old bucket (or unrated if empty)
  var decId = 'stat-' + (old && statIds.indexOf(old) !== -1 ? old : 'unrated');
  var decEl = document.getElementById(decId);
  if (decEl) decEl.textContent = Math.max(0, parseInt(decEl.textContent) - 1);

  // Increment new bucket (or unrated if empty)
  var incId = 'stat-' + (neu && statIds.indexOf(neu) !== -1 ? neu : 'unrated');
  var incEl = document.getElementById(incId);
  if (incEl) incEl.textContent = parseInt(incEl.textContent) + 1;
});

// Poster image fallback (replaces inline onerror blocked by CSP)
document.addEventListener('error', function(e) {
  if (e.target.tagName === 'IMG' && e.target.closest('.poster-card__poster, .rec-feature__poster')) {
    e.target.parentNode.classList.add('poster--fallback');
    e.target.remove();
  }
}, true);
