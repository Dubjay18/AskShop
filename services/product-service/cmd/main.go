package main

import (
	"askshop/shared/env"
	"log"

	"github.com/gin-gonic/gin"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8082")
)

func main() {
	log.Println("Starting Product Service")

	// Set Gin mode based on environment
	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	//TODO Initialize repository

	// TODO Initialize service
	// TODO Initialize HTTP handler

	// TODO Create Gin router

	//TODO  Add CORS middleware

	// TODO Register routes

	// TODO Add a root route

	// TODO Start the server
}
