package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var DEFAULT_DATE = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// main structs. The herarchy is:
// botConfig
//   |_ ...
//   |_ ChatAdminConfig
//   |_ []ChatConfig
//     |_ ...
//     |_ []SelectiveProc

type SelectiveProc struct {
	Name         string
	TemplatePath string
	RegistryPath string
	Url          string
}

type ChatConfig struct {
	ChatId         string
	Name           string
	SelectiveProcs []SelectiveProc
}

func (c *ChatConfig) Recipient() string {
	return c.ChatId
}

type ChatAdminConfig struct {
	ChatId string
	Name   string
}

func (c *ChatAdminConfig) Recipient() string {
	return c.ChatId
}

type BotConfig struct {
	Token           string
	Name            string
	TimeInterval    time.Duration
	ChatAdminConfig *ChatAdminConfig
	ChatConfigs     []ChatConfig
}

func loadEnvVars(bc *BotConfig) error {
	envVar := fmt.Sprintf("BOT_TOKEN_%s", bc.Name)
	if token, ok := os.LookupEnv(envVar); !ok {
		return fmt.Errorf("[ERROR] Environment variable '%s' is unset", envVar)
	} else {
		bc.Token = token
	}

	envVar = fmt.Sprintf("%s_CHAT_ID_%s", bc.Name, bc.ChatAdminConfig.Name)
	if chatId, ok := os.LookupEnv(envVar); !ok {
		return fmt.Errorf("[ERROR] Environment variable '%s' is unset", envVar)
	} else {
		bc.ChatAdminConfig.ChatId = chatId
	}

	for i := range len(bc.ChatConfigs) {
		envVar = fmt.Sprintf("%s_CHAT_ID_%s", bc.Name, bc.ChatConfigs[i].Name)
		if chatId, ok := os.LookupEnv(envVar); !ok {
			return fmt.Errorf("[ERROR] Environment variable '%s' is unset", envVar)
		} else {
			bc.ChatConfigs[i].ChatId = chatId
		}
	}

	return nil
}

func obfuscate(bc *BotConfig) {
	bc.Token = ""
	bc.ChatAdminConfig.ChatId = ""

	for i := range len(bc.ChatConfigs) {
		bc.ChatConfigs[i].ChatId = ""
	}
}

func (bc BotConfig) WriteFile(path string) error {
	obfuscate(&bc)

	bc_data, err := json.MarshalIndent(bc, "", "    ")
	if err != nil {
		log.Printf("[ERROR] Bot Configuration could not be JSON-encoded: %s\n", err)
		return err
	}

	if err = os.WriteFile(path, bc_data, 0664); err != nil {
		log.Printf("[ERROR] Could not write Bot Configuration to file '%s': %s\n", path, err)
		return err
	}

	return nil
}

func (bc *BotConfig) ReadFile(path string) error {
	bc_data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] Could not read Bot Configuration from file '%s': %s\n", path, err)
		return err
	}

	if err = json.Unmarshal(bc_data, bc); err != nil {
		log.Printf("[ERROR] Could not JSON-decode Bot Configuration: %s\n", err)
		return err
	}

	return nil
}

func (bc *BotConfig) SetUp(path string) error {
	if err := bc.ReadFile(path); err != nil {
		log.Println("[ERROR] Could not read bot configuration from file")
		return err
	}

	if err := loadEnvVars(bc); err != nil {
		log.Printf("[ERROR] Could not load environment variables into bot configuration: %s\n", err)
		os.Exit(-1)
	}

	return nil
}
