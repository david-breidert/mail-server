package receiver

import (
	"errors"
	"log"
	"strconv"
	"time"

	idle "github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"
)

// Listener is able to listen for incoming mails
type Listener struct {
	server   string
	email    string
	password string
	running  bool
}

//Run starts the Listener
func (l *Listener) Run(mails chan<- struct{}) error {
	l.running = true
	for l.running {
		log.Println("Connecting to server...")

		// Connect to server
		c, err := client.DialTLS("mail.example.org:993", nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Connected")

		// Login
		if err := c.Login("username", "password"); err != nil {
			log.Fatal(err)
		}
		log.Println("Logged in")
		defer c.Logout()

		// Select a mailbox
		if _, err := c.Select("INBOX", false); err != nil {
			log.Fatal(err)
		}

		idleClient := idle.NewClient(c)

		// Create a channel to receive mailbox updates
		updates := make(chan client.Update)
		c.Updates = updates

		// Start idling
		idl, err := idleClient.SupportIdle()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Client supports idle?: " + strconv.FormatBool(idl))

		done := make(chan error, 1)
		go func() {
			done <- idleClient.IdleWithFallback(nil, time.Second*2)
		}()

		// Listen for updates
		for {
			select {
			case update := <-updates:
				log.Println("New update:", update)
				mails <- struct{}{}
			case err := <-done:
				if err != nil {
					l.running = false
					log.Fatal(err)
				}
			}
		}
	}
	return errors.New("Not idling anymore")
}

//Stop stops the Listener
func (l *Listener) Stop() {
	l.running = false
}
