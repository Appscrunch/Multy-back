package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func responseErr(c *gin.Context, err error, code int) {
	if err != nil {
		c.JSON(code, gin.H{
			"code":    code,
			"message": http.StatusText(code),
		})
	}

	return
}

func decodeBody(c *gin.Context, to interface{}) {
	body := json.NewDecoder(c.Request.Body)
	err := body.Decode(to)
	defer c.Request.Body.Close()
	responseErr(c, err, http.StatusInternalServerError) // 500
}

func makeRequest(c *gin.Context, url string, to interface{}) {
	response, err := http.Get(url)
	responseErr(c, err, http.StatusServiceUnavailable) // 503

	data, err := ioutil.ReadAll(response.Body)
	responseErr(c, err, http.StatusInternalServerError) // 500

	err = json.Unmarshal(data, to)
	responseErr(c, err, http.StatusInternalServerError) // 500
}
