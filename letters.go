package letters

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"

	"golang.org/x/net/html/charset"
)

func ParseEmail(r io.Reader, opts ...ParseOption) (Email, error) {
	var options ParseOptions
	for _, opt := range opts {
		opt(&options)
	}

	var email Email

	msg, err := mail.ReadMessage(r)
	if err != nil {
		return email, fmt.Errorf("letters.ParseEmail: cannot read message: %w", err)
	}

	headers, err := parseHeaders(msg.Header)
	if err != nil {
		return email, fmt.Errorf("letters.ParseEmail: cannot parse headers: %w", err)
	}

	email = Email{
		Headers: headers,
	}
	encoding, _ := charset.Lookup(email.Headers.ContentType.Params["charset"])
	cte, err := parseContentTransferEncoding(msg.Header.Get("Content-Transfer-Encoding"))
	if err != nil {
		return email, fmt.Errorf("letters.ParseEmail: cannot parse Content-Transfer-Encoding: %w", err)
	}

	if email.Headers.ContentType.ContentType == contentTypeTextPlain {
		email.Text, err = parseText(msg.Body, encoding, cte)
		if err != nil {
			return email, fmt.Errorf("letters.ParseEmail: cannot parse plain text: %w", err)
		}

	} else if email.Headers.ContentType.ContentType == contentTypeTextEnriched {
		email.EnrichedText, err = parseText(msg.Body, encoding, cte)
		if err != nil {
			return email, fmt.Errorf("letters.ParseEmail: cannot parse enriched text: %w", err)
		}

	} else if email.Headers.ContentType.ContentType == contentTypeTextHtml {
		email.HTML, err = parseText(msg.Body, encoding, cte)
		if err != nil {
			return email, fmt.Errorf("letters.ParseEmail: cannot parse html text: %w", err)
		}

	} else if strings.HasPrefix(email.Headers.ContentType.ContentType, contentTypeMultipartPrefix) {
		boundary := email.Headers.ContentType.Params["boundary"]
		emailBodies, err := parsePart(msg.Body, email.Headers.ContentType, boundary, opts...)
		if err != nil {
			return email, fmt.Errorf(
				"letters.ParseEmail: cannot parse part %q with boundary %q: %w",
				email.Headers.ContentType.ContentType,
				boundary,
				err)
		}
		email.Text = emailBodies.text
		email.EnrichedText = emailBodies.enrichedText
		email.HTML = emailBodies.html
		email.InlineFiles = emailBodies.InlineFiles
		email.AttachedFiles = emailBodies.AttachedFiles

	} else {
		if !options.SkipAttachments {
			afl, err := decodeAttachmentFileFromBody(msg.Body, email.Headers, cte)
			if err != nil {
				return email, fmt.Errorf(
					"letters.decoders.ParseEmail: cannot decode attached file content from body: %w",
					err)
			}
			email.AttachedFiles = append(email.AttachedFiles, afl)
		}
	}

	email.Text = normalizeMultilineString(email.Text)
	email.EnrichedText = normalizeMultilineString(email.EnrichedText)
	email.HTML = normalizeMultilineString(email.HTML)

	return email, nil
}

func StripEmailAttachments(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer

	echoReader := io.TeeReader(r, &buf)

	_, err := ParseEmail(echoReader, WithParseSkipAttachments())
	if err != nil {
		return nil, fmt.Errorf("letters.StripEmailAttachments: cannot parse email: %w", err)
	}

	return buf.Bytes(), nil
}
