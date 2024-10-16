package email

import (
	"io"

	"github.com/mnako/letters"
	"github.com/pkg/errors"
)

type SkylineEmail struct {
	letters.Email
}

func Parse(r io.Reader) (*SkylineEmail, error) {
	e, err := letters.ParseEmail(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse incoming SMTP message")
	}

	return &SkylineEmail{e}, nil
}

func (s *SkylineEmail) IsHTML() bool {
	return s.Headers.ContentType.ContentType == "text/html"
}

func (s *SkylineEmail) IsPlaintext() bool {
	return s.Headers.ContentType.ContentType == "text/plain"
}

func (s *SkylineEmail) IsMultiPartAlternative() bool {
	return s.Headers.ContentType.ContentType == "multipart/alternative"
}
