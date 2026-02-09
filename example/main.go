package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	ankylogo "github.com/arryllopez/ankyloGo"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	memoryStore := ankylogo.NewMemoryStore()
	router.Use(ankylogo.RateLimiterMiddleware(memoryStore)) // applying the middleware

	// LoggerWithFormatter middleware will write the logs to gin.DefaultWriter
	// By default gin.DefaultWriter = os.Stdout
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "login successful",
		})
	})

	router.GET("/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "search completed",
		})
	})

	router.POST("/purchase", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "purchase successful",
		})
	})

	router.GET("/test", func(c *gin.Context) {
		example := c.MustGet("example").(string)
		// it would print: "12345"
		log.Println(example)
	})

	router.Run() // listen and serve on 0.0.0.0:8080
}
