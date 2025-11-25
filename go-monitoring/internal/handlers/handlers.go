package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/monitoring/charging-stations/internal/database"
	"github.com/monitoring/charging-stations/internal/models"
	"github.com/monitoring/charging-stations/internal/utils"
)

// Handler représente le gestionnaire principal
type Handler struct {
	db        *database.DB
	templates *template.Template
}

// New crée un nouveau handler
func New(db *database.DB) *Handler {
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"formatDate": func(t time.Time) string {
			return t.Format("02/01/2006 15:04")
		},
		"formatDateShort": func(t time.Time) string {
			return t.Format("02/01/2006")
		},
		"json": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return string(b)
		},
		"mult": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		// AJOUTE CETTE FONCTION
		"float64": func(v interface{}) float64 {
			switch val := v.(type) {
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case float64:
				return val
			case float32:
				return float64(val)
			default:
				return 0
			}
		},
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	return &Handler{
		db:        db,
		templates: tmpl,
	}
}

// RegisterRoutes enregistre toutes les routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Page principale
	r.HandleFunc("/", h.Index).Methods("GET")

	// API pour les filtres
	r.HandleFunc("/api/filters", h.GetFilters).Methods("POST")
	r.HandleFunc("/api/kpis", h.GetKPIs).Methods("POST")

	// Tabs
	r.HandleFunc("/tabs/overview", h.TabOverview).Methods("POST")
	r.HandleFunc("/tabs/general", h.TabGeneral).Methods("POST")
	r.HandleFunc("/tabs/comparison", h.TabComparison).Methods("POST")
	r.HandleFunc("/tabs/pdc-details", h.TabPDCDetails).Methods("POST")
	r.HandleFunc("/tabs/stats", h.TabStats).Methods("POST")
	r.HandleFunc("/tabs/projection", h.TabProjection).Methods("POST")
	r.HandleFunc("/tabs/attempts", h.TabAttempts).Methods("POST")
	r.HandleFunc("/tabs/suspicious", h.TabSuspicious).Methods("POST")
	r.HandleFunc("/tabs/error-moment", h.TabErrorMoment).Methods("POST")
	r.HandleFunc("/tabs/error-specific", h.TabErrorSpecific).Methods("POST")
	r.HandleFunc("/tabs/alerts", h.TabAlerts).Methods("POST")
	r.HandleFunc("/tabs/evolution", h.TabEvolution).Methods("POST")
	r.HandleFunc("/tabs/defects", h.TabDefects).Methods("POST")

	// Refresh cache
	r.HandleFunc("/api/refresh-cache", h.RefreshCache).Methods("POST")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

// Index affiche la page principale
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	sessions := h.db.GetSessions()
	sites := utils.GetUniqueSites(sessions)

	data := struct {
		Sites []string
		Year  int
		Month int
	}{
		Sites: sites,
		Year:  time.Now().Year(),
		Month: int(time.Now().Month()),
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// GetFilters récupère et filtre les sessions
func (h *Handler) GetFilters(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := h.db.GetSessions()
	filtered := utils.FilterSessions(sessions, filters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": len(filtered),
	})
}

// GetKPIs calcule et retourne les KPIs
func (h *Handler) GetKPIs(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := h.db.GetSessions()
	filtered := utils.FilterSessions(sessions, filters)

	kpis := utils.CalculateKPIs(filtered, filters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kpis)
}

