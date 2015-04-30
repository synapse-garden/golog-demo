package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	logfile = flag.String("logfile", "log.txt", "path to log file")
	mode    = flag.String("mode", "http", "which protocol to listen on (http | zmq)")
)

func main() {
	flag.Parse()
	f, err := getOrMakeFile(*logfile)
	if err != nil {
		log.Fatal(err)
	}
	// ch will serve as a many-to-one mux for log messages
	ch := writeLog(f)
	switch *mode {
	case "http":
		http.HandleFunc("/log", handleLog(ch))
		log.Fatal(http.ListenAndServe(":8080", nil))
	default:
		log.Fatal(fmt.Sprintf("mode %q not supported", *mode))
	}
}

// getOrMakeFile either opens (write-only, in append mode) an existing file, or creates one.
func getOrMakeFile(file string) (*os.File, error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		e := err.(*os.PathError)
		if os.IsNotExist(e) {
			return os.Create(file)
		}
		return nil, e
	}
	return f, err
}

// handleLog is an http handler which logs the received id, msg to STDOUT
// and writes the msg, id to the given channel.
func handleLog(ch chan string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse args: %s", err.Error()), http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		msg := r.FormValue("msg")
		if id == "" {
			http.Error(w, "id missing", http.StatusBadRequest)
			return
		}
		if msg == "" {
			http.Error(w, "msg missing", http.StatusBadRequest)
			return
		}
		m := fmt.Sprintf("client %s: %s", id, msg)
		fmt.Fprintf(w, m)
		ch <- m
	}
}

// writeLog will return a channel to write log messages to.  Received messages will
// get appended to the given file.
func writeLog(file *os.File) chan string {
	ch := make(chan string)
	lgr := log.New(file, "golog: ", log.LstdFlags)
	go func() {
		defer file.Close()
		for {
			m, ok := <-ch
			if !ok {
				break
			}
			log.Println(m)
			lgr.Println(m)
		}
	}()

	return ch
}
