package main

import (
	"fmt"
	"log"
	"io"
	"errors"
	"strings"
	"regexp"
	"net/http"
	"golang.org/x/net/html"
)

const(
	url = "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022"
	PARSING_FINISHED = "PARSING_FINISHED"
	DATE_REGEXP = `[0-9]{1,2} de [a-z]{1,10} de[l]{0,1} [0-9]{3,4}|[0-9]{1,2} de [a-z]{1,10}`
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

		pdf.date = "1995-10-29"
		return nil
	}
	return errors.New("Date could not be parsed")
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

func get_pdfs(r io.Reader, pdfs chan PDF) {
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

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)
	pdfs <- PDF{url: PARSING_FINISHED, name: "", date: ""}
	return
}

func main() {
	var client *http.Client = &http.Client{}
	res, err := client.Get(url)
	if err != nil && res.StatusCode == 200 {
		log.Fatalf("Something went wrong getting url. StatusCode: %d", res.StatusCode)
	}

	// parse body
	var pdfs = make(chan PDF)

	go get_pdfs(res.Body, pdfs)
	for {
		pdf := <-pdfs
		if pdf.url == PARSING_FINISHED { break }
		fmt.Printf("new pdf:\n\tname: %s\n\turl: %s\n\tdate: %s\n\n", pdf.name, pdf.url, pdf.date)
	}
	
	res.Body.Close()

	log.Println("Finished successfully!")
}
