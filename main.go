package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// Dashboard cache — refresh setiap 30 detik agar dashboard instan
type DashboardCache struct {
	TotalSKU    int
	OutOfStock  int
	LowStock    int
	TotalValue  float64
	TotalCount  int // untuk pagination tanpa COUNT(*)
	LastRefresh time.Time
}

var dashCache DashboardCache

func startCacheRefresher() {
	// Refresh langsung di awal
	refreshDashboardCache()
	
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			refreshDashboardCache()
		}
	}()
}

func refreshDashboardCache() {
	var totalSKU, outOfStock, lowStock int
	var totalValue float64

	db.QueryRow(`SELECT 
		COUNT(*), 
		SUM(CASE WHEN quantity = 0 THEN 1 ELSE 0 END),
		SUM(CASE WHEN quantity > 0 AND quantity <= min_stock_level THEN 1 ELSE 0 END)
		FROM products`).Scan(&totalSKU, &outOfStock, &lowStock)
	
	db.QueryRow("SELECT COALESCE(SUM(quantity * capital_price), 0) FROM products").Scan(&totalValue)

	dashCache = DashboardCache{
		TotalSKU:    totalSKU,
		OutOfStock:  outOfStock,
		LowStock:    lowStock,
		TotalValue:  totalValue,
		TotalCount:  totalSKU,
		LastRefresh: time.Now(),
	}
	log.Printf("Dashboard cache refreshed: %d SKUs", totalSKU)
}
type Product struct {
	ID            int     `json:"id"`
	PartNumber    string  `json:"part_number"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Quantity      int     `json:"quantity"`
	MinStockLevel int     `json:"min_stock_level"`
	CapitalPrice  float64 `json:"capital_price"`
	SellingPrice  float64 `json:"selling_price"`
	Location      string  `json:"location"`
}

type User struct {
	ID       int
	Username string
	Role     string
}

type Transaction struct {
	ID              int
	ReceiptID       string
	PartNumber      string
	Description     string
	Username        string
	TransactionType string
	Quantity        int
	TotalValue      float64
	Notes           string
	Date            string
}

type Expense struct {
	ID          int
	Description string
	Amount      float64
	Date        string
	Username    string
}

type Category struct {
	ID   int
	Name string
}

func initDB() {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}
	fmt.Println("Connected to Database")

	schema1 := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role ENUM('admin', 'staff') NOT NULL DEFAULT 'staff',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	schema2 := `
	CREATE TABLE IF NOT EXISTS inventory_transactions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		product_id INT NOT NULL,
		user_id INT,
		transaction_type ENUM('IN', 'OUT') NOT NULL,
		quantity INT NOT NULL,
		notes VARCHAR(255),
		transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
	);`
	schema3 := `
	CREATE TABLE IF NOT EXISTS expenses (
		id INT AUTO_INCREMENT PRIMARY KEY,
		description VARCHAR(255) NOT NULL,
		amount DECIMAL(30,2) NOT NULL,
		expense_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		user_id INT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
	);`
	schema4 := `
	CREATE TABLE IF NOT EXISTS categories (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE
	);`

	_, err1 := db.Exec(schema1)
	if err1 != nil { log.Printf("Error creating users table: %v", err1) }
	_, err2 := db.Exec(schema2)
	if err2 != nil { log.Printf("Error creating transactions table: %v", err2) }
	_, err3 := db.Exec(schema3)
	if err3 != nil { log.Printf("Error creating expenses table: %v", err3) }
	_, err4 := db.Exec(schema4)
	if err4 != nil { log.Printf("Error creating categories table: %v", err4) }

	var catCount int
	db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&catCount)
	if catCount == 0 {
		db.Exec("INSERT IGNORE INTO categories (name) VALUES ('Speaker Aktif'), ('Speaker Pasif'), ('Amplifier'), ('Microphone'), ('Kabel & Konektor'), ('Aksesoris Audio'), ('Uncategorized')")
	}
	// Advanced Accounting Schema Updates (Silent if already applied)
	_, _ = db.Exec("ALTER TABLE products ADD COLUMN category VARCHAR(100) NOT NULL DEFAULT 'Uncategorized' AFTER description")
	_, _ = db.Exec("ALTER TABLE products ADD COLUMN selling_price DECIMAL(30, 2) NOT NULL DEFAULT 0.00 AFTER capital_price")
	_, _ = db.Exec("ALTER TABLE inventory_transactions ADD COLUMN receipt_id VARCHAR(50) NULL AFTER user_id")
	_, _ = db.Exec("ALTER TABLE inventory_transactions ADD COLUMN total_value DECIMAL(30, 2) NOT NULL DEFAULT 0.00 AFTER quantity")
	
	// Database Performance & Indexing Maximize
	_, _ = db.Exec("CREATE INDEX idx_part_number ON products(part_number)")
	_, _ = db.Exec("CREATE INDEX idx_category ON products(category)")
	_, _ = db.Exec("CREATE INDEX idx_receipt_id ON inventory_transactions(receipt_id)")
	
	// Advanced Performance Indexes for 500K+ records
	_, _ = db.Exec("CREATE INDEX idx_category_qty_price ON products(category, quantity, capital_price)")
	_, _ = db.Exec("CREATE INDEX idx_qty_minstock ON products(quantity, min_stock_level)")
	_, _ = db.Exec("CREATE INDEX idx_desc_search ON products(description(100))")
	_, _ = db.Exec("ALTER TABLE products ADD FULLTEXT INDEX ft_search (part_number, description, category)")
	
	// Connection Pool Optimization (Go-MySQL Best Practices)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Remove arbitrary nominal bounds
	_, _ = db.Exec("ALTER TABLE products MODIFY COLUMN capital_price DECIMAL(30, 2) NOT NULL DEFAULT 0.00")
	_, _ = db.Exec("ALTER TABLE products MODIFY COLUMN selling_price DECIMAL(30, 2) NOT NULL DEFAULT 0.00")
	_, _ = db.Exec("ALTER TABLE inventory_transactions MODIFY COLUMN total_value DECIMAL(30, 2) NOT NULL DEFAULT 0.00")
	_, _ = db.Exec("ALTER TABLE expenses MODIFY COLUMN amount DECIMAL(30, 2) NOT NULL")

	// Create default admin and staff if not exist
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin'), (?, ?, 'staff')", "admin", hash, "staff", hash)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables")
	}

	initDB()
	
	// Start background daily backup
	go startDailyBackup()
	
	// Start dashboard cache refresher (hot cache setiap 30 detik)
	go startCacheRefresher()

	defer db.Close()

	r := gin.Default()
	r.SetFuncMap(funcMap())

	store := cookie.NewStore([]byte("secret_jack_sound_key_12345"))
	r.Use(sessions.Sessions("mysession", store))

	r.LoadHTMLGlob("templates/*")

	// Public Routes
	r.GET("/login", loginGetHandler)
	r.POST("/login", loginPostHandler)
	r.GET("/logout", logoutHandler)

	// Protected Routes
	auth := r.Group("/")
	auth.Use(AuthRequired())
	{
		auth.GET("/", dashboardHandler)
		auth.GET("/inventory", inventoryHandler)
		auth.GET("/product/qrcode/:id", generateQRCodeHandler)

		// Transaction Routes (Both Admin and Staff can do transactions)
		auth.GET("/product/:id/transaction", transactionGetHandler)
		auth.POST("/product/:id/transaction", transactionPostHandler)
		auth.GET("/history", historyHandler)
		
		// POS & APIs
		auth.GET("/pos", posGetHandler)
		auth.GET("/api/product/:part_number", apiGetProductHandler)
		auth.POST("/pos/checkout", posCheckoutHandler)
		auth.GET("/receipt/:receipt_id", receiptHandler)
		auth.GET("/inventory/labels", printLabelsHandler)
		
		auth.GET("/accounting", accountingHandler)
		auth.POST("/accounting/expense", expensePostHandler)
		auth.GET("/accounting/export", exportPnlHandler)

		// Admin Only Routes
		admin := auth.Group("/")
		admin.Use(AdminRequired())
		{
			// Users Management
			admin.GET("/users", usersGetHandler)
			admin.POST("/users/new", usersPostHandler)
			admin.POST("/users/delete/:id", usersDeleteHandler)
			admin.POST("/users/reset/:id", usersResetHandler)

			// Category Management
			admin.GET("/categories", categoriesHandler)
			admin.POST("/categories/new", categoriesPostHandler)
			admin.POST("/categories/delete/:id", categoriesDeleteHandler)

			// Backup Management
			admin.GET("/backup", manualBackupHandler)

			admin.GET("/product/new", newProductHandler)
			admin.POST("/product/new", createProductHandler)
			admin.GET("/product/edit/:id", editProductHandler)
			admin.POST("/product/edit/:id", updateProductHandler)
			admin.POST("/product/delete/:id", deleteProductHandler)
		}
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

func formatRupiah(amount float64) string {
	str := fmt.Sprintf("%.0f", math.Round(amount))
	var result []byte
	isNegative := false
	if len(str) > 0 && str[0] == '-' {
		isNegative = true
		str = str[1:]
	}
	
	for i := len(str) - 1; i >= 0; i-- {
		if len(str)-i-1 > 0 && (len(str)-i-1)%3 == 0 {
			result = append([]byte{'.'}, result...)
		}
		result = append([]byte{str[i]}, result...)
	}
	
	if isNegative {
		return "-" + string(result)
	}
	return string(result)
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"add":    func(a, b int) int { return a + b },
		"sub":    func(a, b int) int { return a - b },
		"rupiah": formatRupiah,
		"StoreName": func() string {
			name := os.Getenv("STORE_NAME")
			if name == "" {
				return "Jack Sound Audio"
			}
			return name
		},
	}
}
func getFlashes(c *gin.Context) []interface{} {
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()
	return flashes
}

func getUser(c *gin.Context) *User {
	session := sessions.Default(c)
	uid := session.Get("user_id")
	if uid == nil {
		return nil
	}
	var u User
	err := db.QueryRow("SELECT id, username, role FROM users WHERE id = ?", uid).Scan(&u.ID, &u.Username, &u.Role)
	if err != nil {
		return nil
	}
	return &u
}

// Middlewares
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := getUser(c)
		if u == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user", u)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exists := c.Get("user")
		if !exists || u.(*User).Role != "admin" {
			session := sessions.Default(c)
			session.AddFlash("Akses ditolak: Hanya Admin yang bisa mengakses halaman ini.")
			session.Save()
			c.Redirect(http.StatusFound, "/inventory")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Auth Handlers
func loginGetHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Flashes": getFlashes(c),
	})
}

func loginPostHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var id int
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)

	session := sessions.Default(c)
	if err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil {
		session.Set("user_id", id)
		session.Save()
		c.Redirect(http.StatusFound, "/")
	} else {
		session.AddFlash("Username atau password salah.")
		session.Save()
		c.Redirect(http.StatusFound, "/login")
	}
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

// View Handlers
func dashboardHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)

	// Gunakan cache — INSTAN, tidak perlu query database
	totalSKU := dashCache.TotalSKU
	outOfStock := dashCache.OutOfStock
	lowStock := dashCache.LowStock
	totalValue := dashCache.TotalValue

	rows, err := db.Query("SELECT id, part_number, description, quantity, min_stock_level, capital_price, location FROM products WHERE quantity <= min_stock_level ORDER BY quantity ASC LIMIT 10")
	var criticalProducts []Product
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p Product
			rows.Scan(&p.ID, &p.PartNumber, &p.Description, &p.Quantity, &p.MinStockLevel, &p.CapitalPrice, &p.Location)
			criticalProducts = append(criticalProducts, p)
		}
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"User":             user,
		"ActiveMenu":       "dashboard",
		"Title":            "Dashboard",
		"TotalSKU":         totalSKU,
		"OutOfStock":       outOfStock,
		"LowStock":         lowStock,
		"TotalValue":       totalValue,
		"CriticalProducts": criticalProducts,
		"Flashes":          getFlashes(c),
	})
}

func inventoryHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	pageStr := c.DefaultQuery("page", "1")
	searchQuery := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	var totalRecords int
	var rows *sql.Rows
	var err error

	if searchQuery != "" {
		// Deteksi apakah user mencari Part Number atau Kata Kunci biasa
		isPartNumberSearch := strings.Contains(searchQuery, "-") || strings.HasPrefix(strings.ToUpper(searchQuery), "DNE")

		if isPartNumberSearch {
			// B-TREE INDEX SCAN (Sangat Cepat - Prefix Match)
			// Hanya gunakan '%' di belakang agar index 'idx_part_number' bekerja secara optimal!
			searchPattern := searchQuery + "%"
			db.QueryRow("SELECT COUNT(*) FROM products WHERE part_number LIKE ?", searchPattern).Scan(&totalRecords)
			rows, err = db.Query("SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, location FROM products WHERE part_number LIKE ? ORDER BY part_number ASC LIMIT ? OFFSET ?", searchPattern, limit, offset)
		} else {
			// FULLTEXT SEARCH (Inverted Index - Sangat Cepat untuk kata)
			if len(searchQuery) >= 3 {
				// Format ke +word1* +word2* agar setiap kata menjadi mandatory, mencegah return "semuanya"
				words := strings.Fields(searchQuery)
				var ftWords []string
				for _, w := range words {
					ftWords = append(ftWords, "+"+w+"*")
				}
				ftQuery := strings.Join(ftWords, " ")

				db.QueryRow("SELECT COUNT(*) FROM products WHERE MATCH(part_number, description, category) AGAINST(? IN BOOLEAN MODE)", ftQuery).Scan(&totalRecords)
				// Hapus ORDER BY id DESC untuk menghindari filesort yang sangat lambat, biarkan engine optimal
				rows, err = db.Query("SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, location FROM products WHERE MATCH(part_number, description, category) AGAINST(? IN BOOLEAN MODE) LIMIT ? OFFSET ?", ftQuery, limit, offset)
			} else {
				// Kata kurang dari 3 huruf (sangat jarang terjadi, fallback aman)
				searchPattern := "%" + searchQuery + "%"
				db.QueryRow("SELECT COUNT(*) FROM products WHERE description LIKE ? OR category LIKE ?", searchPattern, searchPattern).Scan(&totalRecords)
				rows, err = db.Query("SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, location FROM products WHERE description LIKE ? OR category LIKE ? LIMIT ? OFFSET ?", searchPattern, searchPattern, limit, offset)
			}
		}
	} else {
		// Gunakan cached count — instan!
		totalRecords = dashCache.TotalCount
		rows, err = db.Query("SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, location FROM products ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(limit)))

	var products []Product
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p Product
			rows.Scan(&p.ID, &p.PartNumber, &p.Description, &p.Category, &p.Quantity, &p.MinStockLevel, &p.CapitalPrice, &p.Location)
			products = append(products, p)
		}
	}

	// Gunakan cached low stock count — instan!
	lowStockCount := dashCache.LowStock + dashCache.OutOfStock

	// Build Page Numbers Array (window of 5 pages)
	var pageNumbers []int
	startPage := page - 2
	if startPage < 1 {
		startPage = 1
	}
	endPage := startPage + 4
	if endPage > totalPages {
		endPage = totalPages
		startPage = endPage - 4
		if startPage < 1 {
			startPage = 1
		}
	}
	for i := startPage; i <= endPage; i++ {
		pageNumbers = append(pageNumbers, i)
	}

	c.HTML(http.StatusOK, "inventory.html", gin.H{
		"User":          user,
		"ActiveMenu":    "inventory",
		"Title":         "Manajemen Inventaris",
		"Products":      products,
		"CurrentPage":   page,
		"TotalPages":    totalPages,
		"PageNumbers":   pageNumbers,
		"SearchQuery":   searchQuery,
		"Flashes":       getFlashes(c),
		"LowStockCount": lowStockCount,
	})
}

func printLabelsHandler(c *gin.Context) {
	rows, err := db.Query("SELECT id, part_number, description FROM products ORDER BY id DESC LIMIT 100")
	var products []Product
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p Product
			rows.Scan(&p.ID, &p.PartNumber, &p.Description)
			products = append(products, p)
		}
	}
	c.HTML(http.StatusOK, "labels.html", gin.H{
		"Products": products,
	})
}

// ==========================================
// AUTO BACKUP SYSTEM (PURE GO CSV + ZIP)
// ==========================================
func startDailyBackup() {
	// Jalankan setiap 24 Jam
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			<-ticker.C
			log.Println("Memulai Auto-Backup Harian...")
			runBackupLogic()
		}
	}()
}

func runBackupLogic() (string, error) {
	backupDir := "backups"
	os.MkdirAll(backupDir, os.ModePerm)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	zipName := filepath.Join(backupDir, fmt.Sprintf("Backup_JackSound_%s.zip", timestamp))

	zipFile, err := os.Create(zipName)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	tables := []string{"products", "inventory_transactions", "expenses", "users"}

	for _, table := range tables {
		err := exportTableToZip(archive, table)
		if err != nil {
			log.Printf("Gagal membackup tabel %s: %v", table, err)
		}
	}

	log.Printf("Backup berhasil disimpan ke %s", zipName)
	return zipName, nil
}

func exportTableToZip(archive *zip.Writer, tableName string) error {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	f, err := archive.Create(tableName + ".csv")
	if err != nil {
		return err
	}

	writer := csv.NewWriter(f)
	writer.Write(columns)

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			continue
		}
		var record []string
		for _, col := range values {
			if col == nil {
				record = append(record, "NULL")
			} else {
				record = append(record, string(col))
			}
		}
		writer.Write(record)
	}
	writer.Flush()
	return nil
}

func manualBackupHandler(c *gin.Context) {
	zipPath, err := runBackupLogic()
	if err != nil {
		session := sessions.Default(c)
		session.AddFlash("Gagal melakukan backup manual: " + err.Error())
		session.Save()
		c.Redirect(http.StatusFound, "/users")
		return
	}

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(zipPath)))
	c.Writer.Header().Set("Content-Type", "application/zip")
	c.File(zipPath)
}

// Product Management Handlers (Admin Only)
func newProductHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	
	rows, _ := db.Query("SELECT id, name FROM categories ORDER BY name ASC")
	var categories []Category
	defer rows.Close()
	for rows.Next() {
		var cat Category
		rows.Scan(&cat.ID, &cat.Name)
		categories = append(categories, cat)
	}

	c.HTML(http.StatusOK, "create.html", gin.H{
		"User":       user,
		"ActiveMenu": "inventory",
		"Title":      "Tambah Produk Baru",
		"Categories": categories,
		"Flashes":    getFlashes(c),
	})
}

func createProductHandler(c *gin.Context) {
	partNumber := c.PostForm("part_number")
	description := c.PostForm("description")
	category := c.PostForm("category")
	quantity, _ := strconv.Atoi(c.PostForm("quantity"))
	minStockLevel, _ := strconv.Atoi(c.PostForm("min_stock_level"))
	capStr := strings.ReplaceAll(strings.ReplaceAll(c.PostForm("capital_price"), ".", ""), ",", ".")
	sellStr := strings.ReplaceAll(strings.ReplaceAll(c.PostForm("selling_price"), ".", ""), ",", ".")
	capitalPrice, _ := strconv.ParseFloat(capStr, 64)
	sellingPrice, _ := strconv.ParseFloat(sellStr, 64)
	location := c.PostForm("location")

	res, err := db.Exec("INSERT INTO products (part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		partNumber, description, category, quantity, minStockLevel, capitalPrice, sellingPrice, location)

	if err == nil && quantity > 0 {
		user := c.MustGet("user").(*User)
		productID, _ := res.LastInsertId()
		totalValue := capitalPrice * float64(quantity)
		db.Exec("INSERT INTO inventory_transactions (product_id, user_id, transaction_type, quantity, total_value, notes) VALUES (?, ?, ?, ?, ?, ?)", productID, user.ID, "IN", quantity, totalValue, "Stok Awal")
	}

	session := sessions.Default(c)
	if err != nil {
		session.AddFlash("Gagal menambahkan produk: " + err.Error())
	} else {
		session.AddFlash("Produk berhasil ditambahkan!")
	}
	session.Save()

	c.Redirect(http.StatusFound, "/inventory")
}

func editProductHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	id := c.Param("id")
	var product Product
	err := db.QueryRow("SELECT id, part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location FROM products WHERE id = ?", id).
		Scan(&product.ID, &product.PartNumber, &product.Description, &product.Category, &product.Quantity, &product.MinStockLevel, &product.CapitalPrice, &product.SellingPrice, &product.Location)

	if err != nil {
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	rows, _ := db.Query("SELECT id, name FROM categories ORDER BY name ASC")
	var categories []Category
	defer rows.Close()
	for rows.Next() {
		var cat Category
		rows.Scan(&cat.ID, &cat.Name)
		categories = append(categories, cat)
	}

	c.HTML(http.StatusOK, "edit.html", gin.H{
		"User":       user,
		"ActiveMenu": "inventory",
		"Title":      "Edit Produk",
		"Product":    product,
		"Categories": categories,
		"Flashes":    getFlashes(c),
	})
}

func updateProductHandler(c *gin.Context) {
	id := c.Param("id")
	partNumber := c.PostForm("part_number")
	description := c.PostForm("description")
	category := c.PostForm("category")
	minStockLevel, _ := strconv.Atoi(c.PostForm("min_stock_level"))
	capStr := strings.ReplaceAll(strings.ReplaceAll(c.PostForm("capital_price"), ".", ""), ",", ".")
	sellStr := strings.ReplaceAll(strings.ReplaceAll(c.PostForm("selling_price"), ".", ""), ",", ".")
	capitalPrice, _ := strconv.ParseFloat(capStr, 64)
	sellingPrice, _ := strconv.ParseFloat(sellStr, 64)
	location := c.PostForm("location")

	// Note: Quantity is NOT updated here anymore, use Transactions!
	_, err := db.Exec("UPDATE products SET part_number = ?, description = ?, category = ?, min_stock_level = ?, capital_price = ?, selling_price = ?, location = ? WHERE id = ?",
		partNumber, description, category, minStockLevel, capitalPrice, sellingPrice, location, id)

	session := sessions.Default(c)
	if err != nil {
		session.AddFlash("Gagal memperbarui produk: " + err.Error())
	} else {
		session.AddFlash("Produk berhasil diperbarui!")
	}
	session.Save()

	c.Redirect(http.StatusFound, "/inventory")
}

func deleteProductHandler(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM products WHERE id = ?", id)

	session := sessions.Default(c)
	if err != nil {
		session.AddFlash("Gagal menghapus produk: " + err.Error())
	} else {
		session.AddFlash("Produk berhasil dihapus!")
	}
	session.Save()

	c.Redirect(http.StatusFound, "/inventory")
}

// Transaction Handlers (Stock In / Stock Out)
func transactionGetHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	id := c.Param("id")
	var p Product
	err := db.QueryRow("SELECT id, part_number, description, quantity FROM products WHERE id = ?", id).
		Scan(&p.ID, &p.PartNumber, &p.Description, &p.Quantity)

	if err != nil {
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	c.HTML(http.StatusOK, "transaction.html", gin.H{
		"User":       user,
		"ActiveMenu": "inventory",
		"Title":      "Transaksi Stok",
		"Product":    p,
		"Flashes":    getFlashes(c),
	})
}

func transactionPostHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	id := c.Param("id")
	transType := c.PostForm("transaction_type") // "IN" or "OUT"
	quantity, _ := strconv.Atoi(c.PostForm("quantity"))
	notes := c.PostForm("notes")

	session := sessions.Default(c)

	if quantity <= 0 {
		session.AddFlash("Kuantitas harus lebih dari 0!")
		session.Save()
		c.Redirect(http.StatusFound, "/product/"+id+"/transaction")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		session.AddFlash("System Error: " + err.Error())
		session.Save()
		c.Redirect(http.StatusFound, "/product/"+id+"/transaction")
		return
	}

	var currentQty int
	var capPrice, sellPrice float64
	err = tx.QueryRow("SELECT quantity, capital_price, selling_price FROM products WHERE id = ? FOR UPDATE", id).Scan(&currentQty, &capPrice, &sellPrice)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Product not found")
		session.Save()
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	if transType == "OUT" && quantity > currentQty {
		tx.Rollback()
		session.AddFlash("Stok tidak mencukupi untuk dikeluarkan!")
		session.Save()
		c.Redirect(http.StatusFound, "/product/"+id+"/transaction")
		return
	}

	newQty := currentQty
	totalValue := 0.0
	if transType == "IN" {
		newQty += quantity
		totalValue = capPrice * float64(quantity)
	} else if transType == "OUT" {
		newQty -= quantity
		totalValue = sellPrice * float64(quantity)
	} else {
		tx.Rollback()
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	// Update product quantity
	_, err = tx.Exec("UPDATE products SET quantity = ? WHERE id = ?", newQty, id)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Gagal update stok.")
		session.Save()
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	// Insert transaction log
	_, err = tx.Exec("INSERT INTO inventory_transactions (product_id, user_id, transaction_type, quantity, total_value, notes) VALUES (?, ?, ?, ?, ?, ?)", id, user.ID, transType, quantity, totalValue, notes)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Gagal mencatat log transaksi.")
		session.Save()
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	tx.Commit()
	session.AddFlash(fmt.Sprintf("Transaksi Stok %s berhasil! Stok sekarang: %d", transType, newQty))
	session.Save()
	c.Redirect(http.StatusFound, "/inventory")
}

func historyHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	
	rows, err := db.Query(`
		SELECT t.id, p.part_number, p.description, u.username, t.transaction_type, t.quantity, t.notes, DATE_FORMAT(t.transaction_date, '%Y-%m-%d %H:%i') as date
		FROM inventory_transactions t
		JOIN products p ON t.product_id = p.id
		LEFT JOIN users u ON t.user_id = u.id
		ORDER BY t.transaction_date DESC LIMIT 50
	`)
	
	var transactions []Transaction
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t Transaction
			var uname sql.NullString
			var nts sql.NullString
			rows.Scan(&t.ID, &t.PartNumber, &t.Description, &uname, &t.TransactionType, &t.Quantity, &nts, &t.Date)
			t.Username = uname.String
			t.Notes = nts.String
			if t.Username == "" { t.Username = "System" }
			transactions = append(transactions, t)
		}
	}

	c.HTML(http.StatusOK, "history.html", gin.H{
		"User":       user,
		"ActiveMenu": "history",
		"Title":      "Riwayat Transaksi (Audit Trail)",
		"Transactions": transactions,
		"Flashes":    getFlashes(c),
	})
}

// Export PO Handler
func exportPOHandler(c *gin.Context) {
	rows, err := db.Query("SELECT part_number, description, quantity, min_stock_level, capital_price, location FROM products WHERE quantity <= min_stock_level ORDER BY quantity ASC")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating export")
		return
	}
	defer rows.Close()

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	
	// Write Header
	w.Write([]string{"Part Number", "Deskripsi", "Stok Saat Ini", "Batas Minimum", "Harga Modal", "Lokasi", "Saran Restock Qty"})

	for rows.Next() {
		var p Product
		rows.Scan(&p.PartNumber, &p.Description, &p.Quantity, &p.MinStockLevel, &p.CapitalPrice, &p.Location)
		
		suggestedRestock := p.MinStockLevel * 2
		if suggestedRestock < 10 { suggestedRestock = 10 }
		
		w.Write([]string{
			p.PartNumber,
			p.Description,
			strconv.Itoa(p.Quantity),
			strconv.Itoa(p.MinStockLevel),
			fmt.Sprintf("%.2f", p.CapitalPrice),
			p.Location,
			strconv.Itoa(suggestedRestock),
		})
	}
	w.Flush()

	filename := fmt.Sprintf("Purchase_Order_%s.csv", time.Now().Format("20060102"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", b.Bytes())
}

// QR Code Generator
func generateQRCodeHandler(c *gin.Context) {
	id := c.Param("id")
	var partNumber string
	err := db.QueryRow("SELECT part_number FROM products WHERE id = ?", id).Scan(&partNumber)
	if err != nil {
		c.String(http.StatusNotFound, "Product not found")
		return
	}

	png, err := qrcode.Encode(partNumber, qrcode.Medium, 256)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating QR Code")
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}

func posGetHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	c.HTML(http.StatusOK, "pos.html", gin.H{
		"User":       user,
		"ActiveMenu": "pos",
		"Title":      "Kasir / Point of Sale",
		"Flashes":    getFlashes(c),
	})
}

func posPostHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	partNumber := c.PostForm("part_number")
	quantityStr := c.PostForm("quantity")
	transType := c.PostForm("transaction_type")
	notes := c.PostForm("notes")

	session := sessions.Default(c)
	qty, err := strconv.Atoi(quantityStr)
	if err != nil || qty <= 0 {
		session.AddFlash("Kuantitas tidak valid!")
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		session.AddFlash("Sistem Error!")
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	var id, currentQty int
	var capPrice, sellPrice float64
	err = tx.QueryRow("SELECT id, quantity, capital_price, selling_price FROM products WHERE part_number = ? FOR UPDATE", partNumber).Scan(&id, &currentQty, &capPrice, &sellPrice)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Produk tidak ditemukan: " + partNumber)
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	if transType == "OUT" && qty > currentQty {
		tx.Rollback()
		session.AddFlash("Stok tidak mencukupi untuk: " + partNumber)
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	newQty := currentQty
	totalValue := 0.0
	if transType == "IN" {
		newQty += qty
		totalValue = capPrice * float64(qty)
	} else {
		newQty -= qty
		totalValue = sellPrice * float64(qty)
	}

	_, err = tx.Exec("UPDATE products SET quantity = ? WHERE id = ?", newQty, id)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Gagal update stok.")
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	_, err = tx.Exec("INSERT INTO inventory_transactions (product_id, user_id, transaction_type, quantity, total_value, notes) VALUES (?, ?, ?, ?, ?, ?)", id, user.ID, transType, qty, totalValue, notes)
	if err != nil {
		tx.Rollback()
		session.AddFlash("Gagal mencatat transaksi.")
		session.Save()
		c.Redirect(http.StatusFound, "/pos")
		return
	}

	tx.Commit()
	session.AddFlash(fmt.Sprintf("Sukses! %s sebanyak %d %s. Nilai Transaksi: Rp %s", partNumber, qty, transType, formatRupiah(totalValue)))
	session.Save()
	c.Redirect(http.StatusFound, "/pos")
}

func accountingHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	
	// Only Admin can see full accounting
	if user.Role != "admin" {
		session := sessions.Default(c)
		session.AddFlash("Akses ditolak: Hanya Admin yang bisa mengakses Akuntansi.")
		session.Save()
		c.Redirect(http.StatusFound, "/")
		return
	}

	var revenue, cogs, expensesTotal float64

	// Calculate Revenue (Selling Price * Qty OUT)
	db.QueryRow("SELECT COALESCE(SUM(total_value), 0) FROM inventory_transactions WHERE transaction_type = 'OUT'").Scan(&revenue)
	
	// Calculate COGS (Capital Price * Qty OUT)
	db.QueryRow(`
		SELECT COALESCE(SUM(p.capital_price * t.quantity), 0) 
		FROM inventory_transactions t 
		JOIN products p ON t.product_id = p.id 
		WHERE t.transaction_type = 'OUT'
	`).Scan(&cogs)

	// Calculate Expenses
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM expenses").Scan(&expensesTotal)

	grossProfit := revenue - cogs
	netProfit := grossProfit - expensesTotal

	rows, _ := db.Query("SELECT e.id, e.description, e.amount, DATE_FORMAT(e.expense_date, '%Y-%m-%d %H:%i') as date, u.username FROM expenses e LEFT JOIN users u ON e.user_id = u.id ORDER BY e.expense_date DESC LIMIT 20")
	var expenses []Expense
	defer rows.Close()
	for rows.Next() {
		var e Expense
		var uname sql.NullString
		rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &uname)
		e.Username = uname.String
		if e.Username == "" { e.Username = "System" }
		expenses = append(expenses, e)
	}

	c.HTML(http.StatusOK, "accounting.html", gin.H{
		"User":          user,
		"ActiveMenu":    "accounting",
		"Title":         "Akuntansi & Laba Rugi",
		"Revenue":       revenue,
		"COGS":          cogs,
		"GrossProfit":   grossProfit,
		"ExpensesTotal": expensesTotal,
		"NetProfit":     netProfit,
		"Expenses":      expenses,
		"Flashes":       getFlashes(c),
	})
}

func expensePostHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	desc := c.PostForm("description")
	amtStr := strings.ReplaceAll(strings.ReplaceAll(c.PostForm("amount"), ".", ""), ",", ".")
	amount, _ := strconv.ParseFloat(amtStr, 64)

	session := sessions.Default(c)
	if user.Role != "admin" {
		session.AddFlash("Hanya Admin yang bisa mencatat pengeluaran.")
		session.Save()
		c.Redirect(http.StatusFound, "/accounting")
		return
	}

	_, err := db.Exec("INSERT INTO expenses (description, amount, user_id) VALUES (?, ?, ?)", desc, amount, user.ID)
	if err != nil {
		session.AddFlash("Gagal mencatat pengeluaran.")
	} else {
		session.AddFlash("Pengeluaran berhasil dicatat!")
	}
	session.Save()
	c.Redirect(http.StatusFound, "/accounting")
}

// POS APIs
func apiGetProductHandler(c *gin.Context) {
	partNumber := c.Param("part_number")
	var p Product
	err := db.QueryRow("SELECT id, part_number, description, category, quantity, capital_price, selling_price FROM products WHERE part_number = ?", partNumber).
		Scan(&p.ID, &p.PartNumber, &p.Description, &p.Category, &p.Quantity, &p.CapitalPrice, &p.SellingPrice)
	if err != nil {
		c.JSON(404, gin.H{"error": "Produk tidak ditemukan"})
		return
	}
	c.JSON(200, p)
}

type CheckoutItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"qty"`
}

