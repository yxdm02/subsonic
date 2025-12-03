package scanner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	// Guaranteed minimum concurrency for adaptive mode unless user specifies a lower value.
	guaranteedMinConcurrency = 150
	maxConcurrency           = 5000
	adjustInterval           = 2 * time.Second
)

type scheduler struct {
	resolver     *Resolver
	domain       string
	wordlistChan <-chan string
	resultsChan  chan<- *ScanResult
	statusChan   chan<- ScanStatus
	adaptive     bool
	limiter      *rate.Limiter

	wg            sync.WaitGroup
	totalTasks    int
	scanned       int32
	failed        int32
	totalRequests int32
	totalRetries  int32

	activeWorkers  int32
	minConcurrency int32

	// Metrics for new adaptive mode
	totalResolutions   int32
	retriedResolutions int32

	// For retry logic
	failedDomains []string
	mu            sync.Mutex

	// Channels for signaling
	stopChan chan struct{} // Signals workers to stop
	quitChan chan struct{} // Signals the monitor to quit
}

func newScheduler(resolver *Resolver, domain string, wordlistChan <-chan string, totalTasks int, resultsChan chan<- *ScanResult, statusChan chan<- ScanStatus, concurrency int, adaptive bool, maxQPS int) *scheduler {
	minWorkers := int32(guaranteedMinConcurrency)
	if !adaptive && concurrency > 0 {
		minWorkers = int32(concurrency)
	} else if adaptive && concurrency > 0 && concurrency < guaranteedMinConcurrency {
		// If user provides a custom concurrency lower than the guaranteed minimum, respect it.
		minWorkers = int32(concurrency)
	}
	
	initialConcurrency := minWorkers
	if !adaptive && concurrency > 0 {
		initialConcurrency = int32(concurrency)
	}

	var limiter *rate.Limiter
	if maxQPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(maxQPS), maxQPS)
	}

	return &scheduler{
		resolver:      resolver,
		domain:        domain,
		wordlistChan:  wordlistChan,
		totalTasks:    totalTasks,
		resultsChan:   resultsChan,
		statusChan:    statusChan,
		activeWorkers: initialConcurrency,
		minConcurrency: minWorkers,
		adaptive:      adaptive,
		limiter:       limiter,
		stopChan:      make(chan struct{}, maxConcurrency),
		quitChan:      make(chan struct{}),
	}
}

func (s *scheduler) run(enableRetry bool) {
	defer close(s.resultsChan)
	defer close(s.statusChan)

	tasksChan := make(chan string, maxConcurrency)

	// Initial worker setup
	s.wg.Add(int(s.activeWorkers))
	for i := 0; i < int(s.activeWorkers); i++ {
		go s.worker(tasksChan)
	}

	if s.adaptive {
		go s.monitorAndAdjust(tasksChan)
	}

	// --- Phase 1: Main Scan ---
	log.Println("Starting main scan phase...")
	s.sendStatus("main_scan", 0)
	for word := range s.wordlistChan {
		tasksChan <- fmt.Sprintf("%s.%s", word, s.domain)
	}
	close(tasksChan)
	s.wg.Wait()
	log.Println("Main scan phase finished.")

	// --- Decision Point ---
	if !enableRetry || len(s.failedDomains) == 0 {
		log.Println("No retry needed or feature disabled. Finishing scan.")
		if s.adaptive {
			close(s.quitChan) // Signal monitor to stop
		}
		s.sendStatus("done", 0)
		return
	}

	// --- Phase 2: Retry Scan ---
	log.Printf("Starting retry phase for %d failed domains...", len(s.failedDomains))
	retryTasks := s.failedDomains
	s.failedDomains = nil // Clear the slice
	s.totalTasks = len(retryTasks)
	atomic.StoreInt32(&s.scanned, 0)
	atomic.StoreInt32(&s.failed, 0)

	s.sendStatus("retry_scan", s.totalTasks)

	tasksChan = make(chan string, maxConcurrency)

	// Relaunch workers for the retry phase
	s.wg.Add(int(s.activeWorkers))
	for i := 0; i < int(s.activeWorkers); i++ {
		go s.worker(tasksChan)
	}

	// Start a goroutine to feed the retry tasks
	go func() {
		for _, domain := range retryTasks {
			tasksChan <- domain
		}
		close(tasksChan)
	}()

	s.wg.Wait()
	log.Println("Retry scan phase finished.")

	if s.adaptive {
		close(s.quitChan) // Signal monitor to stop
	}
	s.sendStatus("done", 0)
}

