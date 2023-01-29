# TODO add spinners

new:
	@printf "`tput bold`Installing Poetry...this may take some time unless you have it installed already...`tput sgr0`\n"
	@curl -sSL https://install.python-poetry.org | python3 -

	@printf "`tput bold`Creating new Poetry project...`tput sgr0`\n"
	@$(HOME)/.local/bin/poetry new $(CAKEWORK_APP_NAME)
	@cd $(CAKEWORK_APP_NAME); $(HOME)/.local/bin/poetry env use system
	@printf "\n\n"

	@printf "`tput bold`Copying over important files...`tput sgr0`\n"

	@cp assets/.gitignore $(CAKEWORK_APP_NAME)/.gitignore
	
	@sed -i'' -e 's/REPLACE_APPNAME/$(CAKEWORK_APP_NAME)/g' assets/main.py
	@echo $(CAKEWORK_APP_NAME) | sed 's/-/_/g' | xargs -I '{}' cp assets/main.py $(CAKEWORK_APP_NAME)/{}/main.py

	@cd $(CAKEWORK_APP_NAME); $(HOME)/.local/bin/poetry env info | grep -Eo '[0-9]{1,}.[0-9]{1,}' -m 1 | xargs -I '{}' sed -i'' -e 's/REPLACE_PYTHONVERSION/{}/g' ../assets/Dockerfile
	@$(HOME)/.local/bin/poetry --version | grep -Eo '[0-9]{1,}.[0-9]{1,}.[0-9]{1,}' | xargs -I '{}' sed -i'' -e 's/REPLACE_POETRYVERSION/{}/g' assets/Dockerfile
	@echo $(CAKEWORK_APP_NAME) | sed 's/-/_/g' | xargs -I '{}' sed -i'' -e 's/REPLACE_APPNAME/{}/g' assets/Dockerfile
	@cp assets/Dockerfile $(CAKEWORK_APP_NAME)/Dockerfile

	@sed -i'' -e 's/REPLACE_APPNAME/$(CAKEWORK_APP_NAME)/g' assets/bake.py
	@cp assets/bake.py $(CAKEWORK_APP_NAME)/tests/bake.py
	
	@printf "\n\n"


	@printf "`tput bold`Installing Cakework dependency...`tput sgr0`\n"
	@cd $(CAKEWORK_APP_NAME); $(HOME)/.local/bin/poetry add cakework
	@printf "\n\n"

	@printf "`tput bold`Generating sample client...`tput sgr0`\n"
	@cd $(CAKEWORK_APP_NAME)/tests; echo $(CAKEWORK_CLIENT_TOKEN) > .env
	@printf "\n\n"

	@printf "`tput bold`Your Cakework project $(CAKEWORK_APP_NAME) is created!`tput sgr0` 🎂\n"

clean:
	@rm -rf assets Makefile