# TODO: protobuf
# TODO: css
build:
	go build .

dev-app: build
	echo EggLedger | DEV_MODE=1 entr -r ./EggLedger

dev-css:
	yarn tailwindcss -i www/index.pcss -o www/index.css --watch

readme:
	asciidoctor -a nofooter -a 'webfonts!' README.macOS.txt
