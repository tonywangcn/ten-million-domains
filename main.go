package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var workerMap map[string]func(count int) = map[string]func(count int){
	"worker": RunWorker,
	"job":    LoadJob,
	"stats":  Stats,
}

func main() {
	shutdown := make(chan int)
	signal.Notify(Sigchan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<16)
			stackTrace := string(buf[0:runtime.Stack(buf, true)])
			log.Printf("Got pannic, %s, stack trace: %s", r, stackTrace)
			Exit()
		}
	}()

	go func() {
		<-Sigchan
		log.Println("Shutting down...")
		log.Println("Shutted down...")
		shutdown <- 1
	}()

	var worker string
	var count int

	flag.StringVar(&worker, "w", "worker", "Choose the worker to run")
	flag.IntVar(&count, "c", 20, "Number of workers to run")

	flag.Parse()
	f, ok := workerMap[worker]
	if !ok {
		panic("worker not found")
	}
	go f(count)

	<-shutdown
}
