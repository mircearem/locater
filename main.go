package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/mircearem/locater/geo"
	"github.com/sirupsen/logrus"
)

// Load the information from the .env file
func init() {
	// Setup logrus

	// Load environment variables
	err := godotenv.Load(".env")
	// Stop the app if the .env file is not found
	if err != nil {
		logrus.Fatalln("Unable to load .env file")
	}
}

// The GPRS conncection information read from the mdmd configurator
func main() {
	ctx := context.Background()
	s := geo.NewServer(ctx)
	logrus.Fatalln(s.Start())
}
