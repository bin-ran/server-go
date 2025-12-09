package routers

import (
	"context"
	"log/slog"
	"net/http"
	"server-go/managers"
	"server-go/utils"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RequestKey int

const (
	UserID RequestKey = iota + 1
	RequestParam
)

func Init() {
	account()
	admin()
}

func verify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		var msg string

		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			tokenCookie, err := r.Cookie("token")
			if err != nil {
				msg = "No Token"
				slog.Error(msg, "err", err)
				http.Error(w, msg, http.StatusUnauthorized)
				return
			}

			token = tokenCookie.Value
		} else {
			token = strings.TrimPrefix(tokenHeader, "Bearer ")

			if token == "" {
				msg = "No Token"
				slog.Error(msg)
				http.Error(w, msg, http.StatusUnauthorized)
				return
			}
		}

		id, err := managers.Redis.HGet(r.Context(), managers.TOKEN+token, "id").Result()
		if err != nil {
			if err == redis.Nil {
				// 没有找到对应的Token
				msg = "Not Found Token"
				slog.Error(msg)
				http.Error(w, msg, http.StatusUnauthorized)
			} else {
				msg := utils.CacheErrorString
				slog.Error(msg, "err", err)
				http.Error(w, msg, http.StatusInternalServerError)
			}

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserID, id)))
	})
}
