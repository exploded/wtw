package middleware

import (
	"context"
	"net/http"

	"wtw/internal/session"
)

type contextKey string

const (
	UserIDKey     contextKey = "userID"
	UserNameKey   contextKey = "userName"
	UserEmailKey  contextKey = "userEmail"
	UserAvatarKey contextKey = "userAvatar"
)

func GetUserID(ctx context.Context) int64 {
	v, _ := ctx.Value(UserIDKey).(int64)
	return v
}

func GetUserName(ctx context.Context) string {
	v, _ := ctx.Value(UserNameKey).(string)
	return v
}

func GetUserEmail(ctx context.Context) string {
	v, _ := ctx.Value(UserEmailKey).(string)
	return v
}

func GetUserAvatar(ctx context.Context) string {
	v, _ := ctx.Value(UserAvatarKey).(string)
	return v
}

func RequireAuth(store *session.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		sess, user, err := store.GetWithUser(r.Context(), cookie.Value)
		if err != nil || sess == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, user.ID)
		ctx = context.WithValue(ctx, UserNameKey, user.Name)
		ctx = context.WithValue(ctx, UserEmailKey, user.Email)
		if user.AvatarUrl.Valid {
			ctx = context.WithValue(ctx, UserAvatarKey, user.AvatarUrl.String)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
