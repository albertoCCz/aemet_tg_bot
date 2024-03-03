package main

import (
    "log"
    "time"
    "io"
    "fmt"
    "errors"
    "regexp"
    "strings"
    "strconv"
	"golang.org/x/net/html"
)

var MONTHS_ES = map[string]time.Month{"enero": time.January, "febrero": time.February, "marzo": time.March, "abril": time.April, "mayo": time.May, "junio": time.June, "julio": time.July, "agosto": time.August, "septiembre": time.September, "octubre": time.October, "noviembre": time.November, "diciembre": time.December}
const DATE_REGEXP = `[0-9]{1,2} de[l]{0,1} [a-z]{1,10} de[l]{0,1} [0-9]{3,4}|[0-9]{1,2} de [a-z]{1,10}`
const PDF_SIZE_REGEXP = `\([0-9]{1,3} KB\)`
const DATE_LAYOUT = "02/01/2006"

type PDF struct {
	Url string
	Name string
	Date string
}

func parsePDFDate(pdf *PDF) error {
	var re *regexp.Regexp = regexp.MustCompile(DATE_REGEXP)
	s := re.FindString(pdf.Date)
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
				log.Printf("Could not parse day '%s' as int\n", s_comp[0])
				parsing_ok = false
			}

			// month
			month_time, ok := MONTHS_ES[s_comp[1]]
			if !ok {
				log.Printf("Could not parse month '%s' as int\n", s_comp[1])
				parsing_ok = false
			}

			// year
			if n_comp == 2 {
				year_int = time.Now().Year()
			} else {
				year_int, err = strconv.Atoi(s_comp[2])
				if err != nil {
					log.Printf("Could not parse year '%s' as int\n", s_comp[2])
					parsing_ok = false
				}
			}

			if !parsing_ok {
				log.Printf("Could not parse date from date string '%s'.\n Leaving PDF date blank\n", s)
				pdf.Date = ""
				return errors.New(fmt.Sprintf("Date could not be parsed from '%s'", s))
			}

			pdf.Date = time.Date(year_int, month_time, day_int, 0, 0, 0, 0, time.UTC).Format(DATE_LAYOUT)
			return nil
		}
	}
	pdf.Date = ""
	return errors.New(fmt.Sprintf("Date could not be parsed. Regexp do not match with date string '%s'", pdf.Date))
}

func parsePDFName(pdf *PDF) {
	var re *regexp.Regexp = regexp.MustCompile(PDF_SIZE_REGEXP)
	if s := re.FindString(pdf.Name); len(s) > 0 {
		pdf.Name = strings.ReplaceAll(pdf.Name, s, "")
	}
}

func buildPDF(node *html.Node, a *html.Attribute) (PDF, error) {
	if node.FirstChild == nil { return PDF{}, errors.New("Node has no child") }

	var pdf PDF
	pdf.Url = a.Val
	pdf.Name = strings.TrimSpace(node.FirstChild.Data)
	parsePDFName(&pdf)

	if node.Parent != nil && node.Parent.Parent != nil {
		for sibling := node.Parent.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
			if sibling.Type == html.ElementNode && sibling.Data == "p" {
				pdf.Date = sibling.FirstChild.Data
				parsePDFDate(&pdf)
				break
			}
		}
	} else {
		pdf.Date = ""
	}

	return pdf, nil
}

func GenPDFs(r io.Reader, pdfs chan PDF) {
	node, err := html.Parse(r)
	if err != nil {
		log.Fatal("Could not parse Node")
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && strings.HasSuffix(a.Val, ".pdf") {
					pdf, err := buildPDF(n, &a)
					if err == nil {
						if len(pdf.Name) > 0 { pdfs <- pdf }
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