type CheckoutPayload struct {
	Type  string         `json:"type"`
	Notes string         `json:"notes"`
	Items []CheckoutItem `json:"items"`
}

func posCheckoutHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	var payload CheckoutPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"error": "Invalid payload"})
		return
	}
	if len(payload.Items) == 0 {
		c.JSON(400, gin.H{"error": "Keranjang kosong!"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": "Sistem Error"})
		return
	}

	receiptID := fmt.Sprintf("TRX-%d", time.Now().UnixNano())

	for _, item := range payload.Items {
		var currentQty int
		var capPrice, sellPrice float64
		err = tx.QueryRow("SELECT quantity, capital_price, selling_price FROM products WHERE id = ? FOR UPDATE", item.ProductID).Scan(&currentQty, &capPrice, &sellPrice)
		if err != nil {
			tx.Rollback()
			c.JSON(400, gin.H{"error": fmt.Sprintf("Gagal baca produk ID %d", item.ProductID)})
			return
		}

		if payload.Type == "OUT" && item.Quantity > currentQty {
			tx.Rollback()
			c.JSON(400, gin.H{"error": "Stok tidak mencukupi untuk salah satu produk."})
			return
		}

		newQty := currentQty
		totalValue := 0.0
		if payload.Type == "IN" {
			newQty += item.Quantity
			totalValue = capPrice * float64(item.Quantity)
		} else {
			newQty -= item.Quantity
			totalValue = sellPrice * float64(item.Quantity)
		}

		_, err = tx.Exec("UPDATE products SET quantity = ? WHERE id = ?", newQty, item.ProductID)
		if err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Gagal update stok."})
			return
		}

		_, err = tx.Exec("INSERT INTO inventory_transactions (receipt_id, product_id, user_id, transaction_type, quantity, total_value, notes) VALUES (?, ?, ?, ?, ?, ?, ?)", receiptID, item.ProductID, user.ID, payload.Type, item.Quantity, totalValue, payload.Notes)
		if err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Gagal mencatat transaksi."})
			return
		}
	}

	tx.Commit()
	c.JSON(200, gin.H{"success": true, "receipt_id": receiptID})
}