// TabOverview retourne l'onglet vue d'ensemble
func (h *Handler) TabOverview(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)

	// Récupérer les données
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)
	defauts := utils.GetActiveDefauts(h.db.GetDefauts(), filters)
	suspicious := h.db.GetSuspicious()
	multiAttempts := h.db.GetMultiAttempts()
	alertes := h.db.GetAlertes()

	// Filtrer suspicious et multi attempts
	var suspFiltered []models.SuspiciousTransaction
	for _, s := range suspicious {
		if len(filters.Sites) > 0 {
			found := false
			for _, site := range filters.Sites {
				if s.Site == site {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if s.DatetimeStart.After(filters.DateStart) && s.DatetimeStart.Before(filters.DateEnd) {
			suspFiltered = append(suspFiltered, s)
		}
	}

	kpis := utils.CalculateKPIs(sessions, filters)
	siteStats := utils.GetTop10Sites(sessions)

	data := struct {
		KPIs          models.KPISummary
		Defauts       []models.Defaut
		Suspicious    []models.SuspiciousTransaction
		MultiAttempts []models.MultiAttempt
		Alertes       []models.Alerte
		TopSites      []models.SiteStats
	}{
		KPIs:          kpis,
		Defauts:       defauts,
		Suspicious:    suspFiltered,
		MultiAttempts: multiAttempts,
		Alertes:       alertes,
		TopSites:      siteStats,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_overview.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabGeneral retourne l'onglet général
func (h *Handler) TabGeneral(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	kpis := utils.CalculateKPIs(sessions, filters)
	siteStats := utils.GetStatsBySite(sessions)
	momentCounts := utils.GetMomentCounts(sessions)

	data := struct {
		KPIs         models.KPISummary
		SiteStats    []models.SiteStats
		MomentCounts []models.MomentCount
	}{
		KPIs:         kpis,
		SiteStats:    siteStats,
		MomentCounts: momentCounts,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_general.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabComparison retourne l'onglet comparaison par site
func (h *Handler) TabComparison(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	siteStats := utils.GetStatsBySite(sessions)

	data := struct {
		SiteStats []models.SiteStats
	}{
		SiteStats: siteStats,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_comparison.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabPDCDetails retourne l'onglet détails PDC
func (h *Handler) TabPDCDetails(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	site := r.FormValue("site")

	sessions := utils.FilterSessions(h.db.GetSessions(), filters)
	pdcStats := utils.GetStatsByPDC(sessions, site)
	momentCounts := utils.GetMomentCounts(sessions)

	data := struct {
		Site         string
		PDCStats     []models.PDCStats
		MomentCounts []models.MomentCount
	}{
		Site:         site,
		PDCStats:     pdcStats,
		MomentCounts: momentCounts,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_pdc_details.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabStats retourne l'onglet statistiques
func (h *Handler) TabStats(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	// Calculs statistiques
	kpis := utils.CalculateKPIs(sessions, filters)

	data := struct {
		KPIs models.KPISummary
	}{
		KPIs: kpis,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_stats.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabProjection retourne l'onglet projection pivot
func (h *Handler) TabProjection(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	// Logique de pivot sera implémentée ici
	data := struct {
		Sessions []models.Session
	}{
		Sessions: sessions,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_projection.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabAttempts retourne l'onglet tentatives multiples
func (h *Handler) TabAttempts(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	multiAttempts := h.db.GetMultiAttempts()

	// Filtrer par site et date
	var filtered []models.MultiAttempt
	for _, m := range multiAttempts {
		if len(filters.Sites) > 0 {
			found := false
			for _, site := range filters.Sites {
				if m.Site == site {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, m)
	}

	data := struct {
		MultiAttempts []models.MultiAttempt
	}{
		MultiAttempts: filtered,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_attempts.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabSuspicious retourne l'onglet transactions suspectes
func (h *Handler) TabSuspicious(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	suspicious := h.db.GetSuspicious()

	// Filtrer
	var filtered []models.SuspiciousTransaction
	for _, s := range suspicious {
		if len(filters.Sites) > 0 {
			found := false
			for _, site := range filters.Sites {
				if s.Site == site {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if s.DatetimeStart.After(filters.DateStart) && s.DatetimeStart.Before(filters.DateEnd) {
			filtered = append(filtered, s)
		}
	}

	data := struct {
		Suspicious []models.SuspiciousTransaction
	}{
		Suspicious: filtered,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_suspicious.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabErrorMoment retourne l'onglet erreur moment
func (h *Handler) TabErrorMoment(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	momentCounts := utils.GetMomentCounts(sessions)
	eviOccurrences := utils.GetCodeOccurrences(sessions, true)
	dsOccurrences := utils.GetCodeOccurrences(sessions, false)

	data := struct {
		MomentCounts   []models.MomentCount
		EVIOccurrences map[int]*models.CodeOccurrence
		DSOccurrences  map[int]*models.CodeOccurrence
	}{
		MomentCounts:   momentCounts,
		EVIOccurrences: eviOccurrences,
		DSOccurrences:  dsOccurrences,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_error_moment.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabErrorSpecific retourne l'onglet erreur spécifique
func (h *Handler) TabErrorSpecific(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	sessions := utils.FilterSessions(h.db.GetSessions(), filters)

	// Filtres spécifiques pour MAC et codes
	macFilter := r.FormValue("mac")
	codeFilter := r.FormValue("code")

	var filtered []models.Session
	for _, s := range sessions {
		// Filtre MAC
		if macFilter != "" && !strings.Contains(strings.ToLower(s.MACAddress), strings.ToLower(macFilter)) {
			continue
		}

		// Filtre code
		if codeFilter != "" {
			code, err := strconv.Atoi(codeFilter)
			if err == nil {
				if s.EVIErrorCode == nil || *s.EVIErrorCode != code {
					if s.DownstreamCodePC == nil || *s.DownstreamCodePC != code {
						continue
					}
				}
			}
		}

		filtered = append(filtered, s)
	}

	data := struct {
		Sessions []models.Session
	}{
		Sessions: filtered,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_error_specific.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabAlerts retourne l'onglet alertes
func (h *Handler) TabAlerts(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	alertes := h.db.GetAlertes()

	// Filtrer
	var filtered []models.Alerte
	for _, a := range alertes {
		if len(filters.Sites) > 0 {
			found := false
			for _, site := range filters.Sites {
				if a.Site == site {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if a.Detection.After(filters.DateStart) && a.Detection.Before(filters.DateEnd) {
			filtered = append(filtered, a)
		}
	}

	data := struct {
		Alertes []models.Alerte
	}{
		Alertes: filtered,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_alerts.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabEvolution retourne l'onglet évolution
func (h *Handler) TabEvolution(w http.ResponseWriter, r *http.Request) {
	stats := h.db.GetStatsGlobal()

	data := struct {
		Stats []models.StatsGlobal
	}{
		Stats: stats,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_evolution.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// TabDefects retourne l'onglet historique défauts
func (h *Handler) TabDefects(w http.ResponseWriter, r *http.Request) {
	filters := h.parseFilters(r)
	defauts := h.db.GetDefauts()

	// Filtrer
	var filtered []models.Defaut
	for _, d := range defauts {
		if len(filters.Sites) > 0 {
			found := false
			for _, site := range filters.Sites {
				if d.Site == site {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if d.DateDebut.After(filters.DateStart) && d.DateDebut.Before(filters.DateEnd) {
			filtered = append(filtered, d)
		}
	}

	data := struct {
		Defauts []models.Defaut
	}{
		Defauts: filtered,
	}

	if err := h.templates.ExecuteTemplate(w, "tab_defects.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// RefreshCache force le refresh du cache
func (h *Handler) RefreshCache(w http.ResponseWriter, r *http.Request) {
	if err := h.db.RefreshCache(); err != nil {
		http.Error(w, fmt.Sprintf("Error refreshing cache: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Cache refreshed successfully",
	})
}

// parseFilters parse les filtres depuis la requête
func (h *Handler) parseFilters(r *http.Request) models.Filters {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	filters := models.Filters{}

	// Sites
	if sites := r.Form["sites[]"]; len(sites) > 0 {
		filters.Sites = sites
	}

	// Date mode
	filters.DateMode = r.FormValue("date_mode")
	if filters.DateMode == "" {
		filters.DateMode = "mois_complet"
	}

	// Year, month, day
	year, _ := strconv.Atoi(r.FormValue("focus_year"))
	if year == 0 {
		year = time.Now().Year()
	}
	filters.FocusYear = year

	month, _ := strconv.Atoi(r.FormValue("focus_month"))
	if month == 0 {
		month = int(time.Now().Month())
	}
	filters.FocusMonth = month

	// Focus day
	if dayStr := r.FormValue("focus_day"); dayStr != "" {
		if day, err := time.Parse("2006-01-02", dayStr); err == nil {
			filters.FocusDay = day
		}
	}

	// Calculer dates
	filters.DateStart, filters.DateEnd = utils.ParseDateRange(
		filters.DateMode,
		filters.FocusYear,
		filters.FocusMonth,
		filters.FocusDay,
	)

	// Types erreur
	if types := r.Form["types_erreur[]"]; len(types) > 0 {
		filters.TypesErreur = types
	}

	// Moments
	if moments := r.Form["moments[]"]; len(moments) > 0 {
		filters.Moments = moments
	}

	return filters
}
