.PHONY: css css-watch build run

css:
	./tailwindcss-macos-arm64 -i static/css/input.css -o static/css/style.css --minify

css-watch:
	./tailwindcss-macos-arm64 -i static/css/input.css -o static/css/style.css --watch

build:
	go build -o zrp .

run: css build
	./zrp
