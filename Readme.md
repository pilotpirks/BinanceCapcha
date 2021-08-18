### Building for source

1
install golang and related libraries

```sh
go get github.com/go-rod/rod
```

[details here](https://go-rod.github.io/#/get-started/README)

```sh
go get gocv.io/x/gocv
```

[details here](https://gocv.io/)
need install opencv

2
in project root:

```sh
go mod tidy
go build -trimpath -ldflags="-s -w" -o build/captcha.exe
```

use file ".rod" for debugging, [details here](https://go-rod.github.io/#/get-started/README?id=see-what39s-under-the-hood)