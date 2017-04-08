$PWD=$((Get-Item -Path ".\" -Verbose).FullName)
$BIN_NAME="emlmr.exe"
$BUILD_DIR="$PWD\build"
$VERSION="$(git describe --always --dirty=-snapshot)"
$GO_LDFLAGS="-s -X main.version=$VERSION"

if (-not $env:GOPATH) {
	$env:GOPATH = "$env:USERPROFILE\go"
}

$Env:PATH += ";$Env:GOPATH\bin"
go generate
go build -ldflags "$GO_LDFLAGS" -o "$BUILD_DIR\$BIN_NAME"
