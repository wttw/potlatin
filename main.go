package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/SeanMcGoff/piglatin"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/html"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	htmlIgnore  = "ignore"
	htmlRequire = "require"
	htmlAttempt = "attempt"
)

var htmlSupport string

func main() {
	var outfile string
	flags := flag.NewFlagSet("potlatin", flag.ContinueOnError)
	flags.StringVar(&htmlSupport, "html", "require", "How to handle HTML in translations (ignore, attempt, require)")
	flags.StringVarP(&outfile, "output", "o", "", "Write output to file")
	err := flags.Parse(os.Args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalln(err)
	}
	if len(flags.Args()) != 2 {
		log.Fatalln("One .pot file required as a parameter")
	}
	filename := flags.Arg(1)

	in, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open %s: %s", filename, err)
	}

	var buff bytes.Buffer

	err = process(in, &buff, pig)
	if err != nil {
		log.Fatalln(err)
	}
	var out io.Writer
	switch outfile {
	case "-":
		out = os.Stdout
	case "":
		out, err = os.Create("x-piglatin.po")
		if err != nil {
			log.Fatalln(err)
		}
	default:
		out, err = os.Create(outfile)
		if err != nil {
			log.Fatalln(err)
		}
	}
	_, err = out.Write(buff.Bytes())
	if err != nil {
		log.Fatalln(err)
	}
}

func process(r io.Reader, w io.Writer, fn func(msgctx, msgid string) (string, error)) error {
	scanner := bufio.NewScanner(r)
	inmsgId := false
	var msgctx string
	var msgid []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgstr") {
			inmsgId = false
			_, _ = io.WriteString(w, "msgstr ")
			src := strings.ReplaceAll(strings.Join(msgid, ""), `\n`, "\n")
			translated, err := translate(msgctx, src, fn)
			if err != nil {
				return fmt.Errorf("failed to translate '%s': %w", src, err)
			}
			lines := strings.Split(translated, "\n")
			for i, line := range lines {
				if i != len(lines)-1 {
					line += `\n`
				}
				_, _ = fmt.Fprintf(w, "%q\n", line)
			}
			continue
		}
		if !inmsgId {
			if strings.HasPrefix(line, `"PO-Revision-Date:`) {
				fmt.Fprintf(w, "\"PO-Revision-Date: %s\\n\"\n", time.Now().Format("2006-01-02 15:04-0700"))
				continue
			}
		}
		_, _ = fmt.Fprintf(w, "%s\n", line)
		if strings.HasPrefix(line, "msgid ") {
			inmsgId = true
			msgid = nil
			var s string
			_, err := fmt.Sscanf(line, "msgid %q", &s)
			if err != nil {
				return fmt.Errorf("failed to parse '%s': %w", line, err)
			}
			msgid = append(msgid, s)
		}
		if strings.HasPrefix(line, `""`) {
			if inmsgId {
				var s string
				_, err := fmt.Sscanf(line, "%q", &s)
				if err != nil {
					return fmt.Errorf("failed to parse '%s': %w", line, err)
				}
				msgid = append(msgid, s)
			}
		}
	}
	return nil
}

func translate(msgctx, in string, fn func(msgctx, msgid string) (string, error)) (string, error) {
	switch htmlSupport {
	case htmlIgnore:
		return fn(msgctx, in)
	case htmlRequire:
		return fromHtml(msgctx, in, fn)
	case htmlAttempt, "":
		s, err := fromHtml(msgctx, in, fn)
		if err == nil {
			return s, nil
		}
		log.Printf("failed to translate html '%s': %s, continuing", in, err.Error())
		return fn(msgctx, in)
	default:
		log.Fatalf("invalid html style: %s", htmlSupport)
	}
	return "i can't happen", nil
}

func fromHtml(msgctx, in string, fn func(msgctx, msgid string) (string, error)) (string, error) {
	if !strings.Contains(in, "<") || !strings.Contains(in, ">") {
		s, err := fn(msgctx, in)
		return s, err
	}
	var ret strings.Builder
	tokenizer := html.NewTokenizer(strings.NewReader(in))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if !errors.Is(tokenizer.Err(), io.EOF) {
				return "", fmt.Errorf("failed to tokenize HTML: %w", tokenizer.Err())
			}
			return ret.String(), nil
		case html.TextToken:
			tok := tokenizer.Token()
			s, err := fn(msgctx, tok.Data)
			if err != nil {
				return "", fmt.Errorf("failed to translate '%s': %w", tok.Data, err)
			}
			tok.Data = s
			_, _ = io.WriteString(&ret, tok.String())
		default:
			_, _ = io.WriteString(&ret, tokenizer.Token().String())
		}
	}
}

var alpha = regexp.MustCompile(`[a-zA-Z]+`)

func pig(_, msgid string) (string, error) {
	matches := alpha.FindAllStringIndex(msgid, -1)
	if matches == nil {
		return msgid, nil
	}
	var ret []string
	mark := 0
	for _, match := range matches {
		ret = append(ret, msgid[mark:match[0]])
		word := msgid[match[0]:match[1]]
		text, err := piglatin.ToPigLatin(word)
		if err != nil {
			return "", fmt.Errorf("failed to pigify '%s': %w", word, err)
		}
		ret = append(ret, text)
		mark = match[1]
	}
	ret = append(ret, msgid[mark:])
	return strings.Join(ret, ""), nil
}
