#!/usr/bin/env zsh
app=dist/EggLedger.app
version=$(<VERSION)
echo "generating $app v${version}..."
rm -rf $app
mkdir -p $app/Contents/{MacOS,Resources}
amd64_bin=$app/Contents/MacOS/EggLedger_amd64
arm64_bin=$app/Contents/MacOS/EggLedger_arm64
universal_bin=$app/Contents/MacOS/EggLedger
echo "generating darwin amd64 build..."
GOOS=darwin GOARCH=amd64 go build -o $amd64_bin
echo "generating darwin arm64 build..."
GOOS=darwin GOARCH=arm64 go build -o $arm64_bin
echo "generating universal binary..."
lipo -create $amd64_bin $arm64_bin -output $universal_bin
rm $amd64_bin $arm64_bin
cat > $app/Contents/Info.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>EggLedger</string>
  <key>CFBundleIconFile</key>
  <string>icon.icns</string>
  <key>CFBundleIdentifier</key>
  <string>sh.tcl.EggLedger</string>
  <key>CFBundleVersion</key>
  <string>$version</string>
  <key>CFBundleShortVersionString</key>
  <string>$version</string>
</dict>
</plist>
EOF
cp icons/icon.icns $app/Contents/Resources/icon.icns
echo "generated $app"

cd dist
rm -rf EggLedger-mac.zip EggLedger
mkdir EggLedger
cp -r EggLedger.app EggLedger/
zip -r EggLedger-mac.zip EggLedger
rm -rf EggLedger
echo "generated dist/EggLedger-mac.zip"
cd ..
