package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/gopasspw/gopass/pkg/gopass/api"
)

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func pop(m map[string]string, key string) (v string) {
	var exists bool
	if v, exists = m[key]; exists {
		delete(m, key)
	}
	return
}

func printfield(name, value string) {
	fmt.Printf("\t%20s: %s\n", name, value)
}

type row struct {
	title        string
	username     string
	email        string
	password     string
	url          string
	totp         string
	backup_codes string
	seed         string
	body         string
	other        map[string]string
}

type writer interface {
	WriteRow(r row)
	Flush()
}

type csvWriter struct {
	w *csv.Writer
}

func newCsvWriter() csvWriter {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"Title", "Username", "Email", "Password", "Website", "TOTP Secret Key", "*Backup Codes", "*Seed", "*Comment", "*Body"})
	return csvWriter{w: w}
}

func (w csvWriter) WriteRow(r row) {
	var comments []string
	for k, v := range r.other {
		comments = append(comments, fmt.Sprintf("%s: %s", k, v))
	}
	w.w.Write([]string{r.title, r.username, r.email, r.password, r.url, r.totp, r.backup_codes, r.seed, strings.Join(comments, "\n"), r.body})
}

func (w csvWriter) Flush() {
	w.w.Flush()
}

type textWriter struct{}

func (w textWriter) WriteRow(r row) {
	fmt.Printf("%s\n", r.title)
	printfield("url", r.url)
	printfield("password", r.password)
	printfield("username", r.username)
	printfield("totp", r.totp)
	printfield("email", r.email)
	printfield("seed", r.seed)
	printfield("backup_codes", r.backup_codes)
	if r.body != "" {
		printfield("body", r.body)
	}
	for k, v := range r.other {
		printfield(k, v)
	}
}

func (w textWriter) Flush() {
}

func main() {
	ctx := context.TODO()
	gp := must(api.New(ctx))
	ls := must(gp.List(ctx))
	var w writer
	if os.Getenv("CSV") != "" {
		w = newCsvWriter()
	} else {
		w = textWriter{}
	}
	defer w.Flush()
	for _, sName := range ls {
		if !strings.HasPrefix(sName, "browser/") {
			continue
		}
		sec := must(gp.Get(ctx, sName, "latest"))
		values := make(map[string]string)
		for _, k := range sec.Keys() {
			v, _ := sec.Get(k)
			switch k {
			case "comments":
				continue
			case "icon":
				continue
			case "autotype_enabled":
				continue
			default:
				values[k] = v
			}
		}
		r := row{}
		r.title = sName[len("browser/"):]
		r.url = pop(values, "url")
		r.password = sec.Password()

		r.username = pop(values, "login")
		if r.username == "" {
			r.username = pop(values, "user")
			if r.username == "" {
				r.username = pop(values, "username")
			}
		}
		r.totp = pop(values, "totp")
		if r.totp == "" {
			body := sec.Body()
			if strings.Contains(body, "totp") || strings.Contains(body, "otpauth") {
				r.body = body
			}
		}
		r.email = pop(values, "email")
		r.backup_codes = pop(values, "backup_codes")
		r.other = values
		w.WriteRow(r)
	}
}
