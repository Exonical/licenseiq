package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func MountOpenAPI(router gin.IRouter, api huma.API) {
	router.GET("/openapi.json", func(c *gin.Context) {
		data, err := api.OpenAPI().MarshalJSON()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	})
	router.GET("/openapi.yaml", func(c *gin.Context) {
		data, err := api.OpenAPI().YAML()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
	})
}
