local:
	go build -o frontend
	export STAGE=dev
	./frontend -verbose

deploy:
	mkdir build
	cp -R ../../lib ./build/lib
	sed -i '' 's/..\/..\/lib/.\/build\/lib/g' go.mod
	- fly deploy
	rm -r build
	sed -i '' 's/.\/build\/lib/..\/..\/lib/g' go.mod
