package scanner

import "sync"

// scanResultPool is a pool of ScanResult objects to reduce memory allocations.
var scanResultPool = sync.Pool{
	New: func() interface{} {
		return &ScanResult{}
	},
}

// GetScanResult retrieves a ScanResult from the pool.
func GetScanResult() *ScanResult {
	return scanResultPool.Get().(*ScanResult)
}

// PutScanResult returns a ScanResult to the pool for reuse.
func PutScanResult(result *ScanResult) {
	// It's good practice to reset the object before putting it back.
	result.Subdomain = ""
	result.IPAddress = ""
	scanResultPool.Put(result)
}
