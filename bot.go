package main

import (
	"fmt"
	"log"
	"time"
	"os"
	"strings"
	"errors"
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
	BlankPDFDateError
)

var processingErrorCodeToString = map[ProcessingErrorCode]string{
	SendMessageError: "SendMessageError",
	ReadTemplateError: "ReadTemplateError",
	ReadRegistryError: "ReadRegistryError",
	WriteRegistryError: "WriteRegistryError",
	UnmarshalRegistryError: "UnmarshalRegistryError",
	MarshalRegistryError: "MarshalRegistryError",
	GetUrlContentError: "GetUrlContentError",
	BlankPDFDateError: "BlankPDFDateError",
}

type processingErrorMessage struct {
	errCode  ProcessingErrorCode
	chatName string
	procName string
	pdfName  string
	message  error
}

func (errMessage *processingErrorMessage) Format() string {
	format := "Error: <strong>%s</strong>\n" +
		"  - chat name: <i>%s</i>\n" +
		"  - proc name: <i>%s</i>\n" +
		"  - pdf name:  <i>%s</i>\n" +
		"  - message:   <pre language=\"console\">%s</pre>\n"

	return fmt.Sprintf(format,
		processingErrorCodeToString[errMessage.errCode],
		errMessage.chatName,
		errMessage.procName,
		errMessage.pdfName,
		errMessage.message,
	)
}

type pdfRegistry map[string]map[string]string

func processUpdates(bot *tele.Bot, botConfig *BotConfig, err_ch chan processingErrorMessage, send_on bool) {
	var procChat func(ChatConfig)
	procChat = func(c ChatConfig) {
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

			registry_data, err := os.ReadFile(sp.RegistryPath)
			var registry = pdfRegistry{}
			if err != nil {
				log.Printf("[WARNING] Chat '%s' - Selective process '%s'. Could not read registry from path '%s'\n", c.Name, sp.Name, sp.RegistryPath)
				err_message.errCode = ReadRegistryError
				err_message.message = err
				err_ch <- err_message
				// file will be created later, so we dont return in this case
			} else {
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
			go GenPDFs(res.Body, pdfs) // <- this one closes the channel when finishes
			for pdf, ok := <-pdfs; ok; pdf, ok = <-pdfs {
				if pdf.Date == "" {
					log.Printf("[ERROR] Chat '%s' - Selective process '%s'. Could not JSON encode registry\n", c.Name, sp.Name)
					err_message.errCode = BlankPDFDateError
					err_message.pdfName = pdf.Name
					err_message.message = errors.New("PDF Date is blank. This might be due to a error when parsing it")
					err_ch <- err_message
				}

				if _, exists := registry[pdf.Name]; !exists {
					log.Printf("[INFO] Chat '%s' - Selective process '%s'. New pdf found: '%+v'\n", c.Name, sp.Name, pdf)
					registry[pdf.Name] = map[string]string{"pdf_url": pdf.Url, "pdf_date": pdf.Date}

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

					if send_on {
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
					}
				} // if pdf !exists
			} // each pdf
			res.Body.Close()
		} // each sp
	} // procChat
	for _, c := range botConfig.ChatConfigs {
		go procChat(c)
	}

	return
}

func handle_run_command(configPath string) {
	var botConfig BotConfig
	botConfig.SetUp(configPath)

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
				errMessage := errMessageData.Format()
				if _, err := bot.Send(botConfig.ChatAdminConfig, errMessage, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
					log.Printf("[ERROR] Could not send error message to admin chat: %s\n", err)
				}
			}
		default:
			log.Println("[INFO] New round!")
			go processUpdates(bot, &botConfig, err_chan, true)
			time.Sleep(botConfig.TimeInterval)
		}
	}
}

func handle_init_command(configPath string) {
	var botConfig BotConfig
	botConfig.SetUp(configPath)

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

	log.Println("[INFO] Starting initialisation.")
	err_chan := make(chan processingErrorMessage, 50)
	all_processed := false
	for {
		select {
		case errMessageData := <-err_chan:
			if botConfig.ChatAdminConfig != nil {
				errMessage := errMessageData.Format()
				if _, err := bot.Send(botConfig.ChatAdminConfig, errMessage, &tele.SendOptions{ParseMode: "HTML"}); err != nil {
					log.Printf("[ERROR] Could not send error message to admin chat: %s\n", err)
				}
			}
			fmt.Println("error case")
		default:
			if !all_processed {
				go processUpdates(bot, &botConfig, err_chan, false)
				all_processed = true
				time.Sleep(botConfig.TimeInterval * 2)
			}
			log.Println("[INFO] Registries initialised!")
			return
		}
	}
}

func usage() {
	fmt.Println(
		"usage: bot <command> [--bot-config=<config-path>]\n\n" +
		"commands:\n" +
	    "    help       Print this help.\n" +
		"    run        Start running the bot. It needs the --bot-config flag.\n" +
		"    init       Initialise the registries by running the bot. It needs the --bot-config flag.\n" +
		"               Only the error messages to admin chat (if configured) will be sent.\n")
}

func nextFlagValue(command, flag string, args []string) string {
	if !strings.HasPrefix(args[0], flag) {
		usage()
		fmt.Printf("[ERROR] Unknown flag '%s' for command '%s'\n", args[0], command)
		os.Exit(-1)
	}

	flag_split := strings.SplitN(args[0], "=", 2)
	if len(flag_split) < 2 {
		usage()
		fmt.Printf("[ERROR] Missing value for flag '%s' in '%s' command\n", flag, command)
		os.Exit(-1)
	}

	return flag_split[1]
}

func main() {
	if len(os.Args) <= 1 {
		usage()
		fmt.Println("[ERROR] No command provided")
		os.Exit(-1)
	}

	switch command := os.Args[1]; command {
	case "help":
		usage()
		return
	case "run":
		flag := "--bot-config"
		if len(os.Args) <= 2 {
			usage()
			fmt.Printf("[ERROR] You need to pass the flag '%s' with command '%s'\n", flag, command)
			os.Exit(-1)
		}
		configPath := nextFlagValue(command, flag, os.Args[2:])
		handle_run_command(configPath)
	case "init":
		flag := "--bot-config"
		if len(os.Args) <= 2 {
			usage()
			fmt.Printf("[ERROR] You need to pass the flag '%s' with command '%s'\n", flag, command)
			os.Exit(-1)
		}
		configPath := nextFlagValue(command, flag, os.Args[2:])
		handle_init_command(configPath)
	default:
		usage()
		fmt.Printf("[ERROR] Unknown command '%s'\n", command)
		os.Exit(-1)
	}
}
