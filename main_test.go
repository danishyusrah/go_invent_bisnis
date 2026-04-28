package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ==================================================
// UNIT TESTS - Fungsi Utilitas
// ==================================================

func TestFormatRupiah_BasicValues(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 500, "500"},
		{"thousands", 1000, "1.000"},
		{"ten thousands", 15000, "15.000"},
		{"hundred thousands", 150000, "150.000"},
		{"millions", 1500000, "1.500.000"},
		{"ten millions", 15000000, "15.000.000"},
		{"hundred millions", 150000000, "150.000.000"},
		{"billions", 1000000000, "1.000.000.000"},
		{"trillions", 1000000000000, "1.000.000.000.000"},
		{"negative", -50000, "-50.000"},
		{"negative millions", -1500000, "-1.500.000"},
		{"decimal rounding", 1500.75, "1.501"},
		{"large enterprise value", 99999999999.99, "100.000.000.000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRupiah(tt.input)
			if result != tt.expected {
				t.Errorf("formatRupiah(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFuncMap_HasAllFunctions(t *testing.T) {
	fm := funcMap()
	requiredFuncs := []string{"add", "sub", "rupiah", "StoreName"}
	for _, name := range requiredFuncs {
		if fm[name] == nil {
			t.Errorf("funcMap() missing required function: %q", name)
		}
	}
}

func TestFuncMap_Add(t *testing.T) {
	fm := funcMap()
	addFn := fm["add"].(func(int, int) int)
	if addFn(3, 5) != 8 {
		t.Error("add(3,5) should be 8")
	}
	if addFn(0, 0) != 0 {
		t.Error("add(0,0) should be 0")
	}
	if addFn(-1, 1) != 0 {
		t.Error("add(-1,1) should be 0")
	}
}

func TestFuncMap_Sub(t *testing.T) {
	fm := funcMap()
	subFn := fm["sub"].(func(int, int) int)
	if subFn(10, 3) != 7 {
		t.Error("sub(10,3) should be 7")
	}
	if subFn(1, 1) != 0 {
		t.Error("sub(1,1) should be 0")
	}
}

func TestFuncMap_StoreName_Default(t *testing.T) {
	fm := funcMap()
	storeNameFn := fm["StoreName"].(func() string)
	name := storeNameFn()
	// Should return either env value or default
	if name == "" {
		t.Error("StoreName() should never return empty string")
	}
}

// ==================================================
// BENCHMARK TESTS - Kecepatan Performa
// ==================================================

func BenchmarkFormatRupiah_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatRupiah(1500)
	}
}

func BenchmarkFormatRupiah_Medium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatRupiah(15000000)
	}
}

func BenchmarkFormatRupiah_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatRupiah(999999999999.99)
	}
}

func BenchmarkGinRouter_SingleRoute(b *testing.B) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkGinRouter_MultiRoute(b *testing.B) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Simulasi jumlah route yang mirip aplikasi kita
	routes := []string{"/", "/pos", "/inventory", "/history", "/accounting", "/users", "/categories", "/login", "/logout"}
	for _, route := range routes {
		route := route
		r.GET(route, func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})
	}

	req, _ := http.NewRequest("GET", "/inventory", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// ==================================================
// STRESS TEST - Simulasi 100.000 Permintaan
// ==================================================

func TestStress_100K_Requests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "ts": time.Now().UnixNano()})
	})

	totalRequests := 100000
	start := time.Now()

	for i := 0; i < totalRequests; i++ {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Request #%d failed with status %d", i, w.Code)
		}
	}

	elapsed := time.Since(start)
	rps := float64(totalRequests) / elapsed.Seconds()

	t.Logf("============================================")
	t.Logf("  STRESS TEST RESULTS")
	t.Logf("============================================")
	t.Logf("  Total Requests : %d", totalRequests)
	t.Logf("  Total Duration : %v", elapsed)
	t.Logf("  Avg Latency    : %v / request", elapsed/time.Duration(totalRequests))
	t.Logf("  Throughput     : %.0f req/sec", rps)
	t.Logf("============================================")

	// PASS jika mampu menangani minimal 10.000 req/sec
	if rps < 10000 {
		t.Errorf("Performance too low! Got %.0f req/sec, need at least 10,000 req/sec", rps)
	}
}

// ==================================================
// CONCURRENT STRESS TEST - Simulasi Multi-User
// ==================================================

func TestStress_ConcurrentUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent stress test in short mode")
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	numUsers := 50
	requestsPerUser := 2000
	totalRequests := numUsers * requestsPerUser

	errChan := make(chan error, totalRequests)
	start := time.Now()

	for u := 0; u < numUsers; u++ {
		go func(userID int) {
			for i := 0; i < requestsPerUser; i++ {
				req, _ := http.NewRequest("GET", "/api/test", nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					errChan <- fmt.Errorf("user %d request %d: status %d", userID, i, w.Code)
					return
				}
			}
			errChan <- nil
		}(u)
	}

	// Tunggu semua goroutine selesai
	var errors []error
	for i := 0; i < numUsers; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	elapsed := time.Since(start)
	rps := float64(totalRequests) / elapsed.Seconds()

	t.Logf("============================================")
	t.Logf("  CONCURRENT STRESS TEST RESULTS")
	t.Logf("============================================")
	t.Logf("  Simulated Users  : %d", numUsers)
	t.Logf("  Req/User         : %d", requestsPerUser)
	t.Logf("  Total Requests   : %d", totalRequests)
	t.Logf("  Total Duration   : %v", elapsed)
	t.Logf("  Avg Latency      : %v / request", elapsed/time.Duration(totalRequests))
	t.Logf("  Throughput       : %.0f req/sec", rps)
	t.Logf("  Failed Requests  : %d", len(errors))
	t.Logf("============================================")

	if len(errors) > 0 {
		t.Errorf("%d requests failed", len(errors))
	}
	if rps < 5000 {
		t.Errorf("Concurrent performance too low! Got %.0f req/sec, need at least 5,000", rps)
	}
}
