package config

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"sort"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type ParsedDestination struct {
	Addr            string
	Username        string
	Password        string
	From            *mail.Address
	To              []*mail.Address
	SubjectTemplate *template.Template
	BodyTemplate    *template.Template
}

type Destination struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"` // defaults to From.Address
	jsontypes.Password

	From string   `json:"from,omitempty"` // RFC 5322 address, the address part used for auth against Host
	To   []string `json:"to,omitempty"`   // RFC 5322 address

	SendTime *jsontypes.Schedule `json:"sendTime,omitempty"` // defaults to midnight on Monday mornings: "0 0 * * 1"

	SubjectTemplate jsontypes.String `json:"subjectTemplate,omitempty"`
	BodyTemplate    jsontypes.String `json:"bodyTemplate,omitempty"`

	Parsed *ParsedDestination `json:"-"`
}

type Source struct {
	Name      string `json:"name,omitempty"`
	Title     string `json:"title,omitempty"`
	Floor     string `json:"floor,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Subsystem string `json:"subsystem,omitempty"`
}

type TemplateArgs struct {
	EmailTitle string `json:"emailTitle,omitempty"`
}

type Root struct {
	auto.Config
	// Configuration information for how to send the email.
	Destination Destination `json:"destination,omitempty"`

	AlertHubName string `json:"source,omitempty"`

	Now func() time.Time `json:"-"`

	Timeout time.Duration `json:"timeout,omitempty"`

	Subject string `json:"subject,omitempty"`

	TemplateArgs TemplateArgs `json:"templateArgs,omitempty"`
}

type AttachmentCfg struct {
	AttachmentName string
	Attachment     []byte
}

var (
	DefaultSendTime = jsontypes.MustParseSchedule("0 0 * * 1")
)

func ReadBytes(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return
	}
	if cfg.Destination.Host == "" {
		err = fmt.Errorf("destination.host not specified")
		return
	}
	if len(cfg.Destination.To) == 0 {
		err = fmt.Errorf("destination.recipients is empty")
		return
	}
	// defaults
	if cfg.Destination.SendTime == nil {
		cfg.Destination.SendTime = DefaultSendTime
	}

	// validate email addresses
	parsed, err := cfg.Destination.Parse()
	if err != nil {
		return
	}
	cfg.Destination.Parsed = parsed
	for i, recipient := range cfg.Destination.To {
		_, err = mail.ParseAddress(recipient)
		if err != nil {
			err = fmt.Errorf("destination.recipients[%d] is invalid: %w", i, err)
			return
		}
	}
	return
}

func (d Destination) Parse() (*ParsedDestination, error) {
	p := &ParsedDestination{}
	var err error
	p.Addr = d.Addr()
	p.Username = d.Username
	p.Password, err = d.Password.Read()
	if err != nil {
		return nil, fmt.Errorf("destination.password: %w", err)
	}

	p.From, err = mail.ParseAddress(d.From)
	if err != nil {
		return nil, fmt.Errorf("destination.from: %w", err)
	}
	if p.Username == "" {
		p.Username = p.From.Address
	}

	for i, to := range d.To {
		a, err := mail.ParseAddress(to)
		if err != nil {
			return nil, fmt.Errorf("destination.to[%d]: %w", i, err)
		}
		p.To = append(p.To, a)
	}
	sort.Slice(p.To, func(i, j int) bool {
		return p.To[i].Address < p.To[j].Address
	})

	p.SubjectTemplate, err = d.ReadSubjectTemplate()
	if err != nil {
		return nil, fmt.Errorf("destination.subjectTemplate: %w", err)
	}
	p.BodyTemplate, err = d.ReadBodyTemplate()
	if err != nil {
		return nil, fmt.Errorf("destination.bodyTemplate: %w", err)
	}
	return p, nil
}

// Addr returns the combination of Host and Port, taking defaults into account.
// Suitable for smtp.Dial.
func (d Destination) Addr() string {
	p := d.Port
	if p == 0 {
		p = 587
	}
	return net.JoinHostPort(d.Host, strconv.Itoa(p))
}

func (d Destination) ReadSubjectTemplate() (*template.Template, error) {
	s, err := d.SubjectTemplate.Read()
	if err != nil {
		return nil, err
	}
	if s == "" {
		s = DefaultEmailSubject
	}
	return template.New("subject").Parse(s)
}

func (d Destination) ReadBodyTemplate() (*template.Template, error) {
	s, err := d.BodyTemplate.Read()
	if err != nil {
		return nil, err
	}
	if s == "" {
		s = DefaultEmailBody
	}
	return template.New("body").
		Funcs(template.FuncMap{
			"printTime": func(t any) string {
				format := time.Stamp
				switch t := t.(type) {
				case time.Time:
					return t.Format(format)
				case *time.Time:
					return t.Format(format)
				case *timestamppb.Timestamp:
					return t.AsTime().Format(format)
				case nil:
					return ""
				default:
					return fmt.Sprintf("%v", t)
				}
			},
		}).
		Parse(s)
}

func (d *AttachmentCfg) AttachFile(name string, b []byte) {
	d.AttachmentName = name
	d.Attachment = b
}

const DefaultEmailSubject = `Smart Core Notifications Monthly Report {{.Now.Format "Jan 02, 2006"}}`
const DefaultEmailBody = `<html lang="en">
<head>
</head>
<body>
<section>
<h3>{{.EmailTitle}}</h3>
<h4>Please find attached a list of all notifications recorded by the system within the last month</h4>
</section>
</body>
</html>
`
