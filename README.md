## HTTP and RTMP Multiplexed Server

Example is showing how its possible to run HTTP and RTMP server on same port.

Useful when we are limited to run our service on single port.

That example would not be possible without two awesome libraries;

- [cmux](https://github.com/soheilhy/cmux) - connection multiplexer for Go
- [joy4](https://github.com/nareix/joy4) - audio/video library and streaming server for Go

### Running Server

```sh
go get -u -v github.com/jkuri/http-rtmp-multiplex-server
cd $GOPATH/src/github.com/jkuri/http-rtmp-multiplex-server
dep ensure
go run main.go
```

Server will start on port `8300`.

### Example Usage

First run the server, then

```sh
ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost:8300/movie # stream movie from file
ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost:8300/screen # stream screen in MacOS
```

Play the live stream:

```sh
mpv http://localhost:8300/movie # play movie
ffplay http://localhost:8300/screen # play screen
```

### License

MIT