func (s *scheduler) worker(tasksChan <-chan string) {
	defer s.wg.Done()
	for {
		select {
		case subdomain, ok := <-tasksChan:
			if !ok {
				return
			}

			if s.limiter != nil {
				s.limiter.Wait(context.Background())
			}

			ip, status, attempts, _ := s.resolver.Resolve(subdomain)

			atomic.AddInt32(&s.scanned, 1)
			atomic.AddInt32(&s.totalRequests, int32(attempts))
			if attempts > 1 {
				atomic.AddInt32(&s.totalRetries, int32(attempts-1))
				atomic.AddInt32(&s.retriedResolutions, 1)
			}

			if status == Failed {
				atomic.AddInt32(&s.failed, 1)
				s.mu.Lock()
				s.failedDomains = append(s.failedDomains, subdomain)
				s.mu.Unlock()
			}

			atomic.AddInt32(&s.totalResolutions, 1)

			if status == Success && ip != "" {
				result := GetScanResult()
				result.Subdomain = subdomain
				result.IPAddress = ip
				s.resultsChan <- result
			}

			if atomic.LoadInt32(&s.scanned)%1000 == 0 || int(atomic.LoadInt32(&s.scanned)) == s.totalTasks {
				// Status sending logic is now phase-dependent, handled in run()
				// For simplicity, we'll send status based on the main task count during main scan
				// and based on retry count during retry scan.
				// A more robust implementation might check the phase here.
				s.sendStatus("", 0) // Phase will be determined by the receiver context now
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *scheduler) monitorAndAdjust(tasksChan chan string) {
	ticker := time.NewTicker(adjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			total := atomic.SwapInt32(&s.totalResolutions, 0)
			retried := atomic.SwapInt32(&s.retriedResolutions, 0)

			if total == 0 {
				log.Println("[Adaptive] No resolutions in the last interval, maintaining concurrency.")
				continue
			}

			retryRate := float64(retried) / float64(total)
			currentConcurrency := atomic.LoadInt32(&s.activeWorkers)
			log.Printf("[Adaptive] Retry Rate: %.2f%% | Current Concurrency: %d", retryRate*100, currentConcurrency)

			if retryRate < 0.20 && currentConcurrency < maxConcurrency { // Healthy Zone
				log.Println("[Adaptive] Healthy zone, increasing concurrency by 40.")
				s.adjustWorkers(40, tasksChan)
			} else if retryRate >= 0.20 && retryRate < 0.50 && currentConcurrency < maxConcurrency { // Pressure Zone
				log.Println("[Adaptive] Pressure zone, increasing concurrency by 20.")
				s.adjustWorkers(20, tasksChan)
			} else if retryRate >= 0.50 && retryRate < 0.70 && currentConcurrency > s.minConcurrency { // Warning Zone
				log.Println("[Adaptive] Warning zone, decreasing concurrency by 60.")
				s.adjustWorkers(-60, tasksChan)
			} else if retryRate >= 0.70 && currentConcurrency > s.minConcurrency { // Danger Zone
				log.Println("[Adaptive] Danger zone, decreasing concurrency by 120.")
				s.adjustWorkers(-120, tasksChan)
			}

		case <-s.quitChan:
			log.Println("[Adaptive] Scan finished, stopping monitor.")
			return
		}
	}
}

func (s *scheduler) adjustWorkers(delta int, tasksChan chan string) {
	if delta > 0 {
		newWorkers := atomic.AddInt32(&s.activeWorkers, int32(delta))
		log.Printf("[Adaptive] Increasing worker count to %d", newWorkers)
		s.wg.Add(delta)
		for i := 0; i < delta; i++ {
			go s.worker(tasksChan)
		}
	} else if delta < 0 {
		numToStop := -delta
		currentWorkers := atomic.LoadInt32(&s.activeWorkers)
		
		// Ensure we don't go below the guaranteed minimum concurrency
		if currentWorkers - int32(numToStop) < s.minConcurrency {
			numToStop = int(currentWorkers - s.minConcurrency)
		}
		
		if numToStop <= 0 {
			return
		}

		newWorkers := atomic.AddInt32(&s.activeWorkers, int32(-numToStop))
		log.Printf("[Adaptive] Decreasing worker count to %d", newWorkers)
		for i := 0; i < numToStop; i++ {
			s.stopChan <- struct{}{}
		}
	}
}

func (s *scheduler) sendStatus(phase string, totalRetrying int) {
	// A bit of a hack to determine phase if not provided, for simplicity.
	// A better way would be to have the scheduler maintain its current phase in its state.
	if phase == "" {
		if len(s.failedDomains) > 0 {
			phase = "main_scan"
		} else {
			// This is ambiguous, could be retry or main. Let's assume main for now.
			// The main `run` loop will send the definitive phase status.
			phase = "main_scan"
		}
	}

	s.statusChan <- ScanStatus{
		Scanned:       int(atomic.LoadInt32(&s.scanned)),
		Total:         s.totalTasks,
		Failed:        int(atomic.LoadInt32(&s.failed)),
		Concurrency:   int(atomic.LoadInt32(&s.activeWorkers)),
		TotalRequests: int(atomic.LoadInt32(&s.totalRequests)),
		TotalRetries:  int(atomic.LoadInt32(&s.totalRetries)),
		Phase:         phase,
		TotalRetrying: totalRetrying,
	}
}
