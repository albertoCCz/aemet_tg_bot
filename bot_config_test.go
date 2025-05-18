package main

import (
	"errors"
	"testing"
)

func TestLoadEnvVars(t *testing.T) {
	botCon := BotConfig{
		Token: "",
		Name:  "bot_name",
		ChatAdminConfig: &ChatAdminConfig{
			ChatId: "",
			Name:   "admin_name",
		},
		ChatConfigs: []ChatConfig{
			ChatConfig{
				ChatId: "",
				Name:   "chat_1_name",
			},
			ChatConfig{
				ChatId: "",
				Name:   "chat_2_name",
			},
		},
	}

	t.Run("AllEnvVarsExist", func(t *testing.T) {
		t.Setenv("BOT_TOKEN_bot_name", "12")
		t.Setenv("bot_name_CHAT_ID_admin_name", "34")
		t.Setenv("bot_name_CHAT_ID_chat_1_name", "56")
		t.Setenv("bot_name_CHAT_ID_chat_2_name", "78")

		var want error
		err := loadEnvVars(&botCon)
		if err != want {
			t.Errorf(errFmtString, want, err)
		}
	})

	t.Run("MissingEnvVar/BOT_TOKEN", func(t *testing.T) {
		t.Setenv("bot_name_CHAT_ID_admin_name", "34")
		t.Setenv("bot_name_CHAT_ID_chat_1_name", "56")
		t.Setenv("bot_name_CHAT_ID_chat_2_name", "78")

		want := errors.New("some err")
		err := loadEnvVars(&botCon)
		if err == nil {
			t.Errorf(errFmtString, want, err)
		}
	})

	t.Run("MissingEnvVar/REGULAR_CHAT", func(t *testing.T) {
		t.Setenv("BOT_TOKEN_bot_name", "12")
		t.Setenv("bot_name_CHAT_ID_admin_name", "34")
		t.Setenv("bot_name_CHAT_ID_chat_2_name", "78")

		want := errors.New("some err")
		err := loadEnvVars(&botCon)
		if err == nil {
			t.Errorf(errFmtString, want, err)
		}
	})

	t.Run("MissingEnvVar/ADMIN_CHAT", func(t *testing.T) {
		t.Setenv("BOT_TOKEN_bot_name", "12")
		t.Setenv("bot_name_CHAT_ID_chat_1_name", "56")
		t.Setenv("bot_name_CHAT_ID_chat_2_name", "78")

		want := errors.New("some err")
		err := loadEnvVars(&botCon)
		if err == nil {
			t.Errorf(errFmtString, want, err)
		}
	})
}

func TestObfuscate(t *testing.T) {
	botCon := BotConfig{
		Token: "bot_token",
		Name:  "bot_name",
		ChatAdminConfig: &ChatAdminConfig{
			ChatId: "admin_id",
			Name:   "admin_name",
		},
		ChatConfigs: []ChatConfig{
			ChatConfig{
				ChatId: "chat_1_id",
				Name:   "chat_1_name",
			},
			ChatConfig{
				ChatId: "chat_2_id",
				Name:   "chat_2_name",
			},
		},
	}

	obfuscate(&botCon)

	want := ""
	if botCon.Token != want {
		t.Errorf(errFmtString, want, botCon.Token)
	}

	if botCon.ChatAdminConfig.ChatId != want {
		t.Errorf(errFmtString, want, botCon.ChatAdminConfig.ChatId)
	}

	for _, cc := range botCon.ChatConfigs {
		if cc.ChatId != want {
			t.Errorf(errFmtString, want, cc.ChatId)
		}
	}
}
