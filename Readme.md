
#### Video (slow debug mode)

https://user-images.githubusercontent.com/36490605/129956522-b9b0d054-4c42-4922-8174-4b3624749607.mp4

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
