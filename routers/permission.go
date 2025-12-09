package routers

import (
	"log/slog"
	"net/http"
	"server-go/managers"
	"server-go/models"
)

// RequirePermission 权限检查中间件
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserID).(string)

			var user models.User
			if err := managers.DB.First(&user, userID).Error; err != nil {
				slog.Error("Failed to get user", "err", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// 加载权限
			if err := user.LoadPermissions(); err != nil {
				slog.Error("Failed to load permissions", "err", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !user.HasPermission(permission) {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole 角色检查中间件
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserID).(string)

			var user models.User
			if err := managers.DB.First(&user, userID).Error; err != nil {
				slog.Error("Failed to get user", "err", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// 加载角色
			if err := managers.DB.Preload("Role").First(&user, user.ID).Error; err != nil {
				slog.Error("Failed to load roles", "err", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !user.HasRole(role) {
				http.Error(w, "Forbidden: insufficient role", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