type ReceiptItem struct {
	PartNumber string
	Desc       string
	Qty        int
	Price      float64
	Total      float64
}

func receiptHandler(c *gin.Context) {
	receiptID := c.Param("receipt_id")
	
	rows, err := db.Query("SELECT p.part_number, p.description, t.quantity, t.transaction_type, t.total_value, t.transaction_date, u.username FROM inventory_transactions t JOIN products p ON t.product_id = p.id LEFT JOIN users u ON t.user_id = u.id WHERE t.receipt_id = ?", receiptID)
	if err != nil {
		c.String(404, "Struk tidak ditemukan")
		return
	}
	defer rows.Close()

	var items []ReceiptItem
	var total float64
	var tType, date, cashier string

	for rows.Next() {
		var item ReceiptItem
		var tt string
		var uname sql.NullString
		rows.Scan(&item.PartNumber, &item.Desc, &item.Qty, &tt, &item.Total, &date, &uname)
		
		item.Price = item.Total / float64(item.Qty)
		items = append(items, item)
		total += item.Total
		tType = tt
		cashier = uname.String
	}

	if len(items) == 0 {
		c.String(404, "Struk tidak ditemukan")
		return
	}

	c.HTML(http.StatusOK, "receipt.html", gin.H{
		"ReceiptID": receiptID,
		"Type":      tType,
		"Date":      date,
		"Cashier":   cashier,
		"Items":     items,
		"Total":     total,
	})
}

