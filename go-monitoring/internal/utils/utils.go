package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/monitoring/charging-stations/internal/models"
)

// Constantes pour les moments et palettes
var (
	MomentOrder = []string{"Init", "Lock Connector", "CableCheck", "Charge", "Fin de charge", "Unknown"}

	MomentPalette = map[string]string{
		"Init":           "#636EFA",
		"Lock Connector": "#EF553B",
		"CableCheck":     "#00CC96",
		"Charge":         "#AB63FA",
		"Fin de charge":  "#38AC21",
		"Unknown":        "#19D3F3",
	}

	SitePalette = map[string]string{
		"Saint-Jean-de-Maurienne": "#636EFA",
		"La Rochelle":             "#EF553B",
		"Pouilly-en-Auxois":       "#00CC96",
		"Carvin":                  "#AB63FA",
		"Pau - Novotel":           "#38AC21",
		"Unknown":                 "#19D3F3",
	}

	BaseChargeURL = "https://elto.nidec-asi-online.com/Charge/detail?id="
)

// FilterSessions filtre les sessions selon les critères
func FilterSessions(sessions []models.Session, filters models.Filters) []models.Session {
	var filtered []models.Session

	for _, s := range sessions {
		// Filtre site
		if len(filters.Sites) > 0 && !contains(filters.Sites, s.Site) {
			continue
		}

		// Filtre date
		if !s.DatetimeStart.IsZero() {
			if s.DatetimeStart.Before(filters.DateStart) || s.DatetimeStart.After(filters.DateEnd) {
				continue
			}
		}

		// Filtre type erreur
		// Si un filtre de type d'erreur est appliqué, on filtre uniquement les erreurs
		// Les sessions OK sont toujours conservées (comme dans le code Python)
		if len(filters.TypesErreur) > 0 {
			// Si la session est en erreur ET ne correspond pas aux types sélectionnés, on l'exclut
			// Les sessions OK (StateOfCharge == 0) passent toujours ce filtre
			if s.StateOfCharge != 0 && !contains(filters.TypesErreur, s.TypeErreur) {
				continue
			}
		}

		// Filtre moment
		// Si un filtre de moment est appliqué, on filtre uniquement les erreurs
		// Les sessions OK sont toujours conservées (comme dans le code Python)
		if len(filters.Moments) > 0 {
			// Si la session est en erreur ET ne correspond pas aux moments sélectionnés, on l'exclut
			// Les sessions OK (StateOfCharge == 0) passent toujours ce filtre
			if s.StateOfCharge != 0 && !contains(filters.Moments, s.Moment) {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	return filtered
}

// CalculateKPIs calcule les KPIs depuis les sessions filtrées
func CalculateKPIs(sessions []models.Session, filters models.Filters) models.KPISummary {
	total := len(sessions)
	ok := 0
	nok := 0

	sitesMap := make(map[string]bool)
	pdcMap := make(map[string]bool)

	for _, s := range sessions {
		sitesMap[s.Site] = true
		pdcMap[s.PDC] = true

		if s.StateOfCharge == 0 {
			ok++
		} else {
			nok++
		}
	}

	tauxReussite := 0.0
	tauxEchec := 0.0
	if total > 0 {
		tauxReussite = float64(ok) / float64(total) * 100
		tauxEchec = float64(nok) / float64(total) * 100
	}

	return models.KPISummary{
		Total:        total,
		OK:           ok,
		NOK:          nok,
		TauxReussite: round(tauxReussite, 2),
		TauxEchec:    round(tauxEchec, 2),
		NbSites:      len(sitesMap),
		NbPDC:        len(pdcMap),
	}
}

// GetStatsBySite calcule les stats par site
func GetStatsBySite(sessions []models.Session) []models.SiteStats {
	siteMap := make(map[string]*models.SiteStats)

	for _, s := range sessions {
		if _, exists := siteMap[s.Site]; !exists {
			siteMap[s.Site] = &models.SiteStats{
				Site: s.Site,
			}
		}

		stats := siteMap[s.Site]
		stats.Total++
		if s.StateOfCharge == 0 {
			stats.OK++
		} else {
			stats.NOK++
		}
	}

	var result []models.SiteStats
	for _, stats := range siteMap {
		if stats.Total > 0 {
			stats.TauxReussite = round(float64(stats.OK)/float64(stats.Total)*100, 2)
			stats.TauxEchec = round(float64(stats.NOK)/float64(stats.Total)*100, 2)
		}
		result = append(result, *stats)
	}

	return result
}

// GetStatsByPDC calcule les stats par PDC pour un site
func GetStatsByPDC(sessions []models.Session, site string) []models.PDCStats {
	pdcMap := make(map[string]*models.PDCStats)

	for _, s := range sessions {
		if s.Site != site {
			continue
		}

		if _, exists := pdcMap[s.PDC]; !exists {
			pdcMap[s.PDC] = &models.PDCStats{
				PDC: s.PDC,
			}
		}

		stats := pdcMap[s.PDC]
		stats.Total++
		if s.StateOfCharge == 0 {
			stats.OK++
		} else {
			stats.NOK++
		}
	}

	var result []models.PDCStats
	for _, stats := range pdcMap {
		if stats.Total > 0 {
			stats.TauxReussite = round(float64(stats.OK)/float64(stats.Total)*100, 2)
		}
		result = append(result, *stats)
	}

	return result
}

// GetMomentCounts compte les erreurs par moment
func GetMomentCounts(sessions []models.Session) []models.MomentCount {
	counts := make(map[string]int)

	for _, s := range sessions {
		if s.StateOfCharge != 0 { // Erreur
			counts[s.Moment]++
		}
	}

	var result []models.MomentCount
	for _, moment := range MomentOrder {
		if count, exists := counts[moment]; exists {
			result = append(result, models.MomentCount{
				Moment: moment,
				Count:  count,
			})
		}
	}

	return result
}

// GetCodeOccurrences calcule les occurrences par code d'erreur
func GetCodeOccurrences(sessions []models.Session, isEVI bool) map[int]*models.CodeOccurrence {
	occurrences := make(map[int]*models.CodeOccurrence)

	for _, s := range sessions {
		if s.StateOfCharge == 0 { // Pas une erreur
			continue
		}

		var code int
		if isEVI {
			if s.EVIErrorCode != nil && *s.EVIErrorCode != 0 {
				code = *s.EVIErrorCode
			} else {
				continue
			}
		} else {
			if s.DownstreamCodePC != nil && *s.DownstreamCodePC != 0 && *s.DownstreamCodePC != 8192 {
				code = *s.DownstreamCodePC
			} else {
				continue
			}
		}

		if _, exists := occurrences[code]; !exists {
			occurrences[code] = &models.CodeOccurrence{
				Code:     code,
				ByMoment: make(map[string]int),
			}
		}

		occ := occurrences[code]
		occ.Total++
		occ.ByMoment[s.Moment]++
	}

	// Calculer les pourcentages
	total := 0
	for _, occ := range occurrences {
		total += occ.Total
	}

	for _, occ := range occurrences {
		if total > 0 {
			occ.Percentage = round(float64(occ.Total)/float64(total)*100, 2)
		}
	}

	return occurrences
}

// MapMoment mappe un step EVI vers un moment
func MapMoment(step int) string {
	switch {
	case step == 0:
		return "Fin de charge"
	case step >= 1 && step <= 2:
		return "Init"
	case step >= 4 && step <= 6:
		return "Lock Connector"
	case step == 7:
		return "CableCheck"
	case step == 8:
		return "Charge"
	case step > 8:
		return "Fin de charge"
	default:
		return "Unknown"
	}
}

// FormatMAC formate une adresse MAC
func FormatMAC(mac string) string {
	if mac == "" {
		return ""
	}

	cleaned := strings.ToUpper(strings.TrimSpace(mac))
	if strings.Contains(cleaned, ":") {
		return cleaned
	}

	cleaned = strings.ReplaceAll(cleaned, "0X", "")
	re := regexp.MustCompile(`[^0-9A-F]`)
	cleaned = re.ReplaceAllString(cleaned, "")

	if cleaned == "" {
		return ""
	}

	var pairs []string
	for i := 0; i < len(cleaned); i += 2 {
		if i+2 <= len(cleaned) {
			pairs = append(pairs, cleaned[i:i+2])
		}
	}

	return strings.Join(pairs, ":")
}

// FormatSOCEvolution formate l'évolution SOC
func FormatSOCEvolution(start, end *float64) string {
	if start != nil && end != nil {
		return fmt.Sprintf("%.0f%% → %.0f%%", *start, *end)
	}
	return ""
}

// GetChargeLink génère un lien vers la charge
func GetChargeLink(id string) string {
	return BaseChargeURL + id
}

// GetUniqueSites retourne la liste unique des sites
func GetUniqueSites(sessions []models.Session) []string {
	sitesMap := make(map[string]bool)
	for _, s := range sessions {
		if s.Site != "" {
			sitesMap[s.Site] = true
		}
	}

	var sites []string
	for site := range sitesMap {
		sites = append(sites, site)
	}

	return sites
}

// GetUniquePDCs retourne la liste unique des PDCs pour un site
func GetUniquePDCs(sessions []models.Session, site string) []string {
	pdcMap := make(map[string]bool)
	for _, s := range sessions {
		if s.Site == site && s.PDC != "" {
			pdcMap[s.PDC] = true
		}
	}

	var pdcs []string
	for pdc := range pdcMap {
		pdcs = append(pdcs, pdc)
	}

	return pdcs
}

// ParseDateRange calcule les dates de début et fin selon le mode
func ParseDateRange(mode string, year, month int, day time.Time) (time.Time, time.Time) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch mode {
	case "focus_jour":
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour)
		return start, end

	case "mois_complet":
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		return start, end

	case "j_minus_1":
		yesterday := today.Add(-24 * time.Hour)
		return yesterday, yesterday.Add(24 * time.Hour)

	case "semaine_minus_1":
		start := today.Add(-7 * 24 * time.Hour)
		return start, today

	case "toute_periode":
		// Du min au max
		start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		end := today.Add(24 * time.Hour)
		return start, end

	default:
		return today, today.Add(24 * time.Hour)
	}
}

// Fonctions utilitaires

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func round(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}

// GetTop10Sites retourne les top 10 sites avec le plus de charges
func GetTop10Sites(sessions []models.Session) []models.SiteStats {
	stats := GetStatsBySite(sessions)

	// Tri par total décroissant
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Total > stats[i].Total {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if len(stats) > 10 {
		stats = stats[:10]
	}

	return stats
}

// GetActiveDefauts retourne les défauts actifs
func GetActiveDefauts(defauts []models.Defaut, filters models.Filters) []models.Defaut {
	var active []models.Defaut

	for _, d := range defauts {
		// Filtre site
		if len(filters.Sites) > 0 && !contains(filters.Sites, d.Site) {
			continue
		}

		// Seulement les défauts actifs (date_fin IS NULL)
		if d.DateFin == nil {
			active = append(active, d)
		}
	}

	return active
}
