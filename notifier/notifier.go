package notifier

import (
	"log"
)

type Notifier interface {
	Notify(ctx string, message string) error
}

type LogNotifier struct{}

func (l LogNotifier) Notify(prefix string, text string) error {
	log.Printf("%s: %s", prefix, text)
	return nil
}