// Users Management Handlers
func usersGetHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)
	rows, _ := db.Query("SELECT id, username, role FROM users")
	var users []User
	defer rows.Close()
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Role)
		users = append(users, u)
	}

	c.HTML(http.StatusOK, "users.html", gin.H{
		"User":       user,
		"ActiveMenu": "users",
		"Title":      "Manajemen Karyawan",
		"Users":      users,
		"Flashes":    getFlashes(c),
	})
}

func usersPostHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	role := c.PostForm("role")

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", username, hash, role)

	session := sessions.Default(c)
	if err != nil {
		session.AddFlash("Gagal menambah karyawan. Username mungkin sudah ada.")
	} else {
		session.AddFlash("Karyawan berhasil ditambahkan.")
	}
	session.Save()
	c.Redirect(http.StatusFound, "/users")
}

func usersDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	// Prevent self-delete or deleting main admin
	if id == "1" || id == strconv.Itoa(c.MustGet("user").(*User).ID) {
		session := sessions.Default(c)
		session.AddFlash("Tidak dapat menghapus akun ini.")
		session.Save()
		c.Redirect(http.StatusFound, "/users")
		return
	}
	db.Exec("DELETE FROM users WHERE id = ?", id)
	
	session := sessions.Default(c)
	session.AddFlash("Akun karyawan berhasil dihapus.")
	session.Save()
	c.Redirect(http.StatusFound, "/users")
}

