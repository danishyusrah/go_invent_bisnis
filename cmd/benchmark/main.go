package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Konfigurasi Connection Pool seperti di main.go
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Ambil jumlah data
	var totalProducts int
	db.QueryRow("SELECT COUNT(*) FROM products").Scan(&totalProducts)

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          MYSQL DATABASE PERFORMANCE BENCHMARK               ║")
	fmt.Println("║          Danish Elektronik ERP System                        ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Total Records di Tabel Products: %-26d ║\n", totalProducts)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	results := []benchResult{}

	// ============================================================
	// TEST 1: SELECT COUNT(*) — Full Table Scan Count
	// ============================================================
	results = append(results, runBench(db, "COUNT(*) Full Table",
		"SELECT COUNT(*) FROM products", false))

	// ============================================================
	// TEST 2: SELECT dengan LIMIT (Pagination halaman 1)
	// ============================================================
	results = append(results, runBench(db, "Pagination Hal.1 (LIMIT 20)",
		"SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location FROM products ORDER BY id DESC LIMIT 20", false))

	// ============================================================
	// TEST 3: Pagination halaman tengah (OFFSET besar)
	// ============================================================
	results = append(results, runBench(db, "Pagination Hal.5000 (OFFSET 100000)",
		"SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location FROM products ORDER BY id DESC LIMIT 20 OFFSET 100000", false))

	// ============================================================
	// TEST 4: Pencarian LIKE (Live Search simulasi)
	// ============================================================
	results = append(results, runBench(db, "Live Search LIKE '%shure%'",
		"SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location FROM products WHERE part_number LIKE '%shure%' OR description LIKE '%shure%' OR category LIKE '%shure%' LIMIT 20", false))

	// ============================================================
	// TEST 5: Pencarian LIKE kata populer
	// ============================================================
	results = append(results, runBench(db, "Live Search LIKE '%speaker%'",
		"SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location FROM products WHERE part_number LIKE '%speaker%' OR description LIKE '%speaker%' OR category LIKE '%speaker%' LIMIT 20", false))

	// ============================================================
	// TEST 6: SELECT WHERE exact match (Index Hit)
	// ============================================================
	results = append(results, runBench(db, "Exact Match part_number (INDEX)",
		"SELECT id, part_number, description, category FROM products WHERE part_number = 'DNE-001-00001'", false))

	// ============================================================
	// TEST 7: SELECT WHERE category (Index Hit)
	// ============================================================
	results = append(results, runBench(db, "Filter by Category (INDEX)",
		"SELECT COUNT(*) FROM products WHERE category = 'Microphone'", false))

	// ============================================================
	// TEST 8: Aggregate SUM — Total Nilai Inventaris
	// ============================================================
	results = append(results, runBench(db, "SUM(capital_price * quantity) Inventaris",
		"SELECT SUM(capital_price * quantity) FROM products", false))

	// ============================================================
	// TEST 9: GROUP BY Category — Laporan Akuntansi
	// ============================================================
	results = append(results, runBench(db, "GROUP BY category (Laporan)",
		"SELECT category, COUNT(*), SUM(capital_price*quantity) FROM products GROUP BY category", false))

	// ============================================================
	// TEST 10: Low Stock Alert Query
	// ============================================================
	results = append(results, runBench(db, "Low Stock Alert (qty <= min)",
		"SELECT COUNT(*) FROM products WHERE quantity <= min_stock_level", false))

	// ============================================================
	// TEST 11: INSERT single row
	// ============================================================
	results = append(results, runBench(db, "INSERT Single Row",
		"INSERT INTO products (part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location) VALUES ('BENCH-TEST-001', 'Benchmark Test Product', 'Aksesoris Audio', 10, 5, 100000, 150000, 'Rak A1')", true))

	// ============================================================
	// TEST 12: UPDATE single row by ID
	// ============================================================
	results = append(results, runBench(db, "UPDATE Single Row by ID",
		"UPDATE products SET quantity = quantity + 1 WHERE part_number = 'BENCH-TEST-001'", true))

	// ============================================================
	// TEST 13: DELETE single row
	// ============================================================
	results = append(results, runBench(db, "DELETE Single Row",
		"DELETE FROM products WHERE part_number = 'BENCH-TEST-001'", true))

	// ============================================================
	// TEST 14: JOIN Query (Transaksi + Produk)
	// ============================================================
	results = append(results, runBench(db, "JOIN transactions+products",
		"SELECT p.description, t.transaction_type, t.quantity FROM inventory_transactions t JOIN products p ON t.product_id = p.id LIMIT 50", false))

	// ============================================================
	// TEST 15: Concurrent SELECT — 100 query paralel
	// ============================================================
	fmt.Print("   🔄 Running: Concurrent 100x SELECT Parallel... ")
	concStart := time.Now()
	done := make(chan time.Duration, 100)
	for i := 0; i < 100; i++ {
		go func() {
			s := time.Now()
			var c int
			db.QueryRow("SELECT COUNT(*) FROM products WHERE category = 'Amplifier'").Scan(&c)
			done <- time.Since(s)
		}()
	}
	var totalLatency time.Duration
	var maxLatency time.Duration
	for i := 0; i < 100; i++ {
		d := <-done
		totalLatency += d
		if d > maxLatency {
			maxLatency = d
		}
	}
	concElapsed := time.Since(concStart)
	fmt.Printf("✅ %v (avg: %v, max: %v)\n", concElapsed, totalLatency/100, maxLatency)
	results = append(results, benchResult{
		name:    "100x Concurrent SELECT (Parallel)",
		elapsed: concElapsed,
		pass:    concElapsed < 5*time.Second,
	})

	// ============================================================
	// TEST 16: EXPLAIN check — apakah index digunakan
	// ============================================================
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    INDEX HEALTH CHECK                        ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	checkIndex(db, "part_number lookup", "EXPLAIN SELECT * FROM products WHERE part_number = 'DNE-001-00001'")
	checkIndex(db, "category filter", "EXPLAIN SELECT * FROM products WHERE category = 'Microphone'")

	// ============================================================
	// FINAL REPORT
	// ============================================================
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  FINAL PERFORMANCE REPORT                    ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	allPass := true
	for _, r := range results {
		status := "✅ PASS"
		if !r.pass {
			status = "❌ SLOW"
			allPass = false
		}
		fmt.Printf("║  %s %-40s %10v ║\n", status, r.name, r.elapsed.Round(time.Microsecond))
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	if allPass {
		fmt.Println("║  🏆 VERDICT: DATABASE PERFORMA OPTIMAL — SIAP PRODUKSI!    ║")
	} else {
		fmt.Println("║  ⚠️  VERDICT: ADA QUERY YANG PERLU DIOPTIMASI              ║")
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

type benchResult struct {
	name    string
	elapsed time.Duration
	pass    bool
}

func runBench(db *sql.DB, name string, query string, isWrite bool) benchResult {
	fmt.Printf("   🔄 Running: %-45s", name+"...")

	iterations := 10
	var totalElapsed time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		if isWrite {
			db.Exec(query)
		} else {
			rows, err := db.Query(query)
			if err == nil {
				for rows.Next() {
				} // consume all rows
				rows.Close()
			}
		}
		totalElapsed += time.Since(start)
	}

	avg := totalElapsed / time.Duration(iterations)

	// Threshold: query harus selesai di bawah 1 detik rata-rata
	pass := avg < 1*time.Second

	if pass {
		fmt.Printf("✅ avg: %v\n", avg.Round(time.Microsecond))
	} else {
		fmt.Printf("❌ avg: %v (SLOW!)\n", avg.Round(time.Microsecond))
	}

	return benchResult{name: name, elapsed: avg, pass: pass}
}

func checkIndex(db *sql.DB, label string, query string) {
	rows, err := db.Query(query)
	if err != nil {
		fmt.Printf("║  ❌ %-50s ERROR  ║\n", label)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	values := make([]sql.RawBytes, len(cols))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	if rows.Next() {
		rows.Scan(scanArgs...)
		// Find "type" and "key" columns
		typeVal := ""
		keyVal := ""
		for i, col := range cols {
			if col == "type" {
				typeVal = string(values[i])
			}
			if col == "key" {
				keyVal = string(values[i])
			}
		}

		if keyVal != "" && keyVal != "NULL" {
			fmt.Printf("║  ✅ %-35s INDEX USED: %-15s ║\n", label, keyVal)
		} else {
			fmt.Printf("║  ⚠️  %-35s FULL SCAN (type: %-10s) ║\n", label, typeVal)
		}
	}
}
