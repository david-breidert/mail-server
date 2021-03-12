package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/david-breidert/mail-server/receiver"
	"github.com/emersion/go-imap"
	"github.com/joho/godotenv"
)

func init() {
	log.Println("Initializing...")
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("no .env file found in app directory, using system variables")
	} else {
		log.Println("Dotenv loaded")
	}
}

func main() {
	listener := receiver.Listener{Server: os.Getenv("SERVER"), Email: os.Getenv("EMAIL"), Password: os.Getenv("PASSWORD")}
	newMails := make(chan *imap.Literal)
	go listener.Run(newMails)

	for {
		mail := <-newMails
		alm, err := receiver.Parse(mail)
		if err != nil {
			log.Println("error in parsing")
		} else {
			json, _ := json.MarshalIndent(alm, "", "    ")
			log.Println("New Alarm: \n", string(json))
		}
	}
}
