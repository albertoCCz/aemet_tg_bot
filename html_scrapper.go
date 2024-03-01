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
	"os"
	"net/http"
	"encoding/json"
	"golang.org/x/net/html"
	tele "gopkg.in/telebot.v3"
)

// main structs. The herarchy is:
// botConfig
//   |_ ...
//   |_ chatAdminConfig
//   |_ []chatConfig
//     |_ ...
//     |_ []selectiveProc


type selectiveProc struct {
	name         string
	templatePath string
	registryPath string
	url          string
}

type chatConfig struct {
	chatId         string
	name           string
	selectiveProcs []selectiveProc
}

func (c *chatConfig) Recipient() string {
	return c.chatId
}

type chatAdminConfig struct {
	chatId           string
	name             string
	errMessageFormat string
}

func (c *chatAdminConfig) Recipient() string {
	return c.chatId
}

type botConfig struct {
	token           string
	timeInterval    time.Duration
	chatAdminConfig *chatAdminConfig
	chatConfigs     []chatConfig
}

const (
	DATE_REGEXP = `[0-9]{1,2} de[l]{0,1} [a-z]{1,10} de[l]{0,1} [0-9]{3,4}|[0-9]{1,2} de [a-z]{1,10}`
	DATE_LAYOUT = "02/01/2006"
)

var bot_config = botConfig{
	token: os.Getenv("TG_AEMET_TOKEN"),
	timeInterval: 5 * time.Second,
	chatAdminConfig: &chatAdminConfig{
		chatId: os.Getenv("CHAT_ADMIN"),
		name: "Admin chat",
		errMessageFormat: "Error: <strong>%s</strong>\n" +
			"  - chat name: <i>%s</i>\n" +
			"  - proc name: <i>%s</i>\n" +
			"  - pdf name:  <i>%s</i>\n" +
			"  - message:   <pre language=\"console\">%s</pre>\n",
	},
	chatConfigs: []chatConfig{
		chatConfig{
			chatId: os.Getenv("CHAT_ID_TEST_1"),
			name: "CHAT-1",
			selectiveProcs: []selectiveProc{
				selectiveProc{
					name:          "Test1",
					templatePath: "./templates/template_fmt.txt",
					registryPath:  "./pdfs-registry/pdfs-chat1-test1.json",
					url:           "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022",
				},
				selectiveProc{
					name:          "Test2",
					templatePath: "./templates/template_fmt.txt",
					registryPath:  "./pdfs-registry/pdfs-chat1-test2.json",
					url:           "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/promocion_interna/acceso_interna_2021_2022",
				},
			},
		},

		chatConfig{
			chatId: os.Getenv("CHAT_ID_TEST_2"),
			name: "CHAT-2",
			selectiveProcs: []selectiveProc{
				selectiveProc{
					name:          "Test1",
					templatePath: "./templates/template_fmt.txt",
					registryPath:  "./pdfs-registry/pdfs-chat2-test1.json",
					url:           "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022",
				},
				selectiveProc{
					name:          "Test2",
					templatePath: "./templates/template_fmt.txt",
					registryPath:  "./pdfs-registry/pdfs-chat2-test2.json",
					url:           "https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/promocion_interna/acceso_interna_2021_2022",
				},
			},
		},
	},
}


var MONTHS_ES = map[string]time.Month{"enero": time.January, "febrero": time.February, "marzo": time.March, "abril": time.April, "mayo": time.May, "junio": time.June, "julio": time.July, "agosto": time.August, "septiembre": time.September, "octubre": time.October, "noviembre": time.November, "diciembre": time.December}
var DEFAULT_DATE = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

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
				log.Printf("Could not parse date from date string '%s'.\n Setting default date: %v\n", s, DEFAULT_DATE)
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
	if node.FirstChild == nil { return PDF{}, errors.New("Node has no child") }

	var pdf PDF
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

type ProcessingErrorCode int
const (
	SendMessageError ProcessingErrorCode = iota + 1
	ReadTemplateError
	ReadRegistryError
	WriteRegistryError
	UnmarshalRegistryError
	MarshalRegistryError
	GetUrlContentError
)

var processingErrorCodeToString = map[ProcessingErrorCode]string{
	SendMessageError: "SendMessageError",
	ReadTemplateError: "ReadTemplateError",
	ReadRegistryError: "ReadRegistryError",
	WriteRegistryError: "WriteRegistryError",
	UnmarshalRegistryError: "UnmarshalRegistryError",
	MarshalRegistryError: "MarshalRegistryError",
	GetUrlContentError: "GetUrlContentError",
}

type processingErrorMessage struct {
	errCode  ProcessingErrorCode
	chatName string
	procName string
	pdfName  string
	message  error
}

func (errMessage *processingErrorMessage) Format (format string) string {
	return fmt.Sprintf(format,
		processingErrorCodeToString[errMessage.errCode],
		errMessage.chatName,
		errMessage.procName,
		errMessage.pdfName,
		errMessage.message,
	)
}

type pdfRegistry map[string]map[string]string

