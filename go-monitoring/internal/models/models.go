package models

import "time"

// Session représente une session de charge
type Session struct {
	ID                  string    `db:"ID"`
	DatetimeStart       time.Time `db:"Datetime start"`
	DatetimeEnd         *time.Time `db:"Datetime end"`
	Site                string    `db:"Site"`
	PDC                 string    `db:"PDC"`
	StateOfCharge       int       `db:"State of charge(0:good, 1:error)"`
	TypeErreur          string    `db:"type_erreur"`
	Moment              string    `db:"moment"`
	MomentAvancee       string    `db:"moment_avancee"`
	EVIErrorCode        *int      `db:"EVI Error Code"`
	EVIMomentStep       *int      `db:"EVI Status during error"`
	DownstreamCodePC    *int      `db:"Downstream Code PC"`
	EnergyKwh           *float64  `db:"Energy (Kwh)"`
	MeanPowerKw         *float64  `db:"Mean Power (Kw)"`
	MaxPowerKw          *float64  `db:"Max Power (Kw)"`
	SOCStart            *float64  `db:"SOC Start"`
	SOCEnd              *float64  `db:"SOC End"`
	MACAddress          string    `db:"MAC Address"`
	Charge900V          int       `db:"charge_900V"`
}

// Alerte représente une alerte de défaut récurrent
type Alerte struct {
	Site              string    `db:"Site"`
	PDC               string    `db:"PDC"`
	TypeErreur        string    `db:"type_erreur"`
	Detection         time.Time `db:"detection"`
	Occurrences12h    int       `db:"occurrences_12h"`
	Moment            string    `db:"moment"`
	EVICode           *int      `db:"evi_code"`
	DownstreamCodePC  *int      `db:"downstream_code_pc"`
}

// Defaut représente un défaut actif ou historique
type Defaut struct {
	Site       string     `db:"site"`
	DateDebut  time.Time  `db:"date_debut"`
	DateFin    *time.Time `db:"date_fin"`
	Defaut     string     `db:"defaut"`
	Equipement string     `db:"eqp"`
}

// SuspiciousTransaction représente une transaction suspecte (<1 kWh)
type SuspiciousTransaction struct {
	ID            string    `db:"ID"`
	Site          string    `db:"Site"`
	PDC           string    `db:"PDC"`
	MACAddress    string    `db:"MAC Address"`
	Vehicle       string    `db:"Vehicle"`
	DatetimeStart time.Time `db:"Datetime start"`
	DatetimeEnd   *time.Time `db:"Datetime end"`
	EnergyKwh     float64   `db:"Energy (Kwh)"`
	SOCStart      *float64  `db:"SOC Start"`
	SOCEnd        *float64  `db:"SOC End"`
}

// MultiAttempt représente un utilisateur avec multiples tentatives
type MultiAttempt struct {
	Site              string    `db:"Site"`
	Heure             string    `db:"Heure"`
	MAC               string    `db:"MAC"`
	Vehicle           string    `db:"Vehicle"`
	Tentatives        int       `db:"tentatives"`
	PDCs              string    `db:"PDC(s)"`
	PremiereTentative time.Time `db:"1ère tentative"`
	DerniereTentative time.Time `db:"Dernière tentative"`
	IDs               string    `db:"ID(s)"`
	SOCStartMin       *float64  `db:"SOC start min"`
	SOCStartMax       *float64  `db:"SOC start max"`
	SOCEndMin         *float64  `db:"SOC end min"`
	SOCEndMax         *float64  `db:"SOC end max"`
}

// ChargeMAC représente une charge avec informations MAC/véhicule
// Note: kpi_charges_mac ne contient pas PDC, Datetime end, ou Energy (Kwh)
type ChargeMAC struct {
	ID            string    `db:"ID"`
	Site          string    `db:"Site"`
	MACAddress    string    `db:"MAC Address"`
	Vehicle       string    `db:"Vehicle"`
	DatetimeStart time.Time `db:"Datetime start"`
	SOCStart      *float64  `db:"SOC Start"`
	SOCEnd        *float64  `db:"SOC End"`
	IsOK          bool      `db:"is_ok"`
}

// StatsGlobal représente des statistiques globales
type StatsGlobal struct {
	Mois         string  `db:"mois"`
	TauxReussite float64 `db:"tr"`
}

// ChargesDaily représente le nombre de charges par jour
type ChargesDaily struct {
	Site   string    `db:"Site"`
	Day    time.Time `db:"day"`
	Status string    `db:"Status"`
	Nb     int       `db:"Nb"`
}

// DurationsSiteDaily représente les durées par site et jour
type DurationsSiteDaily struct {
	Site   string    `db:"Site"`
	Day    time.Time `db:"day"`
	DurMin float64   `db:"dur_min"`
}

// DurationsPDCDaily représente les durées par PDC et jour
type DurationsPDCDaily struct {
	Site   string    `db:"Site"`
	PDC    string    `db:"PDC"`
	Day    time.Time `db:"day"`
	DurMin float64   `db:"dur_min"`
}

// Filters représente les filtres utilisateur
type Filters struct {
	Sites        []string  `json:"sites"`
	DateMode     string    `json:"date_mode"`
	DateStart    time.Time `json:"date_start"`
	DateEnd      time.Time `json:"date_end"`
	TypesErreur  []string  `json:"types_erreur"`
	Moments      []string  `json:"moments"`
	FocusYear    int       `json:"focus_year"`
	FocusMonth   int       `json:"focus_month"`
	FocusDay     time.Time `json:"focus_day"`
}

// KPISummary représente les KPIs globaux
type KPISummary struct {
	Total         int     `json:"total"`
	OK            int     `json:"ok"`
	NOK           int     `json:"nok"`
	TauxReussite  float64 `json:"taux_reussite"`
	TauxEchec     float64 `json:"taux_echec"`
	NbSites       int     `json:"nb_sites"`
	NbPDC         int     `json:"nb_pdc"`
}

// SiteStats représente les stats par site
type SiteStats struct {
	Site         string  `json:"site"`
	Total        int     `json:"total"`
	OK           int     `json:"ok"`
	NOK          int     `json:"nok"`
	TauxReussite float64 `json:"taux_reussite"`
	TauxEchec    float64 `json:"taux_echec"`
}

// PDCStats représente les stats par PDC
type PDCStats struct {
	PDC          string  `json:"pdc"`
	Total        int     `json:"total"`
	OK           int     `json:"ok"`
	NOK          int     `json:"nok"`
	TauxReussite float64 `json:"taux_reussite"`
}

// MomentCount représente le comptage par moment
type MomentCount struct {
	Moment string `json:"moment"`
	Count  int    `json:"count"`
}

// CodeOccurrence représente les occurrences d'un code d'erreur
type CodeOccurrence struct {
	Code        int               `json:"code"`
	Total       int               `json:"total"`
	Percentage  float64           `json:"percentage"`
	ByMoment    map[string]int    `json:"by_moment"`
}
