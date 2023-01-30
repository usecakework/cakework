publish:
	yarn install
	yarn build
	npm version patch
	npm config set //registry.npmjs.org/:_authToken ${NPM_TOKEN}
	npm publish --ignore-scripts --access public
