-- name: UpsertUser :one
INSERT INTO users (google_sub, email, name, avatar_url)
VALUES (?, ?, ?, ?)
ON CONFLICT(google_sub) DO UPDATE SET email=excluded.email, name=excluded.name, avatar_url=excluded.avatar_url
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetShows :many
SELECT * FROM shows ORDER BY popularity DESC;

-- name: GetShowByID :one
SELECT * FROM shows WHERE id = ?;

-- name: GetUserRatings :many
SELECT show_id, rating FROM ratings WHERE user_id = ?;

-- name: GetUserRatingsWithShows :many
SELECT r.show_id, r.rating, s.title, s.poster_path, s.genre, s.year
FROM ratings r JOIN shows s ON s.id = r.show_id
WHERE r.user_id = ?
ORDER BY r.rated_at DESC;

-- name: SetRating :exec
INSERT INTO ratings (user_id, show_id, rating)
VALUES (?, ?, ?)
ON CONFLICT(user_id, show_id) DO UPDATE SET rating = excluded.rating, rated_at = CURRENT_TIMESTAMP;

-- name: DeleteRating :exec
DELETE FROM ratings WHERE user_id = ? AND show_id = ?;

-- name: CountRatings :one
SELECT
  SUM(CASE WHEN rating='liked' THEN 1 ELSE 0 END) AS liked,
  SUM(CASE WHEN rating='disliked' THEN 1 ELSE 0 END) AS disliked,
  COUNT(*) AS total
FROM ratings WHERE user_id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE token = ? AND expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= datetime('now');

-- name: GetVerdict :one
SELECT * FROM verdicts WHERE user_id = ? AND created_at > datetime('now', '-1 day');

-- name: UpsertVerdict :exec
INSERT INTO verdicts (user_id, verdict, headline, show_id)
VALUES (?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET verdict=excluded.verdict, headline=excluded.headline, show_id=excluded.show_id, created_at=CURRENT_TIMESTAMP;

-- name: UpdateShowMeta :exec
UPDATE shows SET poster_path = ?, synopsis = ?, tmdb_id = ?, popularity = ?, cached_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetUnratedShows :many
SELECT s.* FROM shows s
LEFT JOIN ratings r ON r.show_id = s.id AND r.user_id = ?
WHERE r.show_id IS NULL
ORDER BY s.popularity DESC;

-- name: GetLikedShows :many
SELECT s.* FROM shows s
JOIN ratings r ON r.show_id = s.id AND r.user_id = ? AND r.rating = 'liked'
ORDER BY r.rated_at DESC;

-- name: GetDislikedShows :many
SELECT s.* FROM shows s
JOIN ratings r ON r.show_id = s.id AND r.user_id = ? AND r.rating = 'disliked'
ORDER BY r.rated_at DESC;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CreatePartnership :one
INSERT INTO partnerships (user_id, partner_email, partner_id, status)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetActivePartnershipsAsOwner :many
SELECT p.id, p.user_id, p.partner_id, p.partner_email, p.status, p.created_at,
       u.name AS partner_name, u.avatar_url AS partner_avatar
FROM partnerships p
LEFT JOIN users u ON u.id = p.partner_id
WHERE p.user_id = ? AND p.status = 'active';

-- name: GetActivePartnershipsAsPartner :many
SELECT p.id, p.user_id, p.partner_id, p.partner_email, p.status, p.created_at,
       u.name AS partner_name, u.avatar_url AS partner_avatar
FROM partnerships p
LEFT JOIN users u ON u.id = p.user_id
WHERE p.partner_id = ? AND p.status = 'active';

-- name: GetPendingPartnershipsByUser :many
SELECT * FROM partnerships
WHERE user_id = ? AND status = 'pending';

-- name: GetIncomingPartnerRequests :many
SELECT p.id, p.user_id, p.partner_email, p.status, p.created_at,
       u.name AS requester_name, u.email AS requester_email
FROM partnerships p
JOIN users u ON u.id = p.user_id
WHERE p.partner_email = ? AND p.status = 'pending';

-- name: GetPartnershipByID :one
SELECT * FROM partnerships WHERE id = ?;

-- name: ActivatePartnership :exec
UPDATE partnerships SET status = 'active', partner_id = ? WHERE id = ?;

-- name: DeletePartnership :exec
DELETE FROM partnerships WHERE id = ?;

-- name: GetPartnerVerdict :one
SELECT * FROM partner_verdicts
WHERE partnership_id = ? AND created_at > datetime('now', '-1 day');

-- name: UpsertPartnerVerdict :exec
INSERT INTO partner_verdicts (partnership_id, verdict, headline, show_id)
VALUES (?, ?, ?, ?)
ON CONFLICT(partnership_id) DO UPDATE SET
  verdict=excluded.verdict, headline=excluded.headline,
  show_id=excluded.show_id, created_at=CURRENT_TIMESTAMP;

-- name: GetBothLikedShows :many
SELECT s.* FROM shows s
JOIN ratings r1 ON r1.show_id = s.id AND r1.user_id = ? AND r1.rating = 'liked'
JOIN ratings r2 ON r2.show_id = s.id AND r2.user_id = ? AND r2.rating = 'liked'
ORDER BY s.popularity DESC;
