package logging

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Logger = *zap.Logger
type Field = zap.Field

func New() Logger              { l, _ := zap.NewProduction(); return l }
func String(k, v string) Field { return zap.String(k, v) }
func Error(e error) Field      { return zap.Error(e) }
func HTTPMiddleware(log Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http request", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Duration("duration", time.Since(s)))
	})
}
