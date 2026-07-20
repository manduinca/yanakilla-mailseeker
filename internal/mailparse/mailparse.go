package mailparse

import (
	"io"
	"net/mail"
	"os"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	msg, err := mail.ReadMessage(f)
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
		Folder:    folderOf(path, root),
		Content:   string(body),
	}, nil
}

func folderOf(path, root string) string {
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