func process_updates(bot *tele.Bot, bot_config *botConfig, err_ch chan processingErrorMessage) {
	var proc func(chatConfig)
	proc = func(c chatConfig) {
		for _, sp := range c.selectiveProcs {
			log.Printf("[INFO] Processing updates for chat %s[%s], selective process '%s'", c.name, c.chatId, sp.name)

			err_message := processingErrorMessage{
				chatName: c.name,
				procName: sp.name,
				pdfName: "",
			}

			var client *http.Client = &http.Client{}
			res, err := client.Get(sp.url)
			if err != nil && res.StatusCode == 200 {
				log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Something went wrong getting url. StatusCode: %d: '%s'\n", c.name, sp.name, res.StatusCode, err)
				err_message.errCode = GetUrlContentError
				err_message.message = err
				err_ch <- err_message
				res.Body.Close()
				return
			}

			template, err := os.ReadFile(sp.templatePath)
			if err != nil {
				log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not read teamplate from path '%s'\n", c.name, sp.name, sp.templatePath)
				err_message.errCode = ReadTemplateError
				err_message.message = err
				err_ch <- err_message
				res.Body.Close()
				return
			}

			parse_registry := true
			registry_data, err := os.ReadFile(sp.registryPath)
			if err != nil {
				log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not read registry from path '%s'\n", c.name, sp.name, sp.registryPath)
				err_message.errCode = ReadRegistryError
				err_message.message = err
				err_ch <- err_message
				parse_registry = false
				// file will be created later, so we dont return in this case
			}

			var registry = pdfRegistry{}
			if parse_registry {
				err = json.Unmarshal(registry_data, &registry)
				if err != nil {
					log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not parse JSON from registry data\n", c.name, sp.name)
					err_message.errCode = UnmarshalRegistryError
					err_message.message = err
					err_ch <- err_message
					res.Body.Close()
					return
				}
			}

			pdfs := make(chan PDF)
			go gen_pdfs(res.Body, pdfs)
			for pdf, ok := <-pdfs; ok; pdf, ok = <-pdfs {
				send_pdf := false
				if _, exists := registry[pdf.name]; !exists {
					log.Printf("[INFO] Chat '%s' - Selective process '%s'. New pdf found: '%s'\n", c.name, sp.name, pdf.name)
					registry[pdf.name] = map[string]string{"pdf_url": pdf.url, "pdf_date": pdf.date}
					send_pdf = true
				} else {
					prev_date, _ := time.Parse(DATE_LAYOUT, registry[pdf.name]["pdf_date"])
					new_date, err  := time.Parse(DATE_LAYOUT, pdf.date)
					if err != nil { new_date = DEFAULT_DATE	}
					if new_date.Compare(prev_date) > 0 {
						log.Printf("[INFO] Chat '%s' - Selective process '%s'. Updated pdf found: '%s'. Date changed %s -> %s\n",
							c.name, sp.name, pdf.name, registry[pdf.name]["pdf_date"], pdf.date)
						registry[pdf.name]["pdf_date"] = pdf.date
						send_pdf = true
					}
				}

				if send_pdf {
					registry_data, err = json.Marshal(registry)
					if err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not JSON encode registry\n", c.name, sp.name)
						err_message.errCode = MarshalRegistryError
						err_message.pdfName = pdf.name
						err_message.message = err
						err_ch <- err_message
						continue
					}

					err = os.WriteFile(sp.registryPath, registry_data, 0664)
					if err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not write registry to file '%s'\n", c.name, sp.name, sp.registryPath)
						err_message.errCode = WriteRegistryError
						err_message.pdfName = pdf.name
						err_message.message = err
						err_ch <- err_message
						continue
					}

					message := fmt.Sprintf(string(template),
						sp.name,
						"https://www.aemet.es",
						pdf.url,
						pdf.name,
						pdf.date,
					)
					if _, err := bot.Send(&c, message, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not send message to chat%s\n", c.name, sp.name, err)
						err_message.errCode = SendMessageError
						err_message.pdfName = pdf.name
						err_message.message = err
						err_ch <- err_message
						continue
					}
				} // each pdf
			}  // each sp
			res.Body.Close()
		}
	}
	for _, c := range bot_config.chatConfigs {
		go proc(c)
	}
	return
}

func main() {
	sett := tele.Settings{
		Token: bot_config.token,
		Poller: &tele.LongPoller{Timeout: bot_config.timeInterval},
	}

	bot, err := tele.NewBot(sett)
	if err != nil {
		log.Fatalf("Could not instantiate bot: %s", err)
		return
	}

	go bot.Start()

	err_chan := make(chan processingErrorMessage, 50)
	for {
		select {
		case errMessageData := <-err_chan:
			if bot_config.chatAdminConfig != nil {
				errMessage := errMessageData.Format(bot_config.chatAdminConfig.errMessageFormat)
				if _, err := bot.Send(bot_config.chatAdminConfig, errMessage, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
					log.Printf("[ERROR] Could not send error message to admin chat: %s", err)
				}
			}
		default:
			log.Println("[INFO] New round!")
			go process_updates(bot, &bot_config, err_chan)
			time.Sleep(bot_config.timeInterval)
		}
	}
}
