package receiver

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/emersion/go-imap"
	idle "github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// Listener is able to listen for incoming mails
type Listener struct {
	Server   string
	Email    string
	Password string
}

var running = false

//Run starts the Listener
func (l *Listener) Run(mails chan<- struct{}) error {
	log.Println("Listener connecting to server...")
	c, err := client.DialTLS(l.Server, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Login(l.Email, l.Password); err != nil {
		log.Fatal(err)
	}
	log.Println("Listener logged in")
	defer c.Logout()

	if _, err := c.Select("INBOX", false); err != nil {
		log.Fatal(err)
	}

	updates := make(chan client.Update)
	c.Updates = updates

	idleClient := idle.NewClient(c)
	idl, err := idleClient.SupportIdle()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Client supports idle?: " + strconv.FormatBool(idl))

	// Start idling
	for {
		log.Println("looping")
		upd, err := l.waitForMailboxUpdate(idleClient, updates)
		if err != nil {
			panic(err)
		}
		l.fetchMessages(upd.Mailbox, c)
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
			log.Println("Got UPD")
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
	log.Println("logging")
	<-done

	return mboxupd, nil
}

func asMailboxUpdate(upd client.Update) *client.MailboxUpdate {
	log.Println("asMailboxUpdate")
	if v, ok := upd.(*client.MailboxUpdate); ok {
		return v
	} else {
		log.Println(upd)
	}
	return nil
}

func (l *Listener) fetchMessages(mb *imap.MailboxStatus, c *client.Client) {
	log.Println("trying to fetch messages")
	// Get the last message
	if mb.Messages == 0 {
		log.Fatal("No message in mailbox")
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

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		log.Fatal(err)
	}

	// Print some info about the message
	header := mr.Header
	if date, err := header.Date(); err == nil {
		log.Println("Date:", date)
	}
	if from, err := header.AddressList("From"); err == nil {
		log.Println("From:", from)
	}
	if to, err := header.AddressList("To"); err == nil {
		log.Println("To:", to)
	}
	if subject, err := header.Subject(); err == nil {
		log.Println("Subject:", subject)
	}

	// Process each message's part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := io.ReadAll(p.Body)
			log.Println(h.ContentType())
			log.Printf("Got text: %v", string(b))
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			log.Printf("Got attachment: %v", filename)
		}
	}
}
