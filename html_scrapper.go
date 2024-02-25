package main

import (
	"fmt"
	"log"
	"io"
	"errors"
	"strings"
	"strconv"
	"regexp"
	"time"
	"net/http"
	"golang.org/x/net/html"
	// tele "gopkg.in/telebot.v3"
)

var (
	MONTHS_ES = map[string]time.Month{"enero": time.January, "febrero": time.February, "marzo": time.March, "abril": time.April, "mayo": time.May, "junio": time.June, "julio": time.July, "agosto": time.August, "septiembre": time.September, "octubre": time.October, "noviembre": time.November, "diciembre": time.December}
	DEFAULT_DATE = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
)

const(
	url = "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022"
	PARSING_FINISHED = "PARSING_FINISHED"
	DATE_REGEXP = `[0-9]{1,2} de[l]{0,1} [a-z]{1,10} de[l]{0,1} [0-9]{3,4}|[0-9]{1,2} de [a-z]{1,10}`
	DATE_LAYOUT = "02/01/2006"
)

type PDF struct {
	url string
	name string
	date string
}

func parse_pdf_date(pdf *PDF) error {
	var re *regexp.Regexp = regexp.MustCompile(DATE_REGEXP)
	s := re.FindString(pdf.date)
	if len(s) > 0 {
		if strings.Contains(s, "del") {
			s = strings.ReplaceAll(s, "del", "de")
		}

		s_comp := strings.Split(s, " de ")
		if n_comp := len(s_comp); n_comp == 3 || n_comp == 2 {
			var day_int, year_int int = 0, 0
			var err error = nil
			var parsing_ok bool = true

			// day
			day_int, err = strconv.Atoi(s_comp[0])
			if err != nil {
				log.Printf("Could not parse day '%s' as int", s_comp[0])
				parsing_ok = false
			}

			// month
			month_time, ok := MONTHS_ES[s_comp[1]]
			if !ok {
				log.Printf("Could not parse month '%s' as int", s_comp[1])
				parsing_ok = false
			}

			// year
			if n_comp == 2 {
				year_int = time.Now().Year()
			} else {
				year_int, err = strconv.Atoi(s_comp[2])
				if err != nil {
					log.Printf("Could not parse year '%s' as int", s_comp[2])
					parsing_ok = false
				}
			}

			if !parsing_ok {
				log.Printf("Could not parse date from date string '%s'.\n Setting default date: %v", s, DEFAULT_DATE)
				pdf.date = DEFAULT_DATE.Format(DATE_LAYOUT)
				return errors.New(fmt.Sprintf("Date could not be parsed from '%s'", s))
			}
			pdf.date = time.Date(year_int, month_time, day_int, 0, 0, 0, 0, time.UTC).Format(DATE_LAYOUT)
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Date could not be parsed. Regexp do not match with date string '%s'", pdf.date))
}

func build_pdf(node *html.Node, a *html.Attribute) (PDF, error) {
	var pdf PDF

	if node.FirstChild == nil { return PDF{}, errors.New("Node has no child") }

	pdf.url = a.Val
	pdf.name = strings.TrimSpace(node.FirstChild.Data)

	if node.Parent != nil && node.Parent.Parent != nil {
		for sibling := node.Parent.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
			if sibling.Type == html.ElementNode && sibling.Data == "p" {
				pdf.date = sibling.FirstChild.Data
				parse_pdf_date(&pdf)
				break
			}
		}
	} else {
		pdf.date = ""
	}

	return pdf, nil
}

func gen_pdfs(r io.Reader, pdfs chan PDF) {
	node, err := html.Parse(r)
	if err != nil {
		log.Fatal("Could not parse Node")
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && strings.HasSuffix(a.Val, ".pdf") {
					pdf, err := build_pdf(n, &a)
					if err == nil {
						if len(pdf.name) > 0 { pdfs <- pdf }
					}
					break
				}
			}
		}
		for child_i := n.FirstChild; child_i != nil; child_i = child_i.NextSibling {
			f(child_i)
		}
	}
	f(node)
	close(pdfs)
	return
}

func main() {
	var client *http.Client = &http.Client{}
	res, err := client.Get(url)
	if err != nil && res.StatusCode == 200 {
		log.Fatalf("Something went wrong getting url. StatusCode: %d", res.StatusCode)
	}

	ch := make(chan PDF)
	go gen_pdfs(res.Body, ch)

	// start pooling for pdfs
	pdfs := make([]PDF, 50, 100)
	count := 0
	for pdf, ok := <-ch; ok; pdf, ok = <-ch {
		if count > cap(pdfs) {
			res.Body.Close()
			log.Fatalf("Too many pdfs found. Increase buffer capacity")

		}
		pdfs[count] = pdf
		count++
	}
	res.Body.Close()


	for i, pdf := range pdfs {
		if len(pdf.name) == 0 { break }
		fmt.Printf("(%d) new pdf:\n\tname: %s\n\turl: %s\n\tdate: %s\n\n", i, pdf.name, pdf.url, pdf.date)
	}

	log.Println("Finished successfully!")
}
