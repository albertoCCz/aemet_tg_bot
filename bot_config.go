package main

import "time"


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
	ChatId           string
	Name             string
	ErrMessageFormat string
}

func (c *ChatAdminConfig) Recipient() string {
	return c.ChatId
}

type BotConfig struct {
	Token           string
	TimeInterval    time.Duration
	ChatAdminConfig *ChatAdminConfig
	ChatConfigs     []ChatConfig
}
