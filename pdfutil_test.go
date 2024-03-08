package main

import	(
	"testing"
	"fmt"
	"strings"
	"time"
)


var errFmtString = "want: '%s'; got: '%s'\n"

func TestParsePDFDate(t *testing.T) {
	thisYear := fmt.Sprintf("%d", time.Now().Year())
	want := "02/11/" + thisYear

	pdf := PDF{Date: fmt.Sprintf("Fecha de publicación: 2 de noviembre de %s some other text", thisYear)}
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}

	pdf.Date = fmt.Sprintf("   Fecha de publicación: 2 de noviembre de %s some other text  ", thisYear)
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}

	pdf.Date = fmt.Sprintf("   Fecha de publicación: 2 de noviembre del %s some other text  ", thisYear)
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}

	pdf.Date = fmt.Sprintf("   Fecha de publicación: 2 de noviembre de %ssome other text  ", thisYear)
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}

	pdf.Date = fmt.Sprintf("   Fecha de publicación: 2de noviembre del%ssome other text  ", thisYear)
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}

	pdf.Date = "   Fecha de publicación: 2de noviembre other text  "
	parsePDFDate(&pdf)
	if pdf.Date != want {
		t.Errorf(errFmtString, want, pdf.Date)
	}
}

func BenchmarkParsePDFDate(b *testing.B) {
	pdf := PDF{Date: "Fecha de publicación: 2 de noviembre de 2030 some other text"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsePDFDate(&pdf)
	}
}

func TestParsePDFName(t *testing.T) {
    want := "my pdf name"

	pdf := PDF{Name: "my pdf name"}
	parsePDFName(&pdf)
	if pdf.Name != want {
		t.Errorf(errFmtString, want, pdf.Name)
	}

	pdf.Name = "my pdf name (234KB)"
	parsePDFName(&pdf)
	if pdf.Name != want {
		t.Errorf(errFmtString, want, pdf.Name)
	}

	pdf.Name = "my pdf name (234 KB)"
	parsePDFName(&pdf)
	if pdf.Name != want {
		t.Errorf(errFmtString, want, pdf.Name)
	}

	pdf.Name = " my pdf name (234 KB) "
	parsePDFName(&pdf)
	if pdf.Name != want {
		t.Errorf(errFmtString, want, pdf.Name)
	}
}

func TestGenPDFs(t *testing.T) {
	t.Run("CorrectHTMLStruct", func (t *testing.T) {
		html := "<div class=\"disclaimer\">" +
			"<a href=\"some pdf url.pdf\" target=\"_blank\"> <img alt=\"TO DO xalan:\" src=\"/imagenes_gcd/_iconos_docs_adjuntos/pdf.gif\" title=\"TO DO xalan:\"></a>" +
			"<span> <a href=\"some pdf url.pdf\" target=\"_blank\">some pdf name (757 KB)</a></span>" +
			"<p>Fecha de publicación:14 de junio de 2023</p></div>"

		want := PDF{
			Name: "some pdf name",
			Url: "some pdf url.pdf",
			Date: "14/06/2023",
		}

		r := strings.NewReader(html)
		c := make(chan PDF)
		go GenPDFs(r, c)

		pdf := <-c
		if pdf.Name != want.Name {
			t.Errorf(errFmtString, want.Name, pdf.Name)
		}

		if pdf.Url != want.Url {
			t.Errorf(errFmtString, want.Url, pdf.Url)
		}

		if pdf.Date != want.Date {
			t.Errorf(errFmtString, want.Date, pdf.Date)
		}
	})

		t.Run("NoDateHTMLStruct", func (t *testing.T) {
			html := "<div class=\"disclaimer\">" +
				"<a href=\"some pdf url.pdf\" target=\"_blank\"> <img alt=\"TO DO xalan:\" src=\"/imagenes_gcd/_iconos_docs_adjuntos/pdf.gif\" title=\"TO DO xalan:\"></a>" +
				"<span> <a href=\"some pdf url.pdf\" target=\"_blank\">some pdf name (757 KB)</a></span>" +
				"</div>"

			want := PDF{
				Name: "some pdf name",
				Url: "some pdf url.pdf",
				Date: "",
			}

			r := strings.NewReader(html)
			c := make(chan PDF)
			go GenPDFs(r, c)

			pdf := <-c
			if pdf.Name != want.Name {
				t.Errorf(errFmtString, want.Name, pdf.Name)
			}

			if pdf.Url != want.Url {
				t.Errorf(errFmtString, want.Url, pdf.Url)
			}

			if pdf.Date != want.Date {
				t.Errorf(errFmtString, want.Date, pdf.Date)
			}
		})

}
