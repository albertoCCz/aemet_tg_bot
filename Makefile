# Common variables
WORKDIR = $(CURDIR)
WORKDIR_APP = /usr/src/app
ENTRYPOINT = /usr/bin/bash

# Initialize bot registries. Do not send any message.
init-bot: COMMAND = ./aemet_tg_bot init --bot-config=botConfig.json
init-bot:
	sudo docker run -dit \
		--env-file ./env.list \
		--volume "$(WORKDIR)/botConfig.json":"$(WORKDIR_APP)/botConfig.json" \
		--volume "$(WORKDIR)/logs":"$(WORKDIR_APP)/logs" \
		--volume "$(WORKDIR)/templates":"$(WORKDIR_APP)/templates" \
		--volume "$(WORKDIR)/pdfs-registry":"$(WORKDIR_APP)/pdfs-registry" \
		--entrypoint "$(ENTRYPOINT)" \
		ghcr.io/albertoccz/aemet_tg_bot:main -c "$(COMMAND)"

# Start the bot.
run-bot: COMMAND = ./aemet_tg_bot run --bot-config=botConfig.json
run-bot:
	sudo docker run -dit \
		--env-file ./env.list \
	    --volume "$(WORKDIR)/botConfig.json":"$(WORKDIR_APP)/botConfig.json" \
		--volume "$(WORKDIR)/logs":"$(WORKDIR_APP)/logs" \
		--volume "$(WORKDIR)/templates":"$(WORKDIR_APP)/templates" \
		--volume "$(WORKDIR)/pdfs-registry":"$(WORKDIR_APP)/pdfs-registry" \
		--entrypoint "$(ENTRYPOINT)" \
		ghcr.io/albertoccz/aemet_tg_bot:main -c "$(COMMAND)"
