package statusemail

import (
	"crypto/tls"
	"fmt"
	"math"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/time/rate"

	"github.com/smart-core-os/sc-bos/pkg/auto/statusemail/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

type change struct {
	log    *statuspb.StatusLog
	source config.Source
}

func sendEmailOnChange(dst config.Destination, c <-chan change, logger *zap.Logger) {
	seen := make(map[string]Status)

	retryTimer := newStoppedTimer()
	var failedAttempts int
	const firstRetryDelay = 200 * time.Millisecond
	const nextAttemptScale = 1.2
	const maxAttemptDelay = 30 * time.Second
	errLogLimit := rate.Sometimes{
		Interval: 10 * time.Minute,
		Every:    50,
	}

	minInterval := dst.MinInterval.Or(10 * time.Minute)
	sendTicker := time.NewTicker(minInterval)
	defer sendTicker.Stop()

	// Calculates an email to send and sends it.
	// Only sends an email if there's something to say, aka if any device got better or worse.
	sendUpdates := func(now time.Time) {
		// work out if there's anything to send,
		// then configure our email client and send the email
		vars := Attrs{}
		for _, s := range seen {
			// undefined levels are handled in the select below
			s := s

			if s.Read.Level > statuspb.StatusLog_NOTICE {
				vars.BadLogs = append(vars.BadLogs, &s)
			}
			switch {
			case s.Sent == nil: // don't send, but record we've seen it
				s.Sent = s.Read
				seen[s.Source.Name] = s
			case s.Read.Level > s.Sent.Level: // things got worse
				// only record worse status changes if they've been worse for a while
				badDur := now.Sub(s.Read.RecordTime.AsTime())
				if badDur > minInterval {
					vars.WorseLogs = append(vars.WorseLogs, &s)
				}
			case s.Read.Level < s.Sent.Level: // things got better
				// only record better status changes if they were bad for a while
				badDur := s.Read.RecordTime.AsTime().Sub(s.Sent.RecordTime.AsTime())
				if badDur > minInterval {
					vars.BetterLogs = append(vars.BetterLogs, &s)
				}
			default: // things are the same
				vars.SameLogs = append(vars.SameLogs, &s)
			}
			vars.AllLogs = append(vars.AllLogs, &s)
		}

		if len(vars.WorseLogs) == 0 && len(vars.BetterLogs) == 0 {
			return // no changes recorded, nothing to send
		}

		cmp := func(s []*Status) func(i, j int) bool {
			return func(i, j int) bool {
				return s[i].Source.Name < s[j].Source.Name
			}
		}
		sort.Slice(vars.WorseLogs, cmp(vars.WorseLogs))
		sort.Slice(vars.BetterLogs, cmp(vars.BetterLogs))
		sort.Slice(vars.SameLogs, cmp(vars.SameLogs))
		sort.Slice(vars.BadLogs, cmp(vars.BadLogs))
		sort.Slice(vars.AllLogs, cmp(vars.AllLogs))

		err := sendEmail(dst, vars)

		// handle errors and retry setup
		if retryTimer.Stop() {
			<-retryTimer.C
		}

		if err == nil {
			failedAttempts = 0
			for _, log := range vars.BetterLogs {
				log.Sent = log.Read
				seen[log.Source.Name] = *log
			}
			for _, log := range vars.WorseLogs {
				log.Sent = log.Read
				seen[log.Source.Name] = *log
			}
			return
		}

		delay := time.Duration(float64(firstRetryDelay) * math.Pow(nextAttemptScale, float64(failedAttempts)))
		if delay > maxAttemptDelay {
			delay = maxAttemptDelay
		}
		failedAttempts++
		retryTimer.Reset(delay)
		errLogLimit.Do(func() {
			logger.Warn("failed to send email", zap.Error(err),
				zap.Int("failedAttempts", failedAttempts), zap.Duration("delay", delay),
			)
		})
	}

	for {
		select {
		case t := <-sendTicker.C:
			sendUpdates(t)
		case t := <-retryTimer.C:
			sendUpdates(t)
		case m, ok := <-c:
			if !ok {
				return
			}
			if m.log.Level == statuspb.StatusLog_LEVEL_UNDEFINED {
				continue // ignore undefined levels in an attempt to avoid noise
			}
			old := seen[m.source.Name]
			old.Read = m.log
			old.Source = m.source
			seen[m.source.Name] = old
		}
	}
}

func newStoppedTimer() *time.Timer {
	t := time.NewTimer(0)
	if !t.Stop() {
		<-t.C
	}
	return t
}

func sendEmail(dst config.Destination, attrs Attrs) error {
	p := dst.Parsed
	c, err := smtp.Dial(p.Addr)
	if err != nil {
		return err
	}

	err = c.StartTLS(&tls.Config{
		ServerName: dst.Host,
	})
	if err != nil {
		return err
	}

	err = c.Auth(smtp.PlainAuth("", p.Username, p.Password, dst.Host))
	if err != nil {
		return err
	}

	err = c.Mail(p.From.Address)
	if err != nil {
		return err
	}

	for _, to := range p.To {
		err = c.Rcpt(to.Address)
		if err != nil {
			return err
		}
	}

	wc, err := c.Data()
	if err != nil {
		return err
	}

	// write headers
	headers := make(textproto.MIMEHeader)
	headers.Add("From", p.From.String())
	for _, addr := range p.To {
		headers.Add("To", addr.String())
	}
	var subj strings.Builder
	if err := p.SubjectTemplate.Execute(&subj, attrs); err != nil {
		return err
	}
	headers.Add("Subject", subj.String())
	headers.Add("MIME-Version", `1.0`)
	headers.Add("Content-Type", mime.FormatMediaType("text/html", map[string]string{"charset": "utf-8"}))
	headers.Add("Content-Transfer-Encoding", `quoted-printable`)

	keys := maps.Keys(headers)
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range headers.Values(key) {
			_, err = fmt.Fprintf(wc, "%s: %s\r\n", key, value)
			if err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprint(wc, "\r\n"); err != nil {
		return err
	}

	// write body
	bodyWriter := quotedprintable.NewWriter(wc)
	if err := p.BodyTemplate.Execute(bodyWriter, attrs); err != nil {
		return err
	}
	if err := bodyWriter.Close(); err != nil {
		return err
	}

	// close email
	if err := wc.Close(); err != nil {
		return err
	}

	if err := c.Quit(); err != nil {
		return err
	}

	return nil
}
