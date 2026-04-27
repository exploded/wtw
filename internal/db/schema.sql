CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  google_sub    TEXT NOT NULL UNIQUE,
  email         TEXT NOT NULL,
  name          TEXT NOT NULL,
  avatar_url    TEXT,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shows (
  id            TEXT PRIMARY KEY,
  tmdb_id       INTEGER UNIQUE,
  title         TEXT NOT NULL,
  year          TEXT NOT NULL,
  poster_path   TEXT,
  genre         TEXT,
  synopsis      TEXT,
  popularity    REAL DEFAULT 0,
  cached_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ratings (
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  show_id       TEXT    NOT NULL REFERENCES shows(id),
  rating        TEXT    NOT NULL CHECK (rating IN ('favourite','liked','disliked','unseen')),
  rated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, show_id)
);

CREATE TABLE IF NOT EXISTS sessions (
  token         TEXT PRIMARY KEY,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at    DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS verdicts (
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  verdict       TEXT NOT NULL,
  headline      TEXT NOT NULL,
  show_id       TEXT REFERENCES shows(id),
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS partnerships (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  partner_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
  partner_email TEXT NOT NULL,
  status        TEXT NOT NULL CHECK (status IN ('pending','active','declined')),
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, partner_email)
);

CREATE TABLE IF NOT EXISTS partner_verdicts (
  partnership_id INTEGER NOT NULL REFERENCES partnerships(id) ON DELETE CASCADE,
  verdict        TEXT NOT NULL,
  headline       TEXT NOT NULL,
  show_id        TEXT REFERENCES shows(id),
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (partnership_id)
);

CREATE INDEX IF NOT EXISTS idx_ratings_user ON ratings(user_id);
CREATE INDEX IF NOT EXISTS idx_ratings_show ON ratings(show_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_partnerships_user ON partnerships(user_id);
CREATE INDEX IF NOT EXISTS idx_partnerships_email ON partnerships(partner_email);
CREATE INDEX IF NOT EXISTS idx_partnerships_partner ON partnerships(partner_id);
