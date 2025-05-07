package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	Data string
}

type Worker struct {
	ID    int
	Proxy string
}

var (
	proxyLock          sync.Mutex
	proxyList          []string
	usedProxies        = map[string]bool{}
	proxyIndex         = 0
	rateLimitedProxies = make(map[string]bool)
)

func getNewProxy() (string, bool) {
	for {
		proxyLock.Lock()

		var foundRateLimited bool
		for proxyIndex < len(proxyList) {
			p := proxyList[proxyIndex]
			proxyIndex++

			if usedProxies[p] {
				continue
			}

			if rateLimitedProxies[p] {
				foundRateLimited = true
				continue
			}

			usedProxies[p] = true
			proxyLock.Unlock()
			return p, true
		}

		proxyLock.Unlock()

		if foundRateLimited {
			slog.Info("Waiting for available proxy...")
			time.Sleep(5 * time.Second)
			continue
		}

		return "", false
	}
}

func removeProxy(target string) {
	proxyLock.Lock()
	defer proxyLock.Unlock()

	// Remove from proxyList
	var newList []string
	for i, p := range proxyList {
		if p != target {
			newList = append(newList, p)
		} else if i < proxyIndex {
			proxyIndex-- // Adjust index if removed before current pointer
		}
	}
	proxyList = newList

	// Clean up from other maps
	delete(usedProxies, target)
	delete(rateLimitedProxies, target)

	slog.Info("Removed proxy permanently", "proxy", target)
}

func releaseProxyLater(proxy string, delay time.Duration) {
	proxyLock.Lock()
	defer proxyLock.Unlock()
	rateLimitedProxies[proxy] = true

	go func() {
		time.Sleep(delay)
		proxyLock.Lock()
		defer proxyLock.Unlock()
		rateLimitedProxies[proxy] = false
		usedProxies[proxy] = false
		slog.Info("Proxy re-added after rate limit delay", "proxy", proxy)
	}()
}

func loadLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func worker(w Worker, taskQueue <-chan Task, retryQueue chan<- Task) bool {
	for task := range taskQueue {
		err := process(task, w.Proxy)
		if err != nil {
			slog.Error("Failed processing", "error", err)
			// TODO: error checking

			retryQueue <- task
			return false
		}
	}

	return true
}

func askForWorkerCount() int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter max number of workers: ")
	input, _ := reader.ReadString('\n')

	input = strings.TrimSpace(input)
	count, err := strconv.Atoi(input)
	if err != nil || count <= 0 {
		slog.Error("Invalid number, defaulting to 10 workers.")
		return 10
	}

	return count
}

func main() {
	proxies, err := loadLines("proxies.txt")
	if err != nil {
		slog.Error("Failed loading proxies from \"proxies.txt\", does it exist?")
		return
	}
	proxyList = proxies

	inputs, err := loadLines("input.txt")
	if err != nil {
		slog.Error("Failed loading proxies from \"input.txt\", does it exist?")
		return
	}

	maxWorkers := min(askForWorkerCount(), len(proxyList))
	slog.Info("Set config", "worker count", maxWorkers)

	taskQueue := make(chan Task, len(inputs))
	retryQueue := make(chan Task, len(inputs))
	var wg sync.WaitGroup

	for _, line := range inputs {
		taskQueue <- Task{
			Data: line,
		}
	}
	close(taskQueue)

	go func() {
		for task := range retryQueue {
			time.Sleep(2 * time.Second)
			select {
			case taskQueue <- task:
			default:
				slog.Warn("Task queue full, dropping task", "task", task.Data)
			}
		}
	}()

	for i := 0; i < maxWorkers; i++ {
		proxy, ok := getNewProxy()
		if !ok {
			break
		}

		w := Worker{
			ID:    i,
			Proxy: proxy,
		}

		wg.Add(1)
		go func(w Worker) {
			defer wg.Done()

			for {
				success := worker(w, taskQueue, retryQueue)
				if success {
					return
				}

				newProxy, ok := getNewProxy()
				if !ok {
					slog.Error("No more proxies left to switch, killing worker")
					return
				}
				w.Proxy = newProxy
			}
		}(w)
	}

	wg.Wait()
	slog.Info("All tasks were processed or all workers were errored / closed.")
}
