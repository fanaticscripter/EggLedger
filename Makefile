.PHONY: build css protobuf readme dev-app dev-css dist

build:
	go build .

css:
	yarn build:css

protobuf:
	protoc --proto_path=. --go_out=paths=source_relative:. ei/ei.proto

readme:
	asciidoctor -a nofooter -a 'webfonts!' README.macOS.txt

dev-app: build
	echo EggLedger | DEV_MODE=1 entr -r ./EggLedger

dev-css:
	yarn dev:css

dist: css protobuf readme
	./build-macos.sh
	./build-windows.sh
