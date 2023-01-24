local:
	go build -o poller 
	export STAGE=dev
	./poller

deploy:
	mkdir build
	cp -R ../../lib ./build/lib
	sed -i '' 's/..\/..\/lib/.\/build\/lib/g' go.mod
	- fly deploy
	rm -r build
	sed -i '' 's/.\/build\/lib/..\/..\/lib/g' go.mod
