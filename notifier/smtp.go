package notifier

import (
	"fmt"
	"github.com/rkjdid/errors"
	"gopkg.in/gomail.v2"
	"log"
)

type SMTPConfig struct {
	Host, User, Pass string
	Port             int
	Name             string
	Subscribers      []string
}

type SMTPDialer struct {
	Config         *SMTPConfig
	*gomail.Dialer `json:"-" toml:"-"`
}

func (dialer *SMTPDialer) Init() {
	dialer.Dialer = gomail.NewDialer(
		dialer.Config.Host, dialer.Config.Port, dialer.Config.User, dialer.Config.Pass)
}

func (dialer SMTPDialer) Notify(header string, body string) error {
	return dialer.SendMail(dialer.Config.Subscribers, header, body)
}

func (dialer SMTPDialer) SendTestMail() error {
	return dialer.Notify(testMailHeader, testMailHtml)
}

func (dialer SMTPDialer) String() string {
	return fmt.Sprintf("%s", dialer.Config.Name)
}

func (dialer SMTPDialer) SendMail(dst []string, header string, body string) (err error) {
	msg := gomail.NewMessage()
	msg.SetAddressHeader("From", dialer.Username, dialer.String())
	msg.SetHeader("Subject", header)
	msg.SetBody("text/html; charset=utf-8", body)

	for k, to := range dst {
		msg.SetHeader("To", to)
		err1 := dialer.DialAndSend(msg)
		err = errors.Add(err, err1)
		if err1 == nil {
			log.Printf("%d: '%s' sent to '%s'", k, header, to)
		}
	}
	return err
}

const testMailHeader = `mic-test`
const testMailHtml = `
<h1>test-mail</h1>
<p>
	this mail was sent to you using latest cutting-edge software technology - brought to you by rkj!
</p>
<p>
	do not reply -
</p>
`
