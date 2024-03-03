# AEMET Telegram Bot
This bot has been developed to find new PDF documents in AEMET web pages and send them to the relevant Telegram chats.

## Usage
```console
usage: ./aemet_tg_bot <command> [--bot-config=<config-path>]

commands:
    help       Print this help.
    run        Start running the bot. It needs the --bot-config flag.
    init       Initialise the registries by running the bot. It needs the --bot-config flag.
               Only the error messages to admin chat (if configured) will be sent.
```

## Quickstart
### Using Docker
This option requires having docker installed.

1. Download the docker image `aemet_tg_bot` from the package section of this repository.
2. Set up the bot configuration in the `./botConfig.json` file.
3. Set up the necessary environment variables.
    - For each Telegram chat/group defined in `./botConfig.json`, create a environment variable like `<bot-name>_CHAT_ID_<chat-name>`. For example, in linux, if the bot name is `TEST_BOT` and the chat name is `CHAT_1`:
      ```console
      $ export TEST_BOT_CHAT_ID_CHAT_1="<chat-id>"
      ```
    - Define a variable for the bot token. It must follows this convention: `BOT_TOKEN_<bot-name>`. Following the same example, we would have to define it like this:
      ```console
      $ export BOT_TOKEN_TEST_BOT="<bot-token>"
      ```
4. (_Optionally_) Initialise the registries so already uploaded PDFs are not sent to the Telegram groups:
```console
$ docker run aemet_tg_bot
```
5. Run the bot
```console
$ docker run aemet_tg_bot run --bot-config=./botConfig.json
```

### Without Docker
This option requires having Go v1.22 installed.

1. Clone this repository and cd into it:
```console
$ git clone https://github.com/albertoCCz/aemet_tg_bot
$ cd aemet_tg_bot
```
2. Download dependencies and build executable.
```console
$ go build
```
3. Set up the bot configuration in the `./botConfig.json` file. Leave blank the chatId fields and token field, these will be read from the environment variables.
4. Set up the necessary environment variables.
    - For each Telegram chat/group defined in `./botConfig.json`, create a environment variable like `<bot-name>_CHAT_ID_<chat-name>`. For example, in linux, if the bot name is `TEST_BOT` and the chat name is `CHAT_1`:
      ```console
      $ export TEST_BOT_CHAT_ID_CHAT_1="<chat-id>"
      ```
    - Define a variable for the bot token. It must follows this convention: `BOT_TOKEN_<bot-name>`. Following the same example, we would have to define it like this:
      ```console
      $ export BOT_TOKEN_TEST_BOT="<bot-token>"
      ```
5. (_Optionally_) Initialise the registries so already uploaded PDFs are not sent to the Telegram groups:
```console
$ aemet_tg_bot init --bot-config=./botConfig.json
```
6. Run the bot
```console
$ aemet_tg_bot run --bot-config=./botConfig.json
```
