package scanner

// Scanner is the main struct for the scanning engine.
type Scanner struct {
	resolver     *Resolver
	debugNetwork bool
}

// ScanResult represents a single found subdomain and its IP address.
type ScanResult struct {
	Subdomain string `json:"Subdomain"`
	IPAddress string `json:"IPAddress"`
}

// ScanStatus represents the progress of a scan.
type ScanStatus struct {
	Scanned       int
	Total         int
	Failed        int
	Concurrency   int
	TotalRequests int
	TotalRetries  int
	Phase         string
	TotalRetrying int
}

// NewScanner creates a new Scanner instance.
func NewScanner(debugNetwork bool) *Scanner {
	return &Scanner{
		resolver:     NewResolver(debugNetwork),
		debugNetwork: debugNetwork,
	}
}

// SetDNSServers sets the DNS servers to be used for resolution.
func (s *Scanner) SetDNSServers(servers []string) {
	s.resolver.SetDNSServers(servers)
}

// Start begins the subdomain scanning process.
func (s *Scanner) Start(domain string, wordlistChan <-chan string, totalTasks int, resultsChan chan<- *ScanResult, statusChan chan<- ScanStatus, concurrency int, adaptive bool, maxQPS int, enableRetry bool) {
	scheduler := newScheduler(s.resolver, domain, wordlistChan, totalTasks, resultsChan, statusChan, concurrency, adaptive, maxQPS)
	scheduler.run(enableRetry)
}
