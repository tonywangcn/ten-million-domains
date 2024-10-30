package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var Sigchan = make(chan os.Signal, 1)

const (
	jobQueue  = "job:queue:"
	statsKey  = "http:status:code:stats:"
	key       = "ten-million-domains:"
	userAgent = "Mozilla/5.0 (compatible; TenMillionDomainsBot/1.0; +https://github.com/tonywangcn/ten-million-domains)"
)

func Exit() {
	Sigchan <- syscall.SIGTERM
}

type Worker struct {
	Name  string
	Wg    sync.WaitGroup
	Jobs  chan string
	Stats map[int]int
	Mutex sync.Mutex
}

func RunWorker(workerCount int) {
	worker := NewWorker(key)
	go worker.fetchJobs()
	go worker.syncStatsPeriodically(time.Minute)
	go worker.showStatsPeriodically(time.Minute)
	worker.run(workerCount)
}

func NewWorker(name string) *Worker {
	return &Worker{
		Name:  name,
		Jobs:  make(chan string, 10000),
		Stats: make(map[int]int),
	}
}

func (w *Worker) fetchJobs() {
	for {
		if len(w.Jobs) > 100 {
			time.Sleep(time.Second)
			continue
		}
		jobs := SPopN(w.Name+jobQueue, 100)
		if len(jobs) > 0 {
			for _, job := range jobs {
				w.AddJob(job)
			}
		} else {
			count := Scard(w.Name + jobQueue)
			log.Printf("Total number of jobs in job queue: %d", count)
			time.Sleep(3 * time.Second)
		}
	}
}

func (w *Worker) AddJob(job string) {
	w.Jobs <- job
	log.Printf("New job added: %+v for %s", job, w.Name)
}

func (w *Worker) run(workerCount int) {
	w.Wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer w.Wg.Done()
			for job := range w.Jobs {
				w.worker(job)
			}
		}()
	}
	w.Wg.Wait()
}

var dnsServers = []string{
	"8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1", "208.67.222.222", "208.67.220.220",
	"9.9.9.9", "149.112.112.112",
}

func (w *Worker) worker(job string) {
	var ips []net.IPAddr
	var err error
	var customDNSServer string
	for retry := 0; retry < 5; retry++ {
		customDNSServer = dnsServers[rand.Intn(len(dnsServers))]
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", customDNSServer+":53")
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ips, err = resolver.LookupIPAddr(ctx, job)
		if err == nil && len(ips) > 0 {
			break
		}

		log.Printf("Retry %d: Failed to resolve %s on DNS server: %s, error: %v", retry+1, job, customDNSServer, err)
	}

	if err != nil || len(ips) == 0 {
		log.Printf("Failed to resolve %s on DNS server: %s after retries, error: %v", job, customDNSServer, err)
		w.updateStats(1000)
		return
	}

	customDialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	customTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			port := "80"
			if strings.HasPrefix(addr, "https://") {
				port = "443"
			}
			return customDialer.DialContext(ctx, network, ips[0].String()+":"+port)
		},
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: customTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://"+job, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		w.updateStats(0)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && strings.Contains(urlErr.Err.Error(), "http: server gave HTTP response to HTTPS client") {
			log.Printf("Request failed due to HTTP response to HTTPS client: %v", err)
			// Retry with HTTPS
			req.URL.Scheme = "https"
			customTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return customDialer.DialContext(ctx, network, ips[0].String()+":443")
			}
			resp, err = client.Do(req)
			if err != nil {
				log.Printf("HTTPS request failed: %v", err)
				w.updateStats(0)
				return
			}
		} else {
			log.Printf("Request failed: %v", err)
			w.updateStats(0)
			return
		}
	}
	defer resp.Body.Close()

	log.Printf("Received response from %s: %s", job, resp.Status)
	w.updateStats(resp.StatusCode)
}

func (w *Worker) updateStats(statusCode int) {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	w.Stats[statusCode]++
}

func (w *Worker) syncStatsToRedis() {
	w.Mutex.Lock()
	defer w.Mutex.Unlock()
	for code, count := range w.Stats {
		if count > 0 {
			if err := IncrBy(w.Name+":"+statsKey+fmt.Sprint(code), int64(count)); err != nil {
				log.Printf("Failed to sync stats to Redis for status code %d: %v", code, err)
			}
		}
	}
	log.Printf("Synced stats to Redis for worker %s", w.Name)
	w.Stats = make(map[int]int)
}

func (w *Worker) syncStatsPeriodically(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		w.syncStatsToRedis()
	}
}

func (w *Worker) showStatsPeriodically(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := w.showStats(); err != nil {
			log.Printf("Error showing stats: %v", err)
		}
	}
}

func (w *Worker) showStats() error {
	var cursor uint64
	pattern := w.Name + ":" + statsKey + "*"

	for {
		keys, newCursor, err := Redis.SScan(ctx, pattern, cursor, "", 10).Result()
		if err != nil {
			return fmt.Errorf("error scanning Redis keys: %w", err)
		}

		for _, key := range keys {
			val, err := Redis.Get(ctx, key).Result()
			if err != nil {
				log.Printf("Error retrieving key %s: %v", key, err)
				continue
			}
			log.Printf("Key: %s, Value: %s", key, val)
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
