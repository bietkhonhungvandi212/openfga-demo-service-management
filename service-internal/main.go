package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	client "github.com/openfga/go-sdk/client"
)

func main() {
	port := os.Getenv("PORT")
	name := os.Getenv("NAME")
	if port == "" {
		port = "8080"
	}

	fga := NewFGA()

	router := gin.Default()

	router.Use(Authorize(fga))

	router.Use(func(c *gin.Context) {
		c.Request.Header.Add("X-Service-Name", name)
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	router.Run(":" + port)
}

func Authorize(fga *client.OpenFgaClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		caller := c.Request.Header.Get("X-Service-Name")

		bodyOptions := client.ClientCheckRequest{
			User:     "service:" + caller,
			Relation: "can_call",
			Object:   "service:" + os.Getenv("NAME"),
		}

		fmt.Printf("bodyOptions::: %s, %s, %s\n", bodyOptions.User, bodyOptions.Relation, bodyOptions.Object)

		resp, err := fga.Check(context.Background()).
			Body(bodyOptions).
			Execute()

		if err != nil || *resp.Allowed != true {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Forbidden",
			})
			return
		}

		c.Next()
	}
}

func NewFGA() *client.OpenFgaClient {
	cfg := &client.ClientConfiguration{
		ApiUrl:  os.Getenv("OPENFGA_API"),
		StoreId: os.Getenv("STORE_ID"),
	}

	fga, err := client.NewSdkClient(cfg)
	if err != nil {
		panic(err)
	}

	return fga
}
