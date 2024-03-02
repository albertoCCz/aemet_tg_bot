package main

import (
	"fmt"
	"log"
	"time"
	"os"
	"net/http"
	"encoding/json"
	tele "gopkg.in/telebot.v3"
)

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

func processUpdates(bot *tele.Bot, botConfig *BotConfig, err_ch chan processingErrorMessage) {
	var proc func(ChatConfig)
	proc = func(c ChatConfig) {
		for _, sp := range c.SelectiveProcs {
			log.Printf("[INFO] Processing updates for chat %s[%s], selective process '%s'\n", c.Name, c.ChatId, sp.Name)

			err_message := processingErrorMessage{
				chatName: c.Name,
				procName: sp.Name,
				pdfName: "",
			}

			var client *http.Client = &http.Client{}
			res, err := client.Get(sp.Url)
			if err != nil && res.StatusCode == 200 {
				log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Something went wrong getting url. StatusCode: %d: '%s'\n", c.Name, sp.Name, res.StatusCode, err)
				err_message.errCode = GetUrlContentError
				err_message.message = err
				err_ch <- err_message
				res.Body.Close()
				return
			}

			template, err := os.ReadFile(sp.TemplatePath)
			if err != nil {
				log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not read teamplate from path '%s'\n", c.Name, sp.Name, sp.TemplatePath)
				err_message.errCode = ReadTemplateError
				err_message.message = err
				err_ch <- err_message
				res.Body.Close()
				return
			}

			parse_registry := true
			registry_data, err := os.ReadFile(sp.RegistryPath)
			if err != nil {
				log.Printf("[WARNING] Chat '%s' - Selective process '%s'. Could not read registry from path '%s'\n", c.Name, sp.Name, sp.RegistryPath)
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
					log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not parse JSON from registry data\n", c.Name, sp.Name)
					err_message.errCode = UnmarshalRegistryError
					err_message.message = err
					err_ch <- err_message
					res.Body.Close()
					return
				}
			}

			pdfs := make(chan PDF)
			go GenPDFs(res.Body, pdfs)
			for pdf, ok := <-pdfs; ok; pdf, ok = <-pdfs {
				send_pdf := false
				if _, exists := registry[pdf.Name]; !exists {
					log.Printf("[INFO] Chat '%s' - Selective process '%s'. New pdf found: '%s'\n", c.Name, sp.Name, pdf.Name)
					registry[pdf.Name] = map[string]string{"pdf_url": pdf.Url, "pdf_date": pdf.Date}
					send_pdf = true
				} else {
					prev_date, _ := time.Parse(DATE_LAYOUT, registry[pdf.Name]["pdf_date"])
					new_date, err  := time.Parse(DATE_LAYOUT, pdf.Date)
					if err != nil { new_date = DEFAULT_DATE	}
					if new_date.Compare(prev_date) > 0 {
						log.Printf("[INFO] Chat '%s' - Selective process '%s'. Updated pdf found: '%s'. Date changed %s -> %s\n",
							c.Name, sp.Name, pdf.Name, registry[pdf.Name]["pdf_date"], pdf.Date)
						registry[pdf.Name]["pdf_date"] = pdf.Date
						send_pdf = true
					}
				}

				if send_pdf {
					registry_data, err = json.Marshal(registry)
					if err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not JSON encode registry\n", c.Name, sp.Name)
						err_message.errCode = MarshalRegistryError
						err_message.pdfName = pdf.Name
						err_message.message = err
						err_ch <- err_message
						continue
					}

					err = os.WriteFile(sp.RegistryPath, registry_data, 0664)
					if err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not write registry to file '%s'\n", c.Name, sp.Name, sp.RegistryPath)
						err_message.errCode = WriteRegistryError
						err_message.pdfName = pdf.Name
						err_message.message = err
						err_ch <- err_message
						continue
					}

					message := fmt.Sprintf(string(template),
						sp.Name,
						"https://www.aemet.es",
						pdf.Url,
						pdf.Name,
						pdf.Date,
					)
					if _, err := bot.Send(&c, message, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
						log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not send message to chat%s\n", c.Name, sp.Name, err)
						err_message.errCode = SendMessageError
						err_message.pdfName = pdf.Name
						err_message.message = err
						err_ch <- err_message
						continue
					}
				} // each pdf
			}  // each sp
			res.Body.Close()
		}
	}
	for _, c := range botConfig.ChatConfigs {
		go proc(c)
	}
	return
}

func main() {
	var botConfig BotConfig
	botConfig.SetUp("./botConfig.json")

	sett := tele.Settings{
		Token: botConfig.Token,
		Poller: &tele.LongPoller{Timeout: botConfig.TimeInterval},
	}

	bot, err := tele.NewBot(sett)
	if err != nil {
		log.Fatalf("Could not instantiate bot: %s\n", err)
		return
	}

	go bot.Start()

	err_chan := make(chan processingErrorMessage, 50)
	for {
		select {
		case errMessageData := <-err_chan:
			if botConfig.ChatAdminConfig != nil {
				errMessage := errMessageData.Format(botConfig.ChatAdminConfig.ErrMessageFormat)
				if _, err := bot.Send(botConfig.ChatAdminConfig, errMessage, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
					log.Printf("[ERROR] Could not send error message to admin chat: %s\n", err)
				}
			}
		default:
			log.Println("[INFO] New round!")
			go processUpdates(bot, &botConfig, err_chan)
			time.Sleep(botConfig.TimeInterval)
		}
	}
}
