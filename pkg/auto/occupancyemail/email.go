package occupancyemail

import (
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/smart-core-os/sc-bos/pkg/auto/occupancyemail/config"
)

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
