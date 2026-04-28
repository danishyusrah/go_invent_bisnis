package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Brand-brand real per kategori
var brands = map[string][]string{
	"Speaker Aktif": {"JBL", "Yamaha", "Bose", "Harman Kardon", "Mackie", "Behringer", "RCF", "QSC", "Electro-Voice", "Turbosound", "dB Technologies", "LD Systems", "Alto Professional", "Samson", "PreSonus"},
	"Speaker Pasif": {"JBL", "Yamaha", "Peavey", "Wharfedale Pro", "RCF", "Electro-Voice", "Turbosound", "Community", "dB Technologies", "Tannoy", "Martin Audio", "Nexo", "FBT", "DAS Audio", "Dynacord"},
	"Amplifier": {"Crown", "QSC", "Yamaha", "Behringer", "Powersoft", "Lab Gruppen", "Crest Audio", "Dynacord", "Ashly", "Peavey", "Denon", "Marantz", "NAD", "Cambridge Audio", "Rotel"},
	"Microphone": {"Shure", "Sennheiser", "Audio-Technica", "AKG", "Rode", "Neumann", "Beyerdynamic", "Electro-Voice", "Blue Microphones", "MXL", "Samson", "CAD Audio", "Heil Sound", "SE Electronics", "Lewitt"},
	"Mixer Audio": {"Yamaha", "Allen & Heath", "Behringer", "Mackie", "Soundcraft", "PreSonus", "Midas", "Tascam", "Zoom", "Roland", "Dynacord", "Peavey", "Samson", "Alto Professional", "Phonic"},
	"Headphone & Earphone": {"Sony", "Sennheiser", "Audio-Technica", "Beyerdynamic", "AKG", "Shure", "JBL", "Bose", "Jabra", "Plantronics", "HyperX", "SteelSeries", "Razer", "Corsair", "Focal"},
	"Kabel & Konektor": {"Canare", "Belden", "Neutrik", "Mogami", "Hosa", "Planet Waves", "Monster Cable", "Switchcraft", "Amphenol", "Klotz", "Cordial", "Pro Co", "Whirlwind", "Livewire", "Rapco Horizon"},
	"Lighting & Stage": {"Chauvet DJ", "ADJ", "Martin Lighting", "Elation", "Robe", "Clay Paky", "Ayrton", "GLP", "Vari-Lite", "ETC", "Philips", "Showtec", "Eurolite", "Stairville", "Cameo"},
	"Aksesoris Audio": {"K&M", "Ultimate Support", "On-Stage", "Hercules", "Gator Cases", "SKB", "Pelican", "Auralex", "Primacoustic", "IsoAcoustics", "Radial Engineering", "DBX", "Lexicon", "TC Electronic", "Furman"},
	"Alat Musik Digital": {"Roland", "Yamaha", "Korg", "Nord", "Casio", "Kawai", "Arturia", "Native Instruments", "Akai", "M-Audio", "Novation", "Alesis", "Kurzweil", "Dave Smith", "Sequential"},
	"TV & Display": {"Samsung", "LG", "Sony", "TCL", "Hisense", "Panasonic", "Sharp", "Toshiba", "Philips", "ViewSonic", "BenQ", "NEC", "Optoma", "Epson", "JVC"},
	"Sound System Portable": {"JBL", "Bose", "Sony", "Marshall", "Ultimate Ears", "Anker Soundcore", "Harman Kardon", "Bang & Olufsen", "Sonos", "Denon", "Tribit", "Tronsmart", "Xiaomi", "LG XBOOM", "Edifier"},
	"Perekam & Interface Audio": {"Focusrite", "Universal Audio", "PreSonus", "MOTU", "RME", "Steinberg", "Tascam", "Zoom", "Behringer", "Audient", "SSL", "Apogee", "Antelope Audio", "IK Multimedia", "Mackie"},
	"DJ Equipment": {"Pioneer DJ", "Numark", "Denon DJ", "Native Instruments", "Rane", "Allen & Heath", "Reloop", "Hercules DJ", "Gemini", "Stanton", "Vestax", "Akai Professional", "Roland DJ", "Mixars", "Phase"},
	"CCTV & Security": {"Hikvision", "Dahua", "SPC", "Avtech", "Samsung Hanwha", "Uniview", "CP Plus", "Ezviz", "Reolink", "TP-Link VIGI", "Axis", "Bosch Security", "Honeywell", "Vivotek", "Pelco"},
}

