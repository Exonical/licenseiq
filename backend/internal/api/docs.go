package api

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed docs/assets/scalar.js
var scalarAssets embed.FS

const docsHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>LicenseIQ API Docs</title>
  <style>html,body,#app{margin:0;padding:0;height:100%;width:100%;}</style>
</head>
<body>
  <div id="app"></div>
  <script src="/docs/assets/scalar.js"></script>
  <script>
    Scalar.createApiReference('#app', { url: '/openapi.json', withDefaultFonts: false });
  </script>
</body>
</html>`

func MountDocs(router gin.IRouter) {
	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docsHTML))
	})
	router.GET("/docs/assets/scalar.js", func(c *gin.Context) {
		data, err := scalarAssets.ReadFile("docs/assets/scalar.js")
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
	})
}
