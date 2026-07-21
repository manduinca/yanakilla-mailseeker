package mailparse

import (
	"bytes"
	"io"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Email struct {
	MessageID string `json:"message_id"`
	Date      string `json:"date"`
	From      string `json:"from"`
	To        string `json:"to"`
	Cc        string `json:"cc"`
	Bcc       string `json:"bcc"`
	Subject   string `json:"subject"`
	Folder    string `json:"folder"`
	Content   string `json:"content"`
}

func ParseFile(path, root string) (*Email, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data, folderOf(path, root))
}

func Parse(r io.Reader, folder string) (*Email, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data, folder)
}

func ParseBytes(data []byte, folder string) (*Email, error) {
	email, err := decode(data, folder)
	if err == nil {
		return email, nil
	}

	repaired, changed := foldLooseHeaders(data)
	if !changed {
		return nil, err
	}
	return decode(repaired, folder)
}

func decode(data []byte, folder string) (*Email, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}

	return &Email{
		MessageID: msg.Header.Get("Message-ID"),
		Date:      msg.Header.Get("Date"),
		From:      strings.TrimSpace(msg.Header.Get("From")),
		To:        normalizeList(msg.Header.Get("To")),
		Cc:        normalizeList(msg.Header.Get("Cc")),
		Bcc:       normalizeList(msg.Header.Get("Bcc")),
		Subject:   strings.TrimSpace(msg.Header.Get("Subject")),
		Folder:    folder,
		Content:   string(body),
	}, nil
}

func foldLooseHeaders(data []byte) ([]byte, bool) {
	var out bytes.Buffer
	out.Grow(len(data) + 64)

	changed := false
	rest := data

	for len(rest) > 0 {
		var line []byte
		if idx := bytes.IndexByte(rest, '\n'); idx < 0 {
			line, rest = rest, nil
		} else {
			line, rest = rest[:idx+1], rest[idx+1:]
		}

		trimmed := bytes.TrimRight(line, "\r\n")

		if len(trimmed) == 0 {
			out.Write(line)
			out.Write(rest)
			return out.Bytes(), changed
		}

		if trimmed[0] == ' ' || trimmed[0] == '\t' || isHeaderStart(trimmed) {
			out.Write(line)
			continue
		}

		out.WriteByte(' ')
		out.Write(line)
		changed = true
	}

	return out.Bytes(), changed
}

func isHeaderStart(line []byte) bool {
	colon := bytes.IndexByte(line, ':')
	if colon <= 0 {
		return false
	}
	for _, c := range line[:colon] {
		if c <= ' ' || c >= 0x7f {
			return false
		}
	}
	return true
}

func folderOf(path, root string) string {
	if root == "" {
		return FolderOfEntry(filepath.ToSlash(path))
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return ""
	}
	return filepath.ToSlash(dir)
}

func FolderOfEntry(name string) string {
	dir := path.Dir(name)
	if dir == "." || dir == "/" {
		return ""
	}
	return dir
}

func normalizeList(v string) string {
	if v == "" {
		return ""
	}
	fields := strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if s := strings.TrimSpace(f); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, ", ")
}
