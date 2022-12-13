/*
	Building blocks for creating web server:
	1- Multiplexer or MUX -> which handler belongs to which route
	2- Listener -> host and port
	3- Handlers -> handler functions

	(💡): Each request is dealt with by a single goroutine on the web server
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"
	"time"
)

type Server struct {
	http http.Server
	mux  *http.ServeMux
}

const (
	httpAPITimeout  = time.Second * 3
	shutdownTimeout = time.Second * 10
)

func New(port int) *Server {
	s := &Server{}
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.mux = mux

	s.http = http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.TimeoutHandler(s.mux, httpAPITimeout, ""),
	}
	return s
}

func (s *Server) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	var err error

	go func() {
		if err2 := s.http.ListenAndServe(); err2 != http.ErrServerClosed {
			panic(fmt.Errorf("could not start http server %s\n", err2))
		}
	}()
	fmt.Printf("listening on %s \n", s.http.Addr)

	<-stopCh

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = s.http.Shutdown(ctx); err == nil {
		fmt.Println("http server shutdown")
		return
	}

	if err = context.DeadlineExceeded; err != nil {
		fmt.Println("Shutdown timeout exceeded. Closing http server")
		if err = s.http.Close(); err != nil {
			fmt.Printf("could not close http connection: %v \n", err)
		}
		return
	}
	fmt.Println("could not shutdown http server")
}

func (s *Server) HandlerFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	server := New(8080)
	server.HandlerFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//<-time.After(time.Second * 4)
		fmt.Println(r.Header)
		b, _ := os.ReadFile("resources/flower.webp")
		w.Header().Add("content-type", "image-webp")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(b)
		if err != nil {
			log.Fatal(err)
		}
	})

	stop := make(chan struct{})
	go server.Run(stop, wg)

	go func() {
		<-time.After(time.Second * 20)
		stop <- struct{}{}
	}()

	wg.Wait()
	fmt.Println("Party is over!")
}
