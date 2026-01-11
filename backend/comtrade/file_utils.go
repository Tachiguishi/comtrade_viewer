package comtrade

import (
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func detectEncoding(f io.Reader) (string, error) {
	buf := make([]byte, 128)
	n, _ := f.Read(buf)
	sample := buf[:n]

	if utf8.Valid(sample) {
		return "UTF-8", nil
	}
	return "GBK", nil
}

func transformToUTF8(f io.Reader, encoding string) (io.Reader, error) {
	// reset f to the begining
	if seeker, ok := f.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	switch encoding {
	case "UTF-8":
		return f, nil
	case "GBK":
		return transform.NewReader(f, simplifiedchinese.GBK.NewDecoder()), nil
	default:
		return nil, nil
	}
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
