package receiver

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
)

type Alarm struct {
	Zeitstempel   time.Time `json:"zeitstempel"`
	Einsatznummer int       `json:"einsatznummer"`
	Ort           string    `json:"ort"`
	Ortsteil      string    `json:"ortsteil"`
	Strasse       string    `json:"strasse"`
	Hausnummer    int       `json:"hausnummer"`
	Objekt        string    `json:"objekt"`
	EOrtZusatz    string    `json:"eOrtZusatz"`
	LAT           string    `json:"lat"`
	LNG           string    `json:"lng"`
	Einsatzmittel string    `json:"einsatzmittel"`
	Stichwort     string    `json:"stichwort"`
	Text          string    `json:"text"`
	Meldender     string    `json:"meldender"`
	Telefonnummer string    `json:"telefonnummer"`
}

// Parse turns an imap.Literal and turns it into an Alarm
func Parse(r *imap.Literal) (Alarm, error) {
	var alarm Alarm
	mr, err := mail.CreateReader(*r)
	if err != nil {
		return Alarm{}, errors.New("Error creating MailReader")
	}

	header := mr.Header

	if from, err := header.AddressList("From"); err == nil && from[0].Address == os.Getenv("VALIDSENDER") {
		log.Println("Parser: Sender valid")
		date, err := header.Date()
		if err != nil {
			return Alarm{}, errors.New("Error getting Date header")
		}

		alarm.Zeitstempel = date

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				return Alarm{}, errors.New("Error reading next part")
			}

			t, _, err := p.Header.(*mail.InlineHeader).ContentType()
			if err != nil {
				return Alarm{}, errors.New("Error getting Date header")
			} else if t != "text/plain" {
				break
			}

			scanner := bufio.NewScanner(p.Body)

			line := 0

			for scanner.Scan() {
				line++

				t := scanner.Text()
				s := strings.SplitN(t, ":", 2)

				for i := range s {
					s[i] = strings.TrimSpace(s[i])
				}
				if s[0] != "" {
					switch s[0] {
					case "Einsatznummer":
						alarm.Einsatznummer, err = strconv.Atoi(s[1])
						if err != nil {
							log.Println("Error parsing Einsatznummer")
						}

					case "Ort":
						alarm.Ort = s[1]

					case "Ortsteil":
						alarm.Ortsteil = s[1]

					case "Strasse":
						alarm.Strasse = s[1]

					case "Haus-Nr.":
						alarm.Hausnummer, err = strconv.Atoi(s[1])
						if err != nil {
							log.Println("Error parsing Hausnummer")
						}

					case "Objekt":
						alarm.Objekt = s[1]

					case "E-Stelle-Zusatz":
						alarm.EOrtZusatz = s[1]

					case "Koordinate":
						reg := regexp.MustCompile(`\(([^)]+)\)`)
						res := reg.FindStringSubmatch(s[1])
						ks := strings.SplitN(strings.TrimSpace(res[1]), " ", 2)
						alarm.LNG = ks[0]
						alarm.LAT = ks[1]

					case "Stichwort":
						alarm.Stichwort = s[1]

					case "Bemerkung":
						alarm.Text = s[1]

					case "Meldender":
						ms := strings.SplitN(s[1], "/", 2)
						alarm.Meldender = strings.TrimSpace(ms[0])
						alarm.Telefonnummer = strings.Replace(strings.TrimSpace(ms[1]), "Tel.:", "", 1)

					case "Einsatzmittel":
						alarm.Einsatzmittel = s[1]

					}
				}

			}
		}
	} else {
		log.Println("Invalid Sender")
		return Alarm{}, errors.New("Invalid Sender")
	}

	return alarm, nil
}
