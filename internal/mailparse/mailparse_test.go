package mailparse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `Message-ID: <18782981.1075855378110.JavaMail.evans@thyme>
Date: Mon, 14 May 2001 16:39:00 -0700 (PDT)
From: phillip.allen@enron.com
To: tim.belden@enron.com, john.lavorato@enron.com
Cc: greg.whalley@enron.com
Subject: Reporte de posicion
Mime-Version: 1.0
Content-Type: text/plain; charset=us-ascii

Here is our forecast
`

func TestParseExtraeCabecerasYCuerpo(t *testing.T) {
	email, err := Parse(strings.NewReader(sample), "allen-p/_sent_mail")
	if err != nil {
		t.Fatalf("Parse devolvio error: %v", err)
	}

	if email.From != "phillip.allen@enron.com" {
		t.Errorf("From = %q", email.From)
	}
	if email.Subject != "Reporte de posicion" {
		t.Errorf("Subject = %q", email.Subject)
	}
	if email.Folder != "allen-p/_sent_mail" {
		t.Errorf("Folder = %q", email.Folder)
	}
	if !strings.Contains(email.Content, "Here is our forecast") {
		t.Errorf("Content = %q", email.Content)
	}
	if email.MessageID == "" {
		t.Error("MessageID quedo vacio")
	}
}

func TestParseNormalizaDestinatarios(t *testing.T) {
	email, err := Parse(strings.NewReader(sample), "")
	if err != nil {
		t.Fatalf("Parse devolvio error: %v", err)
	}

	want := "tim.belden@enron.com, john.lavorato@enron.com"
	if email.To != want {
		t.Errorf("To = %q, se esperaba %q", email.To, want)
	}
	if email.Cc != "greg.whalley@enron.com" {
		t.Errorf("Cc = %q", email.Cc)
	}
	if email.Bcc != "" {
		t.Errorf("Bcc = %q, se esperaba vacio", email.Bcc)
	}
}

func TestParseDestinatariosEnVariasLineas(t *testing.T) {
	raw := "From: a@b.com\r\nTo: uno@enron.com,\r\n\tdos@enron.com,\r\n\ttres@enron.com\r\nSubject: x\r\n\r\ncuerpo\r\n"

	email, err := Parse(strings.NewReader(raw), "")
	if err != nil {
		t.Fatalf("Parse devolvio error: %v", err)
	}

	want := "uno@enron.com, dos@enron.com, tres@enron.com"
	if email.To != want {
		t.Errorf("To = %q, se esperaba %q", email.To, want)
	}
}

func TestParseRecuperaAsuntoMultilineaSinIndentar(t *testing.T) {
	raw := "Message-ID: <1@thyme>\nDate: Thu, 13 Dec 2001 06:39:18 -0800 (PST)\nFrom: don.baughman@enron.com\nSubject: Call Laddie for house party:\n1. Mom &dad\n2. Troy & Andrea\nMime-Version: 1.0\n\nel cuerpo queda intacto\n"

	email, err := Parse(strings.NewReader(raw), "baughman-d/tasks")
	if err != nil {
		t.Fatalf("Parse devolvio error: %v", err)
	}

	for _, frag := range []string{"Call Laddie for house party:", "1. Mom &dad", "2. Troy & Andrea"} {
		if !strings.Contains(email.Subject, frag) {
			t.Errorf("Subject = %q, no contiene %q", email.Subject, frag)
		}
	}
	if email.From != "don.baughman@enron.com" {
		t.Errorf("From = %q", email.From)
	}
	if strings.TrimSpace(email.Content) != "el cuerpo queda intacto" {
		t.Errorf("Content = %q", email.Content)
	}
}

func TestParseNoAlteraCorreosValidos(t *testing.T) {
	email, err := Parse(strings.NewReader(sample), "")
	if err != nil {
		t.Fatalf("Parse devolvio error: %v", err)
	}
	if strings.HasPrefix(email.Content, " ") {
		t.Errorf("el cuerpo fue modificado: %q", email.Content)
	}
	if email.Subject != "Reporte de posicion" {
		t.Errorf("Subject = %q", email.Subject)
	}
}

func TestParseRechazaEntradaInvalida(t *testing.T) {
	if _, err := Parse(strings.NewReader("esto no es un correo"), ""); err == nil {
		t.Error("se esperaba error con entrada sin cabeceras")
	}
}

func TestParseFileDerivaCarpetaDeLaRuta(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "allen-p", "inbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "1")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	email, err := ParseFile(path, root)
	if err != nil {
		t.Fatalf("ParseFile devolvio error: %v", err)
	}
	if email.Folder != "allen-p/inbox" {
		t.Errorf("Folder = %q, se esperaba allen-p/inbox", email.Folder)
	}
}

func TestFolderOfEntry(t *testing.T) {
	cases := map[string]string{
		"allen-p/_sent_mail/1.": "allen-p/_sent_mail",
		"allen-p/inbox/23":      "allen-p/inbox",
		"suelto":                "",
	}

	for in, want := range cases {
		if got := FolderOfEntry(in); got != want {
			t.Errorf("FolderOfEntry(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}
