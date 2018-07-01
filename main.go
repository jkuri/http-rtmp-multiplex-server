package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/jkuri/http-rtmp-multiplex-server/rtmp"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/flv"
	"github.com/soheilhy/cmux"
)

func init() {
	format.RegisterAll()
}

type writeFlusher struct {
	httpFlusher http.Flusher
	io.Writer
}

func (wf writeFlusher) Flush() error {
	wf.httpFlusher.Flush()
	return nil
}

func main() {
	server := &rtmp.Server{}
	l := &sync.RWMutex{}

	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	server.HandlePlay = func(conn *rtmp.Conn) {
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			avutil.CopyFile(conn, cursor)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		streams, _ := conn.Streams()

		l.Lock()
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.WriteHeader(streams)
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyPackets(ch.que, conn)

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	httpFlvHandler := func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.RLock()
			ch := channels[r.URL.Path]
			l.RUnlock()

			if ch != nil {
				w.Header().Set("Content-Type", "video/x-flv")
				w.Header().Set("Transfer-Encoding", "chunked")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(200)
				flusher := w.(http.Flusher)
				flusher.Flush()

				muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpFlusher: flusher, Writer: w})
				cursor := ch.que.Latest()

				avutil.CopyFile(muxer, cursor)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	router := http.NewServeMux()
	router.Handle("/", httpFlvHandler())

	ln, err := net.Listen("tcp", ":8300")
	if err != nil {
		log.Fatal(err)
	}

	m := cmux.New(ln)

	httpL := m.Match(cmux.HTTP1Fast())
	rtmpL := m.Match(cmux.Any())

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	httpServer := &http.Server{
		Handler: logging(logger)(router),
	}

	go httpServer.Serve(httpL)
	go server.ListenAndServe(rtmpL)

	m.Serve()
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}
