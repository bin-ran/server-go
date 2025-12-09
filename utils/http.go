package utils

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	ParamsWrongString  = "Params Wrong"
	DBErrorString      = "DB Error"
	ReturnFailedString = "Return Data failed"
)

var (
	developmentMode bool
	webURL          string
)

func Init(DevelopmentMode bool, WebURL string) {
	developmentMode = DevelopmentMode
	webURL = WebURL
}

type responseData struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

/* type errorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
} */

func Sucess(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(responseData{Code: 0})
}

func SucessWithData(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(responseData{Code: 0, Data: data})
}

type GzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (gw *GzipResponseWriter) Write(data []byte) (int, error) {
	return gw.gz.Write(data)
}

func (gw *GzipResponseWriter) Close() error {
	return gw.gz.Close()
}

func GZipWithLevel(w http.ResponseWriter, level int) *GzipResponseWriter {
	w.Header().Set("Content-Encoding", "gzip")
	gz, _ := gzip.NewWriterLevel(w, level)
	return &GzipResponseWriter{ResponseWriter: w, gz: gz}
}

func GZip(w http.ResponseWriter) *GzipResponseWriter {
	return GZipWithLevel(w, gzip.DefaultCompression)
}

func CORS(next http.Handler, methods ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			var allowOrigin string

			if developmentMode {
				allowOrigin = origin
			} else {
				allowOrigin = webURL
			}

			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Origin-URL")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}

type methodHandler struct {
	method string
	next   http.Handler
}

func (m methodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != m.method {
		return false
	}

	m.next.ServeHTTP(w, r)
	return true
}

func Get(next http.Handler) methodHandler {
	return methodHandler{method: http.MethodGet, next: next}
}

func Post(next http.Handler) methodHandler {
	return methodHandler{method: http.MethodPost, next: next}
}

func Put(next http.Handler) methodHandler {
	return methodHandler{method: http.MethodPut, next: next}
}

func Delete(next http.Handler) methodHandler {
	return methodHandler{method: http.MethodDelete, next: next}
}

func Methods(next ...methodHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, m := range next {
			if m.ServeHTTP(w, r) {
				return
			}
		}

		http.NotFound(w, r)
	})
}
