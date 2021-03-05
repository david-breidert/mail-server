package main

import (
	"log"
	"os"

	"github.com/david-breidert/mail-server/receiver"
	"github.com/joho/godotenv"
)

func init() {
	log.Println("Initializing dotenv")
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	listener := receiver.Listener{Server: os.Getenv("SERVER"), Email: os.Getenv("EMAIL"), Password: os.Getenv("PASSWORD")}
	newMails := make(chan struct{})
	go listener.Run(newMails)
	<-newMails
}
