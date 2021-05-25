package varys

import irc "github.com/qaisjp/go-ircevent"

// Subset of irc.Event without unserialisable data
type Event struct {
	VarysUID string

	Code      string
	Raw       string
	Nick      string //<nick>
	Host      string //<nick>!<usr>@<host>
	Source    string //<host>
	User      string //<usr>
	Arguments []string
	Tags      map[string]string
}

func eventFomReal(uid string, input *irc.Event) *Event {
	return &Event{
		VarysUID: uid,

		Code:      input.Code,
		Raw:       input.Raw,
		Nick:      input.Nick,
		Host:      input.Host,
		Source:    input.Source,
		User:      input.User,
		Arguments: input.Arguments, // in-memory does not copy here!
		Tags:      input.Tags,      // in-memory does not copy here!
	}
}

func (e *Event) toReal() *irc.Event {
	return &irc.Event{
		Code:      e.Code,
		Raw:       e.Raw,
		Nick:      e.Nick,
		Host:      e.Host,
		Source:    e.Source,
		User:      e.User,
		Arguments: e.Arguments,
		Tags:      e.Tags,
	}
}