var productTypes = map[string][]string{
	"Speaker Aktif": {
		"Speaker Aktif 8 inch 2-Way", "Speaker Aktif 10 inch", "Speaker Aktif 12 inch 1000W",
		"Speaker Aktif 15 inch Full Range", "Subwoofer Aktif 18 inch", "Column Speaker Aktif",
		"Speaker Monitor Studio 5 inch", "Speaker Monitor Studio 8 inch", "Speaker Ceiling Aktif",
		"Speaker Portable Aktif 12 inch", "Speaker Aktif Line Array", "Speaker Wall Mount Aktif",
		"Studio Monitor Nearfield", "Speaker Aktif Compact 6 inch", "Subwoofer Aktif 12 inch",
	},
	"Speaker Pasif": {
		"Speaker Pasif 12 inch 500W", "Speaker Pasif 15 inch Full Range", "Subwoofer Pasif 18 inch",
		"Speaker Pasif 10 inch Compact", "Column Speaker Pasif", "Speaker Ceiling Pasif 6 inch",
		"Speaker Pasif Line Array", "Speaker Monitor Pasif Wedge", "Speaker Pasif 2-Way 800W",
		"Horn Tweeter Pasif", "Speaker Wall Mount Pasif", "Subwoofer Pasif 21 inch",
		"Speaker Pasif Coaxial 8 inch", "Speaker Pasif Horn Loaded", "Speaker Toa Pasif 10W",
	},
	"Amplifier": {
		"Power Amplifier 2x500W", "Power Amplifier 2x1000W", "Power Amplifier 4-Channel",
		"Amplifier Karaoke 2 Channel", "Integrated Amplifier HiFi", "Power Amplifier Class D 2000W",
		"Amplifier Mixer Combo", "Power Amplifier Class H 1500W", "Mini Amplifier Bluetooth",
		"Power Amplifier Touring 2x2000W", "Amplifier Mono Block 5000W", "Stereo Receiver Amplifier",
		"Power Amplifier DSP Built-in", "Amplifier Install 100V Line", "Power Amplifier 2x300W",
	},
	"Microphone": {
		"Microphone Dynamic Vocal", "Microphone Condenser Large Diaphragm", "Wireless Mic Handheld UHF",
		"Wireless Mic Clip On Lavalier", "Wireless Mic Headset", "Mic Condenser Small Diaphragm",
		"Mic Instrument Drum Set 7pcs", "Shotgun Microphone", "USB Microphone Podcasting",
		"Microphone Gooseneck Conference", "Wireless Mic Dual Handheld", "Ribbon Microphone",
		"Boundary Microphone Flat", "Microphone Harmonica", "Wireless Mic System 4-Channel",
	},
	"Mixer Audio": {
		"Mixer Analog 12 Channel", "Mixer Analog 16 Channel", "Mixer Digital 32 Channel",
		"Mixer Analog 8 Channel Compact", "Mixer USB Recording 4 Channel", "Mixer Powered 10 Channel",
		"Mixer Digital Touchscreen 24Ch", "Mixer Rack Mount 1U 8Ch", "Mixer DJ 4 Channel",
		"Mixer Broadcast Streaming", "Mixer Analog 24 Channel", "Mini Mixer 4 Channel Passive",
		"Mixer Digital Stagebox 16Ch", "Mixer Install Zone 8 Input", "Mixer Analog 6 Channel",
	},
	"Headphone & Earphone": {
		"Headphone Studio Closed Back", "Headphone Studio Open Back", "Headphone DJ Over-Ear",
		"Headphone Wireless Bluetooth ANC", "Earphone In-Ear Monitor", "Headphone Gaming 7.1 Surround",
		"Earphone True Wireless TWS", "Headphone Monitoring Semi-Open", "Earphone Wired HiFi",
		"Headphone Reference Premium", "Headset Communication USB-C", "Earphone Custom Molded",
		"Headphone Portable Foldable", "Earphone Stage Monitor Dual Driver", "Headphone Wireless Studio",
	},
	"Kabel & Konektor": {
		"Kabel XLR Male to Female 5M", "Kabel XLR 10M Premium", "Kabel Jack 6.35mm TRS 3M",
		"Kabel Speaker 2x1.5mm 100M", "Kabel RCA to RCA 2M", "Konektor XLR Male Neutrik",
		"Konektor XLR Female Neutrik", "Konektor Jack 6.35mm TRS", "Kabel Snake 8 Channel 15M",
		"Kabel Snake 16 Channel 30M", "Kabel HDMI 2.1 4K 3M", "Kabel Optical Toslink 2M",
		"Kabel Speaker Speakon 10M", "Konektor Speakon 4-Pole", "Kabel Mic XLR 3M Economy",
	},
	"Lighting & Stage": {
		"Moving Head Beam 230W", "Moving Head Wash 36x10W", "Moving Head Spot 150W LED",
		"Par LED RGBW 54x3W", "Par LED COB 200W Warm White", "Laser RGB 500mW",
		"Strobe LED 1500W DMX", "Fog Machine 1500W Timer", "Haze Machine 600W",
		"LED Bar 8x10W RGBW", "Follow Spot 1200W", "LED Panel Video 600 Bi-Color",
		"DMX Controller 512 Channel", "Truss Aluminium 3M Kotak", "Standing Tripod Lighting 3.5M",
	},
	"Aksesoris Audio": {
		"Stand Mic Boom Lantai", "Stand Mic Meja Desktop", "Stand Speaker Tripod Pair",
		"Hardcase Mixer 12U Rack", "Softcase Speaker 15 inch", "Rack Case 6U Rolling",
		"Pop Filter Microphone", "Shock Mount Universal", "Windscreen Busa Mic",
		"DI Box Active 2 Channel", "DI Box Passive 1 Channel", "Power Conditioner Rack 1U",
		"Acoustic Foam Panel 12pcs", "Isolation Pad Monitor 2pcs", "Cable Organizer Velcro 10pcs",
	},
	"Alat Musik Digital": {
		"Keyboard Workstation 88 Key", "Keyboard Arranger 61 Key", "Digital Piano 88 Key Hammer",
		"Synthesizer Analog 49 Key", "MIDI Controller 49 Key", "MIDI Controller 25 Key Mini",
		"Drum Pad Controller 16 Pad", "Drum Elektronik Full Kit", "MIDI Keyboard 61 Key USB",
		"Synthesizer FM 8 Operator", "Piano Digital Portable 73 Key", "Keytar 37 Key",
		"Sampler Groovebox", "Sequencer Hardware 16 Track", "MIDI Foot Controller",
	},
	"TV & Display": {
		"Smart TV LED 32 inch HD", "Smart TV LED 43 inch FHD", "Smart TV QLED 55 inch 4K",
		"Smart TV OLED 65 inch 4K", "Monitor Profesional 27 inch 4K", "Videotron LED Indoor P3",
		"Projector Full HD 4000 Lumens", "Projector 4K Laser 5000 Lumens", "Screen Projector 120 inch",
		"TV Bracket Wall Mount 55 inch", "Digital Signage 43 inch", "Interactive Whiteboard 75 inch",
		"Monitor LED 24 inch FHD", "TV Outdoor 55 inch Weatherproof", "Portable Monitor 15.6 inch",
	},
	"Sound System Portable": {
		"Speaker Bluetooth Portable 20W", "Speaker Bluetooth Waterproof 30W", "Boombox Portable 60W",
		"Party Speaker 100W RGB", "Speaker Portable Karaoke 2 Mic", "Speaker Trolley 15 inch 300W",
		"Mini Speaker BT 5W Clip", "Speaker Outdoor Portable 50W", "Speaker BT Fabric 10W",
		"Speaker Multi-Room WiFi", "Speaker Portable Solar Charge", "Party Speaker TWS 200W",
		"Speaker Alarm Clock BT 15W", "Speaker Lantern Camping BT", "Speaker Floating Pool BT",
	},
	"Perekam & Interface Audio": {
		"Audio Interface USB 2in 2out", "Audio Interface USB 4in 4out", "Audio Interface Thunderbolt 8Ch",
		"Portable Recorder Handheld", "Multi-Track Recorder 8 Channel", "Audio Interface USB-C 2Ch",
		"Podcast Interface 4 Mic Input", "Audio Interface DSP Built-in", "Portable Recorder 32-bit Float",
		"USB Audio Interface 1Ch Budget", "Audio Interface 16x16 Rack", "Streaming Audio Interface",
		"Field Recorder 6 Track", "Audio Interface MIDI Combo", "Audio Interface 2Ch with Loopback",
	},
	"DJ Equipment": {
		"DJ Controller 2-Deck USB", "DJ Controller 4-Deck Standalone", "CDJ Media Player USB",
		"DJ Mixer 4 Channel Club", "DJ Turntable Direct Drive", "DJ Headphone Swivel Cup",
		"DJ Controller Portable 2Ch", "DJ Media Player Standalone", "Mixer DJ 2 Channel Scratch",
		"DJ Effect Processor", "DJ Controller Motorized Jog", "Vinyl Turntable Belt Drive",
		"DJ Controller with Screen", "DJ Mixer Rotary 4 Channel", "DJ Controller Budget 2Ch",
	},
	"CCTV & Security": {
		"Kamera CCTV Indoor 2MP Dome", "Kamera CCTV Outdoor 4MP Bullet", "DVR 8 Channel 5MP",
		"NVR 16 Channel 4K", "Kamera IP Outdoor 8MP Varifocal", "Kamera PTZ 2MP 30x Zoom",
		"Kamera CCTV Wireless WiFi 3MP", "DVR 4 Channel Economy", "NVR 8 Channel PoE",
		"Kamera CCTV ColorVu 4MP", "HDD Surveillance 2TB Purple", "HDD Surveillance 4TB Purple",
		"Paket CCTV 4 Kamera 2MP", "Paket CCTV 8 Kamera 4MP", "Access Door Lock Fingerprint",
	},
}

