package notificationsemail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/textproto"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/notificationsemail/config"
)

// sendEmail creates the MIME/multipart email with attachment and then sends it over SMTP to the recipient in  dst
//
//goland:noinspection GoUnhandledErrorResult
func sendEmail(dst config.Destination, attachment config.AttachmentCfg, subj string, templateArgs config.TemplateArgs, logger *zap.Logger) error {
	buf := bytes.NewBuffer(nil)
	withAttachments := len(attachment.Attachment) > 0
	p := dst.Parsed

	// Subject
	fmt.Fprintf(buf, "Subject: %s\n", subj)

	// From
	fmt.Fprintf(buf, "From: %s\n", p.From.String())

	// To
	var addrs []string
	for _, a := range p.To {
		addrs = append(addrs, a.Address)
		fmt.Fprintf(buf, "To: %s\n", a.String())
	}

	fmt.Fprintf(buf, "MIME-Version: 1.0\n")

	// Main message
	mainMessageWriter := multipart.NewWriter(buf)
	mainMessageBoundary := mainMessageWriter.Boundary()
	fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=\"%s\"\n", mainMessageBoundary)

	// Create a part for the attachment
	attachmentWriter := multipart.NewWriter(buf)
	attachmentBoundaryString := attachmentWriter.Boundary()
	attachmentHeader := make(textproto.MIMEHeader)
	attachmentHeader.Set("Content-Type", fmt.Sprintf("multipart/related; boundary=\"%s\"\n", attachmentBoundaryString))
	if _, err := mainMessageWriter.CreatePart(attachmentHeader); err != nil {
		return err
	}

	// Create a part for the HTML email
	htmlWriter := multipart.NewWriter(buf)
	htmlMessageBoundary := htmlWriter.Boundary()
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=\"%s\"\n", htmlMessageBoundary))
	if _, err := attachmentWriter.CreatePart(htmlHeader); err != nil {
		return err
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	partWriter, _ := htmlWriter.CreatePart(header)
	if err := p.BodyTemplate.Execute(partWriter, templateArgs); err != nil {
		return err
	}
	htmlWriter.Close()

	if withAttachments {
		v := attachment.Attachment
		k := attachment.AttachmentName
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", http.DetectContentType(v))
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-Disposition", "attachment; filename="+k)
		partWriter, err := attachmentWriter.CreatePart(header)
		if err != nil {
			return err
		}
		b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
		base64.StdEncoding.Encode(b, v)
		partWriter.Write(b)
		partWriter.Write([]byte("\n"))
		attachmentWriter.Close()

	}

	mainMessageWriter.Close()

	auth := smtp.PlainAuth("", p.Username, p.Password, dst.Host)
	addr := dst.Addr()
	logger.Debug("Sending mail ", zap.String("addr", addr))
	err := smtp.SendMail(addr, auth, p.Username, addrs, buf.Bytes())

	if err != nil {
		logger.Error("Error sending email", zap.Error(err))
		return err
	}

	return nil
}