func usersResetHandler(c *gin.Context) {
	id := c.Param("id")
	newPassword := "password123" // Default reset password
	hash, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	
	session := sessions.Default(c)
	session.AddFlash("Password berhasil di-reset menjadi: " + newPassword)
	session.Save()
	c.Redirect(http.StatusFound, "/users")
}

func exportPnlHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/csv")
	c.Writer.Header().Set("Content-Disposition", `attachment; filename="Laporan_Keuangan_JackSound.csv"`)

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write Headers
	writer.Write([]string{"Kategori", "Deskripsi", "Nominal (Rp)", "Tanggal", "Penanggung Jawab"})

	// Fetch Revenues (Penjualan)
	rows, err := db.Query("SELECT p.description, t.total_value, t.transaction_date, COALESCE(u.username, 'System') FROM inventory_transactions t JOIN products p ON t.product_id = p.id LEFT JOIN users u ON t.user_id = u.id WHERE t.transaction_type = 'OUT'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var desc, date, uname string
			var val float64
			rows.Scan(&desc, &val, &date, &uname)
			writer.Write([]string{"Pendapatan (Penjualan)", desc, fmt.Sprintf("%.2f", val), date, uname})
		}
	}

	// Fetch Expenses (Pengeluaran)
	rows2, err2 := db.Query("SELECT expenses.description, expenses.amount, expenses.expense_date, COALESCE(u.username, 'System') FROM expenses LEFT JOIN users u ON expenses.user_id = u.id")
	if err2 == nil {
		defer rows2.Close()
		for rows2.Next() {
			var desc, date, uname string
			var val float64
			rows2.Scan(&desc, &val, &date, &uname)
			writer.Write([]string{"Pengeluaran Operasional", desc, fmt.Sprintf("%.2f", val), date, uname})
		}
	}
}

