/* What to Watch — vanilla JS shell.
   Mirrors the htmx swap pattern: every screen is a section with [data-screen],
   navigation is via [data-go="<screen>"]. Server-side, each [data-go] becomes
   an hx-get to /screens/<name> that swaps #app innerHTML.
*/
(function () {
  'use strict';

  const POSTER_BASE = 'https://image.tmdb.org/t/p/w500';
  const SHOWS = window.SHOWS || [];
  const SHOW_BY_ID = {};
  SHOWS.forEach(s => { SHOW_BY_ID[s[0]] = { id:s[0], title:s[1], year:s[2], poster:s[3], genre:s[4] }; });

  /* In-memory ratings store. In the Go app, replace with htmx posts to
     /ratings/:show_id with form value "rating", and SQLite persistence. */
  const ratings = Object.assign({}, window.DEMO_RATINGS || {});

  /* ---------- screen routing ---------- */
  function showScreen(name) {
    document.querySelectorAll('[data-screen]').forEach(el => {
      el.hidden = el.getAttribute('data-screen') !== name;
    });
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  document.addEventListener('click', e => {
    const t = e.target.closest('[data-go]');
    if (!t) return;
    e.preventDefault();
    showScreen(t.getAttribute('data-go'));
  });

  /* ---------- poster card factory ---------- */
  function posterCard(show, opts) {
    opts = opts || {};
    const rating = ratings[show.id] || null;
    const card = document.createElement('div');
    card.className = 'poster-card' + (rating ? ' is-rated' : '');
    card.dataset.showId = show.id;
    if (rating) card.dataset.rating = rating;

    const badgeChar = rating === 'liked' ? '\u2713' : rating === 'disliked' ? '\u00d7' : rating === 'unseen' ? '\u2014' : '';

    card.innerHTML = `
      <div class="poster-card__poster">
        <img alt="${escapeHtml(show.title)}" loading="lazy"
             src="${POSTER_BASE}${show.poster}"
             onerror="this.parentNode.classList.add('poster--fallback');this.remove()">
        <div class="poster-card__poster-fb"><span>${escapeHtml(show.title)}</span></div>
        <div class="poster-card__badge">${badgeChar}</div>
        <div class="poster-card__rate" role="group" aria-label="Rate ${escapeHtml(show.title)}">
          <button class="rate-btn rate-btn--liked ${rating==='liked'?'is-active':''}" data-rate="liked" title="Liked" aria-label="Liked">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88L14 10h5.83a2 2 0 0 1 1.97 2.34l-1.5 9A2 2 0 0 1 18.33 23H7"/></svg>
          </button>
          <button class="rate-btn rate-btn--disliked ${rating==='disliked'?'is-active':''}" data-rate="disliked" title="Disliked" aria-label="Disliked">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12L10 14H4.17a2 2 0 0 1-1.97-2.34l1.5-9A2 2 0 0 1 5.67 1H17"/></svg>
          </button>
          <button class="rate-btn rate-btn--unseen ${rating==='unseen'?'is-active':''}" data-rate="unseen" title="Haven't seen" aria-label="Haven't seen">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2 2l20 20"/><path d="M6.7 6.7C3.7 8.4 2 12 2 12s3 7 10 7c2 0 3.7-.5 5.2-1.3"/><path d="M9.9 4.2A11 11 0 0 1 12 4c7 0 10 7 10 7-.7 1.3-1.5 2.5-2.5 3.5"/></svg>
          </button>
        </div>
      </div>
      <div class="poster-card__meta">
        <span class="poster-card__title">${escapeHtml(show.title)}</span>
        <span class="poster-card__year">${escapeHtml(show.year)} &middot; ${escapeHtml(show.genre)}</span>
      </div>
    `;

    /* rating clicks */
    card.querySelectorAll('.rate-btn').forEach(btn => {
      btn.addEventListener('click', ev => {
        ev.stopPropagation();
        const next = btn.dataset.rate;
        const cur  = ratings[show.id];
        if (cur === next) {
          delete ratings[show.id];
          card.removeAttribute('data-rating');
          card.classList.remove('is-rated');
        } else {
          ratings[show.id] = next;
          card.dataset.rating = next;
          card.classList.add('is-rated');
        }
        const newBadge = ratings[show.id] === 'liked' ? '\u2713' : ratings[show.id] === 'disliked' ? '\u00d7' : ratings[show.id] === 'unseen' ? '\u2014' : '';
        card.querySelector('.poster-card__badge').textContent = newBadge;
        card.querySelectorAll('.rate-btn').forEach(b => b.classList.toggle('is-active', b.dataset.rate === ratings[show.id]));
        updateCounters();
      });
    });

    return card;
  }

  function escapeHtml(s){return String(s).replace(/[&<>"']/g, c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));}

  /* ---------- populate grids ---------- */
  function fill(elId, ids) {
    const el = document.getElementById(elId);
    if (!el) return;
    el.innerHTML = '';
    ids.forEach(id => {
      const s = SHOW_BY_ID[id];
      if (s) el.appendChild(posterCard(s));
    });
  }

  function fillAll(elId, list) {
    const el = document.getElementById(elId);
    if (!el) return;
    el.innerHTML = '';
    list.forEach(s => el.appendChild(posterCard(s)));
  }

  /* Counters */
  function updateCounters() {
    const liked    = Object.values(ratings).filter(v => v === 'liked').length;
    const disliked = Object.values(ratings).filter(v => v === 'disliked').length;
    const total    = SHOWS.length;
    const rated    = Object.keys(ratings).length;
    const unrated  = total - rated;
    setText('catLiked', liked);
    setText('catDisliked', disliked);
    setText('catUnseen', unrated);
    setText('onboardCount', rated);
    const bar = document.getElementById('onboardProgress');
    if (bar) bar.style.width = Math.min(100, (rated / 12) * 100) + '%';
  }
  function setText(id, v){const el=document.getElementById(id); if(el) el.textContent=v;}

  /* Catalog filters / search */
  function bindCatalog() {
    const search = document.getElementById('catSearch');
    const filterBtns = document.querySelectorAll('.filter');
    let activeFilter = 'all';
    function apply() {
      const q = (search?.value || '').toLowerCase().trim();
      const list = SHOWS.map(s => SHOW_BY_ID[s[0]]).filter(s => {
        if (q && !s.title.toLowerCase().includes(q)) return false;
        if (activeFilter === 'liked')    return ratings[s.id] === 'liked';
        if (activeFilter === 'disliked') return ratings[s.id] === 'disliked';
        if (activeFilter === 'unseen')   return !ratings[s.id];
        return true;
      });
      fillAll('catalogGrid', list);
    }
    filterBtns.forEach(b => b.addEventListener('click', () => {
      filterBtns.forEach(x => x.classList.remove('filter--active'));
      b.classList.add('filter--active');
      activeFilter = b.dataset.filter;
      apply();
    }));
    search?.addEventListener('input', apply);
    apply();
  }

  /* Refresh recs button */
  function bindRefresh() {
    const b = document.getElementById('refreshRecs');
    if (!b) return;
    b.addEventListener('click', () => {
      b.classList.add('is-spinning');
      const pool = SHOWS.map(s => s[0]).filter(id => !ratings[id] || ratings[id] === 'unseen');
      shuffle(pool);
      fill('recsRow', pool.slice(0, 6));
      setTimeout(() => b.classList.remove('is-spinning'), 400);
    });
  }
  function shuffle(a){for(let i=a.length-1;i>0;i--){const j=Math.floor(Math.random()*(i+1));[a[i],a[j]]=[a[j],a[i]];}}

  /* Init */
  function init() {
    /* Onboarding: show first 24 */
    fillAll('onboardGrid', SHOWS.slice(0, 24).map(s => SHOW_BY_ID[s[0]]));

    /* Recs rows */
    fill('recsRow',          window.RECS_TONIGHT);
    fill('recsRowSeverance', window.RECS_SEVERANCE);
    fill('recsRowComfort',   window.RECS_COMFORT);

    /* Profile top */
    fill('profileTop', window.PROFILE_TOP);

    bindCatalog();
    bindRefresh();
    updateCounters();

    /* Decorative landing poster wall */
    buildLandingWall();
  }

  function buildLandingWall() {
    const wall = document.querySelector('.landing__poster-grid');
    if (!wall) return;
    const ids = SHOWS.slice(0, 40);
    wall.innerHTML = ids.map(s => `
      <div style="aspect-ratio:2/3;border-radius:8px;overflow:hidden;background:#1a1714;">
        <img src="${POSTER_BASE}${s[3]}" alt="" loading="lazy" style="width:100%;height:100%;object-fit:cover;"
             onerror="this.style.display='none'">
      </div>`).join('');
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
