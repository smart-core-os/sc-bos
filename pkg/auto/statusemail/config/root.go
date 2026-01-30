package config

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/mail"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func ReadBytes(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if cfg.Debounce != nil {
		for i, source := range cfg.Sources {
			if source.Debounce == nil {
				source.Debounce = cfg.Debounce
				cfg.Sources[i] = source
			}
		}
	}
	if cfg.Destination.Host == "" {
		err = fmt.Errorf("destination.host not specified")
	}
	if len(cfg.Destination.To) == 0 {
		err = fmt.Errorf("destination.recipients is empty")
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

type Root struct {
	auto.Config
	// Configuration information for how to send the email.
	Destination Destination `json:"destination,omitempty"`
	// If true, all devices on the current node that implement Status will be monitored.
	// Additional sources may be defined via Sources.
	DiscoverSources bool `json:"discoverSources,omitempty"`
	// Name of the devices that implement Status trait and that are monitored.
	Sources []Source `json:"sources,omitempty"`
	// Delay querying the status of devices by this much, to allow them to boot up.
	DelayStart *jsontypes.Duration `json:"delayStart,omitempty"`
	// Default debounce time for all sources.
	Debounce *jsontypes.Duration `json:"debounce,omitempty"`
	// Device name prefixes to ignore.
	// Only used if DiscoverSources is true.
	IgnorePrefixes []string `json:"ignorePrefixes,omitempty"`
}

type Destination struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"` // defaults to From.Address
	jsontypes.Password
	// todo: TLS config

	From string   `json:"from,omitempty"` // RFC 5322 address, the address part used for auth against Host
	To   []string `json:"to,omitempty"`   // RFC 5322 address

	MinInterval *jsontypes.Duration `json:"minInterval,omitempty"` // minimum time between emails, to avoid spamming

	SubjectTemplate jsontypes.String `json:"subjectTemplate,omitempty"`
	BodyTemplate    jsontypes.String `json:"bodyTemplate,omitempty"`

	Parsed *ParsedDestination `json:"-"`
}

type ParsedDestination struct {
	Addr            string
	Username        string
	Password        string
	From            *mail.Address
	To              []*mail.Address
	SubjectTemplate *template.Template
	BodyTemplate    *template.Template
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
	return fmt.Sprintf("%s:%d", d.Host, p)
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

type Source struct {
	Name      string `json:"name,omitempty"`
	Title     string `json:"title,omitempty"`
	Floor     string `json:"floor,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Subsystem string `json:"subsystem,omitempty"`

	// Don't send emails until after this time expires, reduces noise.
	Debounce *jsontypes.Duration `json:"debounce,omitempty"`
}

const DefaultDebounce = 15 * time.Second

func (s Source) DebounceOrDefault() time.Duration {
	return s.Debounce.Or(DefaultDebounce)
}

const DefaultEmailSubject = "Smart Core Notification"
const DefaultEmailBody = `<html lang="en">
<head>
  <title>Smart Core Notification</title>
  <style>
    table {border-collapse: collapse;}
    td, th {border: 1px solid black; padding: 0.2em 0.5em;}
  </style>
</head>
<body>
<h1>A Smart Core status notification has been triggered</h1>
<p>
  {{with len .WorseLogs}}{{.}} notifications have worsened.{{end}}
  {{with len .BetterLogs}}{{.}} notifications have improved.{{end}}
</p>
<table>
  <thead>
    <tr>
      <td>Status</td>
      <td>Device</td>
      <td>Description</td>
      <td>Change Time</td>
      <td>Last Status</td>
    </tr>
  </thead>
  <tbody>
    {{if .WorseLogs}}
    <tr>
      <td colspan="5">The following notifications have worsened:</td>
    </tr>
    {{range .WorseLogs}}
    <tr>
      <td>{{.Read.Level}}</td>
      {{if .Source.Title}}
      <td title="{{.Source.Name}}">{{.Source.Floor}} {{.Source.Zone}} {{.Source.Subsystem}} {{.Source.Title}}</td>
      {{else}}
      <td>{{.Source.Name}}</td>
      {{end}}
      <td>{{.Read.Description}}</td>
      <td>{{printTime .Read.RecordTime}}</td>
      <td>{{.Sent.Level}}</td>
    </tr>
    {{end}}
    {{end}}
    {{if .BetterLogs}}
    <tr style="border-top: 1px solid currentColor">
      <td colspan="5">The following notifications have improved:</td>
    </tr>
    {{range .BetterLogs}}
    <tr>
      <td>{{.Read.Level}}</td>
      {{if .Source.Title}}
      <td title="{{.Source.Name}}">{{.Source.Floor}} {{.Source.Zone}} {{.Source.Subsystem}}  {{.Source.Title}}</td>
      {{else}}
      <td>{{.Source.Name}}</td>
      {{end}}
      <td>{{.Read.Description}}</td>
      <td>{{printTime .Read.RecordTime}}</td>
      <td>{{.Sent.Level}}</td>
    </tr>
    {{end}}
    {{end}}
  </tbody>
</table>
</body>
</html>
`
