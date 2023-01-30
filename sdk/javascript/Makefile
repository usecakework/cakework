update_fern_client:
	cd ../../fern && fern generate
	cd ../typescript && sed -i'' -e 's/withCredentials\: true/withCredentials\: false/g' src/core/fetcher/Fetcher.ts
	cd ../typescript && npm version patch
	cd ../typescript && npm config set //registry.npmjs.org/:_authToken ${NPM_TOKEN}
	cd ../typescript && npm publish --ignore-scripts --access public
	npm install @cakework/fern_client@latest

publish:
	npm version patch
	npm config set //registry.npmjs.org/:_authToken ${NPM_TOKEN}
	npm publish --ignore-scripts --access public