var locations = []string{
	"Rak A1", "Rak A2", "Rak A3", "Rak A4", "Rak A5",
	"Rak B1", "Rak B2", "Rak B3", "Rak B4", "Rak B5",
	"Rak C1", "Rak C2", "Rak C3", "Rak C4", "Rak C5",
	"Rak D1", "Rak D2", "Rak D3", "Rak D4", "Rak D5",
	"Etalase Depan 1", "Etalase Depan 2", "Etalase Depan 3",
	"Gudang Utama", "Gudang Belakang", "Gudang Atas",
	"Display Showroom", "Area Demo", "Counter Kasir",
}

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
	db.SetMaxOpenConns(50)

	// Hapus data lama terlebih dahulu
	fmt.Println("🗑️  Menghapus data dummy lama...")
	db.Exec("DELETE FROM inventory_transactions WHERE 1=1")
	db.Exec("DELETE FROM products WHERE 1=1")
	db.Exec("ALTER TABLE products AUTO_INCREMENT = 1")

	// Sync categories
	fmt.Println("📁 Menyinkronkan kategori...")
	for cat := range brands {
		db.Exec("INSERT IGNORE INTO categories (name) VALUES (?)", cat)
	}

	totalTarget := 500000
	batchSize := 1000
	inserted := 0
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	categories := make([]string, 0, len(brands))
	for k := range brands {
		categories = append(categories, k)
	}

	fmt.Printf("🚀 Memulai penyisipan %d data produk realistis...\n", totalTarget)
	startTime := time.Now()

	for inserted < totalTarget {
		valueParts := make([]string, 0, batchSize)
		args := make([]interface{}, 0, batchSize*8)

		for j := 0; j < batchSize && inserted < totalTarget; j++ {
			cat := categories[rng.Intn(len(categories))]
			brandList := brands[cat]
			typeList := productTypes[cat]

			brand := brandList[rng.Intn(len(brandList))]
			prodType := typeList[rng.Intn(len(typeList))]

			partNum := fmt.Sprintf("DNE-%03d-%05d", rng.Intn(999)+1, inserted+1)
			desc := fmt.Sprintf("%s %s", brand, prodType)
			qty := rng.Intn(200)
			minStock := rng.Intn(15) + 3
			loc := locations[rng.Intn(len(locations))]

			// Harga realistis per kategori (Rupiah)
			var capitalPrice, sellingPrice float64
			switch cat {
			case "TV & Display":
				capitalPrice = float64(rng.Intn(35000000) + 1500000)
			case "Speaker Aktif", "Alat Musik Digital":
				capitalPrice = float64(rng.Intn(25000000) + 500000)
			case "Amplifier", "DJ Equipment":
				capitalPrice = float64(rng.Intn(20000000) + 750000)
			case "Speaker Pasif":
				capitalPrice = float64(rng.Intn(15000000) + 300000)
			case "Mixer Audio", "Perekam & Interface Audio":
				capitalPrice = float64(rng.Intn(18000000) + 400000)
			case "Microphone":
				capitalPrice = float64(rng.Intn(12000000) + 150000)
			case "Headphone & Earphone":
				capitalPrice = float64(rng.Intn(8000000) + 75000)
			case "Lighting & Stage":
				capitalPrice = float64(rng.Intn(10000000) + 250000)
			case "CCTV & Security":
				capitalPrice = float64(rng.Intn(8000000) + 200000)
			case "Sound System Portable":
				capitalPrice = float64(rng.Intn(5000000) + 100000)
			case "Kabel & Konektor":
				capitalPrice = float64(rng.Intn(500000) + 15000)
			default:
				capitalPrice = float64(rng.Intn(2000000) + 50000)
			}

			// Margin 15-40%
			margin := 1.15 + rng.Float64()*0.25
			sellingPrice = float64(int(capitalPrice*margin/1000) * 1000) // Bulatkan ke ribuan

			valueParts = append(valueParts, "(?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args, partNum, desc, cat, qty, minStock, capitalPrice, sellingPrice, loc)
			inserted++
		}

		query := "INSERT INTO products (part_number, description, category, quantity, min_stock_level, capital_price, selling_price, location) VALUES " + strings.Join(valueParts, ",")
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("Error batch insert at row %d: %v", inserted, err)
		}

		if inserted%50000 == 0 {
			elapsed := time.Since(startTime)
			speed := float64(inserted) / elapsed.Seconds()
			fmt.Printf("   ✅ %d / %d produk dimasukkan (%.0f produk/detik)\n", inserted, totalTarget, speed)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Println("\n============================================")
	fmt.Println("   SEEDING SELESAI!")
	fmt.Println("============================================")
	fmt.Printf("   Total Produk  : %d\n", inserted)
	fmt.Printf("   Durasi        : %v\n", elapsed)
	fmt.Printf("   Kecepatan     : %.0f produk/detik\n", float64(inserted)/elapsed.Seconds())
	fmt.Println("============================================")
	fmt.Println("🎉 Database siap untuk presentasi!")
}
