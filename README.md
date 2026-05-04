# What to Watch

### *A cinephile in your pocket.*

Stop scrolling. Start watching. **What to Watch** learns your taste in TV in about a minute, then tells you exactly which series to put on tonight — and *why*.

### [Try it free at wtw.mchugh.au &rarr;](https://wtw.mchugh.au)

---

## The pitch

Streaming has more shows than ever and recommendations that feel worse than ever. Every platform pushes whatever it's paying to promote. Your friends only know what *they've* seen. And the algorithm just keeps serving more of the same.

**What to Watch** is different. It's a single, honest opinion — an AI cinephile that has watched everything, knows what you actually love, and isn't trying to sell you anything.

---

## What you get

### Tonight's verdict
One pick. One paragraph. Your personal cinephile reads your ratings and writes a short, opinionated take on what you should put on **tonight** — naming a single series, explaining what makes it your kind of thing, and what mood it's right for. Don't like the call? Hit *new verdict* and it'll argue for something else.

### Rate the canon in 60 seconds
Twenty of the most beloved series on a single page. Tap the ones you've loved. Tap the ones you couldn't stand. Skip the rest. Three ratings is all it takes to unlock recommendations — but the more you tick, the sharper the picks get.

### A wall of posters, not a wall of text
Browse a curated catalog of hundreds of series. Filter to your favourites, your dislikes, your unrated. Search the whole TMDB database to add anything that's missing. Every poster is one tap to rate.

### "Because you liked..." rows
Beyond tonight's pick, get themed rows: *Because you loved Severance*, *Comfort rewatches we think you'll love*, *More for tonight*. Refresh any row to get a fresh set.

### Watch together
Link your account with a partner, flatmate, or friend. **Together mode** finds the overlap in your taste, calls out series you've both loved, and surfaces picks you'll *both* enjoy — no more arguing over the remote. Invitations are a single email; either side can unlink anytime.

### Your taste profile
A page that just shows *you*: your top-rated series, your stats, and a short written summary of your taste — the kind of thing you'd put in a dating app bio if dating apps were about TV.

### Refines forever
Every rating you add — anywhere in the app — sharpens the next set of recommendations. Mark something as a favourite from the verdict screen. Dismiss a recommendation as *not for me*. The cinephile remembers.

---

## Sign-in & privacy

- One-tap **Sign in with Google**. No passwords, no sign-up form.
- Free. Always.
- Your taste data is yours. **We never sell it, share it, or train models on it.**

---

## Built with

Pure Go HTTP server, [HTMX](https://htmx.org) for snappy partial updates, SQLite, [Anthropic Claude](https://www.anthropic.com) for the cinephile's voice, [TMDB](https://www.themoviedb.org) for show metadata, and Google OAuth for sign-in.

## Run it locally

```bash
cp .env.example .env
# fill in: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL,
#         TMDB_API_KEY, ANTHROPIC_API_KEY, PORT

./build.bat   # Windows
# or:  go build -o wtw ./cmd/wtw && ./wtw
```

Server listens on `:8080` (override with `PORT`). SQLite database is created at `wtw.db` on first run.

## License

Private project. Source available for reference.
