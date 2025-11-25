package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/monitoring/charging-stations/internal/models"
)

// DB représente la connexion à la base de données
type DB struct {
	conn  *sql.DB
	cache *Cache
	mu    sync.RWMutex
}

// Cache pour les données KPI
type Cache struct {
	sessions              []models.Session
	alertes               []models.Alerte
	defauts               []models.Defaut
	suspicious            []models.SuspiciousTransaction
	multiAttempts         []models.MultiAttempt
	chargesMAC            []models.ChargeMAC
	statsGlobal           []models.StatsGlobal
	chargesDaily          []models.ChargesDaily
	durationsSiteDaily    []models.DurationsSiteDaily
	durationsPDCDaily     []models.DurationsPDCDaily
	lastUpdate            time.Time
	mu                    sync.RWMutex
}

var (
	instance *DB
	once     sync.Once
)

// GetDB retourne l'instance singleton de la base de données
func GetDB() *DB {
	once.Do(func() {
		instance = &DB{
			cache: &Cache{},
		}
		if err := instance.Connect(); err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
	})
	return instance
}

// Connect établit la connexion à MySQL
func (db *DB) Connect() error {
	dsn := "nidec:MaV38f5xsGQp83@tcp(162.19.251.55:3306)/Charges?parseTime=true"

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Configuration de la connexion
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test de connexion
	if err := conn.Ping(); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	db.conn = conn
	log.Println("✅ Connected to MySQL database")

	// Charger le cache initial
	go db.RefreshCache()

	return nil
}

// RefreshCache recharge toutes les données KPI
func (db *DB) RefreshCache() error {
	db.cache.mu.Lock()
	defer db.cache.mu.Unlock()

	log.Println("🔄 Refreshing cache...")

	// Charger les sessions
	sessions, err := db.loadSessions()
	if err != nil {
		log.Printf("Error loading sessions: %v", err)
	} else {
		db.cache.sessions = sessions
	}

	// Charger les alertes
	alertes, err := db.loadAlertes()
	if err != nil {
		log.Printf("Error loading alertes: %v", err)
	} else {
		db.cache.alertes = alertes
	}

	// Charger les défauts
	defauts, err := db.loadDefauts()
	if err != nil {
		log.Printf("Error loading defauts: %v", err)
	} else {
		db.cache.defauts = defauts
	}

	// Charger les transactions suspectes
	suspicious, err := db.loadSuspicious()
	if err != nil {
		log.Printf("Error loading suspicious: %v", err)
	} else {
		db.cache.suspicious = suspicious
	}

	// Charger les tentatives multiples
	multiAttempts, err := db.loadMultiAttempts()
	if err != nil {
		log.Printf("Error loading multi attempts: %v", err)
	} else {
		db.cache.multiAttempts = multiAttempts
	}

	// Charger charges_mac
	chargesMAC, err := db.loadChargesMAC()
	if err != nil {
		log.Printf("Error loading charges_mac: %v", err)
	} else {
		db.cache.chargesMAC = chargesMAC
	}

	// Charger stats globales
	statsGlobal, err := db.loadStatsGlobal()
	if err != nil {
		log.Printf("Error loading stats global: %v", err)
	} else {
		db.cache.statsGlobal = statsGlobal
	}

	// Charger charges daily
	chargesDaily, err := db.loadChargesDaily()
	if err != nil {
		log.Printf("Error loading charges daily: %v", err)
	} else {
		db.cache.chargesDaily = chargesDaily
	}

	// Charger durées site daily
	durationsSiteDaily, err := db.loadDurationsSiteDaily()
	if err != nil {
		log.Printf("Error loading durations site daily: %v", err)
	} else {
		db.cache.durationsSiteDaily = durationsSiteDaily
	}

	// Charger durées PDC daily
	durationsPDCDaily, err := db.loadDurationsPDCDaily()
	if err != nil {
		log.Printf("Error loading durations pdc daily: %v", err)
	} else {
		db.cache.durationsPDCDaily = durationsPDCDaily
	}

	db.cache.lastUpdate = time.Now()
	log.Println("✅ Cache refreshed successfully")

	return nil
}

