package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/baaami/dorandoran/broker/pkg/redis"
)

func SessionMiddleware(redisClient *redis.RedisClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 로그인의 경우에는 인증을 하지 않음
			if strings.HasPrefix(r.URL.Path, "/auth") || strings.HasPrefix(r.URL.Path, "/profile") {
				next.ServeHTTP(w, r)
				return
			}

			// 쿠키에서 세션 ID 추출
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, "Unauthorized: No session ID provided", http.StatusUnauthorized)
				return
			}
			sessionID := cookie.Value

			// Redis에서 세션 ID로 사용자 정보 조회
			userID, err := redisClient.GetUserBySessionID(sessionID)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid session ID", http.StatusUnauthorized)
				return
			}

			// 사용자 ID를 요청의 헤더에 추가
			r.Header.Set("X-User-ID", strconv.Itoa(userID))

			// 다음 핸들러로 요청을 전달
			next.ServeHTTP(w, r)
		})
	}
}
