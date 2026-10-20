package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goddtriffin/helmet"
)

var helmetMiddleware = helmet.Default()

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		helmetMiddleware.Secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}
