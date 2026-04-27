package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"wtw/internal/db"
)

const (
	MaxAge    = 30 * 24 * time.Hour
	MaxAgeSec = 30 * 24 * 60 * 60
)

type Store struct {
	queries *db.Queries
}

func NewStore(queries *db.Queries) *Store {
	return &Store{queries: queries}
}

func (s *Store) Create(ctx context.Context, userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(MaxAge)

	err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expires.UTC(),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) Get(ctx context.Context, token string) (*db.Session, error) {
	sess, err := s.queries.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) GetWithUser(ctx context.Context, token string) (*db.Session, *db.User, error) {
	sess, err := s.queries.GetSession(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.queries.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	return &sess, &user, nil
}

func (s *Store) Delete(ctx context.Context, token string) error {
	return s.queries.DeleteSession(ctx, token)
}

func (s *Store) CleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.queries.DeleteExpiredSessions(context.Background()); err != nil {
				slog.Error("failed to cleanup expired sessions", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