// loadSessions charge les sessions depuis la table kpi_sessions
func (db *DB) loadSessions() ([]models.Session, error) {
	query := "SELECT ID, `Datetime start`, `Datetime end`, COALESCE(Site, `Name Project`) as Site, " +
		"PDC, `State of charge(0:good, 1:error)`, type_erreur, moment, moment_avancee, " +
		"`EVI Error Code`, `EVI Status during error`, `Downstream Code PC`, `Energy (Kwh)`, " +
		"`Mean Power (Kw)`, `Max Power (Kw)`, `SOC Start`, `SOC End`, `MAC Address`, charge_900V " +
		"FROM kpi_sessions"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		err := rows.Scan(
			&s.ID,
			&s.DatetimeStart,
			&s.DatetimeEnd,
			&s.Site,
			&s.PDC,
			&s.StateOfCharge,
			&s.TypeErreur,
			&s.Moment,
			&s.MomentAvancee,
			&s.EVIErrorCode,
			&s.EVIMomentStep,
			&s.DownstreamCodePC,
			&s.EnergyKwh,
			&s.MeanPowerKw,
			&s.MaxPowerKw,
			&s.SOCStart,
			&s.SOCEnd,
			&s.MACAddress,
			&s.Charge900V,
		)
		if err != nil {
			log.Printf("Error scanning session: %v", err)
			continue
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

// loadAlertes charge les alertes
func (db *DB) loadAlertes() ([]models.Alerte, error) {
	query := `SELECT Site, PDC, type_erreur, detection, occurrences_12h, moment, evi_code, downstream_code_pc
		FROM kpi_alertes ORDER BY detection DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alertes []models.Alerte
	for rows.Next() {
		var a models.Alerte
		err := rows.Scan(&a.Site, &a.PDC, &a.TypeErreur, &a.Detection, &a.Occurrences12h,
			&a.Moment, &a.EVICode, &a.DownstreamCodePC)
		if err != nil {
			continue
		}
		alertes = append(alertes, a)
	}

	return alertes, nil
}

// loadDefauts charge les défauts
func (db *DB) loadDefauts() ([]models.Defaut, error) {
	query := `SELECT site, date_debut, date_fin, defaut, eqp
		FROM kpi_defauts_log ORDER BY date_debut DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defauts []models.Defaut
	for rows.Next() {
		var d models.Defaut
		err := rows.Scan(&d.Site, &d.DateDebut, &d.DateFin, &d.Defaut, &d.Equipement)
		if err != nil {
			continue
		}
		defauts = append(defauts, d)
	}

	return defauts, nil
}

// loadSuspicious charge les transactions suspectes
func (db *DB) loadSuspicious() ([]models.SuspiciousTransaction, error) {
	query := "SELECT ID, Site, PDC, `MAC Address`, Vehicle, `Datetime start`, `Datetime end`, " +
		"`Energy (Kwh)`, `SOC Start`, `SOC End` FROM kpi_suspicious_under_1kwh"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suspicious []models.SuspiciousTransaction
	for rows.Next() {
		var s models.SuspiciousTransaction
		err := rows.Scan(&s.ID, &s.Site, &s.PDC, &s.MACAddress, &s.Vehicle,
			&s.DatetimeStart, &s.DatetimeEnd, &s.EnergyKwh, &s.SOCStart, &s.SOCEnd)
		if err != nil {
			continue
		}
		suspicious = append(suspicious, s)
	}

	return suspicious, nil
}

// loadMultiAttempts charge les tentatives multiples
func (db *DB) loadMultiAttempts() ([]models.MultiAttempt, error) {
	query := "SELECT Site, Heure, MAC, Vehicle, tentatives, `PDC(s)`, " +
		"`1ère tentative`, `Dernière tentative`, `ID(s)`, " +
		"`SOC start min`, `SOC start max`, `SOC end min`, `SOC end max` " +
		"FROM kpi_multi_attempts_hour"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []models.MultiAttempt
	for rows.Next() {
		var m models.MultiAttempt
		err := rows.Scan(&m.Site, &m.Heure, &m.MAC, &m.Vehicle, &m.Tentatives, &m.PDCs,
			&m.PremiereTentative, &m.DerniereTentative, &m.IDs,
			&m.SOCStartMin, &m.SOCStartMax, &m.SOCEndMin, &m.SOCEndMax)
		if err != nil {
			continue
		}
		attempts = append(attempts, m)
	}

	return attempts, nil
}

// loadChargesMAC charge les charges avec MAC/véhicule
func (db *DB) loadChargesMAC() ([]models.ChargeMAC, error) {
	// Note: kpi_charges_mac contient seulement: ID, Site, MAC Address, Vehicle, Datetime start, is_ok, SOC Start, SOC End
	query := "SELECT ID, Site, `MAC Address`, Vehicle, `Datetime start`, " +
		"`SOC Start`, `SOC End`, is_ok " +
		"FROM kpi_charges_mac"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charges []models.ChargeMAC
	for rows.Next() {
		var c models.ChargeMAC
		var isOKInt int
		err := rows.Scan(&c.ID, &c.Site, &c.MACAddress, &c.Vehicle,
			&c.DatetimeStart, &c.SOCStart, &c.SOCEnd, &isOKInt)
		if err != nil {
			continue
		}
		c.IsOK = isOKInt == 1
		charges = append(charges, c)
	}

	return charges, nil
}

// loadStatsGlobal charge les stats globales d'évolution
func (db *DB) loadStatsGlobal() ([]models.StatsGlobal, error) {
	query := `SELECT mois, tr FROM kpi_evo ORDER BY mois`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.StatsGlobal
	for rows.Next() {
		var s models.StatsGlobal
		err := rows.Scan(&s.Mois, &s.TauxReussite)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// loadChargesDaily charge les charges quotidiennes
func (db *DB) loadChargesDaily() ([]models.ChargesDaily, error) {
	query := `SELECT Site, day, Status, Nb FROM kpi_charges_daily_by_site`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charges []models.ChargesDaily
	for rows.Next() {
		var c models.ChargesDaily
		err := rows.Scan(&c.Site, &c.Day, &c.Status, &c.Nb)
		if err != nil {
			continue
		}
		charges = append(charges, c)
	}

	return charges, nil
}

// loadDurationsSiteDaily charge les durées par site
func (db *DB) loadDurationsSiteDaily() ([]models.DurationsSiteDaily, error) {
	query := `SELECT Site, day, dur_min FROM kpi_durations_site_daily`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var durations []models.DurationsSiteDaily
	for rows.Next() {
		var d models.DurationsSiteDaily
		err := rows.Scan(&d.Site, &d.Day, &d.DurMin)
		if err != nil {
			continue
		}
		durations = append(durations, d)
	}

	return durations, nil
}

// loadDurationsPDCDaily charge les durées par PDC
func (db *DB) loadDurationsPDCDaily() ([]models.DurationsPDCDaily, error) {
	query := `SELECT Site, PDC, day, dur_min FROM kpi_durations_pdc_daily`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var durations []models.DurationsPDCDaily
	for rows.Next() {
		var d models.DurationsPDCDaily
		err := rows.Scan(&d.Site, &d.PDC, &d.Day, &d.DurMin)
		if err != nil {
			continue
		}
		durations = append(durations, d)
	}

	return durations, nil
}

// GetSessions retourne les sessions du cache
func (db *DB) GetSessions() []models.Session {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.sessions
}

// GetAlertes retourne les alertes du cache
func (db *DB) GetAlertes() []models.Alerte {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.alertes
}

// GetDefauts retourne les défauts du cache
func (db *DB) GetDefauts() []models.Defaut {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.defauts
}

// GetSuspicious retourne les transactions suspectes du cache
func (db *DB) GetSuspicious() []models.SuspiciousTransaction {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.suspicious
}

// GetMultiAttempts retourne les tentatives multiples du cache
func (db *DB) GetMultiAttempts() []models.MultiAttempt {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.multiAttempts
}

// GetChargesMAC retourne les charges avec MAC/véhicule du cache
func (db *DB) GetChargesMAC() []models.ChargeMAC {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.chargesMAC
}

// GetStatsGlobal retourne les stats globales du cache
func (db *DB) GetStatsGlobal() []models.StatsGlobal {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.statsGlobal
}

// GetChargesDaily retourne les charges quotidiennes du cache
func (db *DB) GetChargesDaily() []models.ChargesDaily {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.chargesDaily
}

// GetDurationsSiteDaily retourne les durées par site du cache
func (db *DB) GetDurationsSiteDaily() []models.DurationsSiteDaily {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.durationsSiteDaily
}

// GetDurationsPDCDaily retourne les durées par PDC du cache
func (db *DB) GetDurationsPDCDaily() []models.DurationsPDCDaily {
	db.cache.mu.RLock()
	defer db.cache.mu.RUnlock()
	return db.cache.durationsPDCDaily
}

// Close ferme la connexion à la base de données
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}
