package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	serviceURL := os.Getenv("SERVICE_INTERNAL_A_URL")
	port := os.Getenv("PORT")
	name := os.Getenv("NAME")

	fmt.Println("service name", name)

	r := gin.Default()
	r.GET("/internal", func(c *gin.Context) {
		// resp, err := http.Get(serviceURL + "/health")
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{
		// 		"error": "cannot reach service-internal",
		// 	})
		// 	return
		// }
		// defer resp.Body.Close()

		// body, _ := io.ReadAll(resp.Body)
		req, err := http.NewRequest(http.MethodGet, serviceURL+"/health", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "cannot create request",
			})
			return
		}

		req.Header.Add("X-Service-Name", name)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "cannot send request",
			})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		c.JSON(http.StatusOK, gin.H{
			"service":  "service-internal",
			"response": string(body),
		})
	})

	r.Run(":" + port)
}