// Category Handlers (Admin)
func categoriesHandler(c *gin.Context) {
	user := c.MustGet("user").(*User)

	rows, err := db.Query("SELECT id, name FROM categories ORDER BY id ASC")
	var categories []Category
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat Category
			rows.Scan(&cat.ID, &cat.Name)
			categories = append(categories, cat)
		}
	}

	c.HTML(http.StatusOK, "categories.html", gin.H{
		"User":       user,
		"ActiveMenu": "categories",
		"Title":      "Manajemen Kategori",
		"Categories": categories,
		"Flashes":    getFlashes(c),
	})
}

func categoriesPostHandler(c *gin.Context) {
	name := c.PostForm("name")
	if name != "" {
		_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", name)
		session := sessions.Default(c)
		if err != nil {
			session.AddFlash("Gagal: Kategori sudah ada atau error.")
		} else {
			session.AddFlash("Kategori berhasil ditambahkan.")
		}
		session.Save()
	}
	c.Redirect(http.StatusFound, "/categories")
}

func categoriesDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM categories WHERE id = ? AND name != 'Uncategorized'", id)
	session := sessions.Default(c)
	if err != nil {
		session.AddFlash("Gagal menghapus kategori.")
	} else {
		session.AddFlash("Kategori berhasil dihapus.")
	}
	session.Save()
	c.Redirect(http.StatusFound, "/categories")
}
