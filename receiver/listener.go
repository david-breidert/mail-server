package receiver

import (
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	idle "github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"
)

// Listener is able to listen for incoming mails
type Listener struct {
	Server   string
	Email    string
	Password string
}

var running = false

//Run starts the Listener
func (l *Listener) Run(mails chan<- *imap.Literal) error {
	log.Println("Listener: Connecting to mail server...")
	c, err := client.DialTLS(l.Server, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Login(l.Email, l.Password); err != nil {
		log.Fatal(err)
	}
	log.Println("Listener: Logged in")
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
		log.Fatal(err)
	}

	updates := make(chan client.Update)
	c.Updates = updates

	idleClient := idle.NewClient(c)

	// Start idling
	for {
		log.Println("Listener: Listening for incoming mails...")
		upd, err := l.waitForMailboxUpdate(idleClient, updates)
		if err != nil {
			panic(err)
		}
		if mail := l.fetchMessages(upd.Mailbox, c); mail != nil {
			mails <- mail
		}
	}
}

func (l *Listener) waitForMailboxUpdate(ic *idle.Client, updates <-chan client.Update) (*client.MailboxUpdate, error) {
	done := make(chan error, 1)
	stop := make(chan struct{})
	go func() {
		done <- ic.IdleWithFallback(stop, 5*time.Second)
	}()
	var mboxupd *client.MailboxUpdate

waitLoop:
	for {
		select {
		case upd := <-updates:
			if mboxupd = asMailboxUpdate(upd); mboxupd != nil {
				break waitLoop
			}
		case err := <-done:
			log.Println("err")
			if err != nil {
				return nil, fmt.Errorf("error while idling: %s", err.Error())
			}
			log.Println("nil return")
			return nil, nil
		}
	}

	close(stop)
	<-done
	log.Println("Listener: Received new mail, passing to parser.")
	return mboxupd, nil
}

func asMailboxUpdate(upd client.Update) *client.MailboxUpdate {
	if v, ok := upd.(*client.MailboxUpdate); ok {
		return v
	}
	return nil
}

func (l *Listener) fetchMessages(mb *imap.MailboxStatus, c *client.Client) *imap.Literal {
	if mb.Messages == 0 {
		log.Println("Keine neuen Nachrichten")
		return nil
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(mb.Messages)

	// Get the whole message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			return
		}
	}()

	msg := <-messages
	if msg == nil {
		log.Fatal("Server didn't return message")
	}

	r := msg.GetBody(&section)
	if r == nil {
		log.Fatal("Server didn't return message body")
	}
	return &r
}
