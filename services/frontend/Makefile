deploy:
	cp -R ../../lib ./build
	sed -i '' 's/..\/..\/lib/.\/build\/lib/g' go.mod
	fly deploy
	rm -r build
	sed -i '' 's/.\/build\/lib/..\/..\/lib/g' go.mod
