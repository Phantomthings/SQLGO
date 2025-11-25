# 🔋 Monitoring Bornes de Recharge - Go + HTMX

Application de monitoring des bornes de recharge électrique convertie depuis Streamlit vers **Go + HTMX + Chart.js**.

## 📋 Fonctionnalités

### 13 Onglets d'analyse :

1. **Vue d'ensemble** - Dashboard principal avec défauts actifs, transactions suspectes, alertes
2. **Générale** - KPIs globaux et récapitulatif erreurs par site/moment
3. **Comparaison par site** - Statistiques et analyse temporelle (heatmap, distributions)
4. **Détails PDC** - Analyse par Point De Charge avec graphiques erreurs EVI
5. **Statistiques** - Énergie, puissance, SOC, durées, véhicules
6. **Projection pivot** - Table pivot Moments × Codes avec coloration
7. **Tentatives multiples** - Utilisateurs avec multiples tentatives dans l'heure
8. **Transactions suspectes** - Charges < 1 kWh
9. **Erreur Moment** - Top 3 erreurs EVI/Downstream, répartition par phase
10. **Erreur Spécifique** - Filtres MAC/codes, histogrammes véhicules et temporels
11. **Alertes** - Erreurs récurrentes par PDC
12. **Évolution** - Taux de réussite mensuel
13. **Historique Défauts** - Défauts actifs et résolus avec statistiques

### Filtres globaux :
- **Sites** : Sélection multiple avec option "Tous les sites"
- **Période** : Focus Jour, Focus Mois, J-1, Semaine -1, Toute la période
- **Type d'erreur** : EVI, DownStream
- **Moment** : Init, Lock Connector, CableCheck, Charge, Fin de charge
- **Raccourcis** : Avant charge, Charge, Fin de charge

## 🏗️ Architecture

```
go-monitoring/
├── cmd/server/           # Point d'entrée
│   └── main.go
├── internal/
│   ├── models/          # Structures de données
│   ├── database/        # Connexion MySQL + Cache
│   ├── handlers/        # Handlers HTTP + HTMX
│   └── utils/           # Fonctions utilitaires
├── web/
│   ├── templates/       # Templates HTML
│   └── static/          # CSS, JS, Images
└── go.mod
```

### Stack technique :
- **Backend** : Go 1.21+ avec Gorilla Mux
- **Frontend** : HTMX pour interactions, Alpine.js pour réactivité
- **Graphiques** : Chart.js côté client
- **Styling** : Tailwind CSS (CDN)
- **Base de données** : MySQL avec cache mémoire

## 🚀 Installation et Démarrage

### Prérequis
- Go 1.21 ou supérieur
- Accès au serveur MySQL (162.19.251.55:3306)

### 1. Récupérer les dépendances
```bash
cd go-monitoring
go mod download
```

### 2. Compiler
```bash
# Compilation simple
go build -o bin/monitoring cmd/server/main.go

# OU utiliser le Makefile
make build
```

### 3. Lancer le serveur
```bash
# Directement
./bin/monitoring

# OU avec Make
make run
```

Le serveur démarre sur **http://localhost:8080**

## 📊 Base de données

### Connexion MySQL :
- **Host** : 162.19.251.55:3306
- **Database** : Charges
- **User** : nidec
- **Tables KPI** : kpi_sessions, kpi_alertes, kpi_defauts_log, etc.

### Cache automatique :
- Chargement initial au démarrage
- Refresh automatique toutes les heures
- Endpoint manuel : `POST /api/refresh-cache`

## 🎨 Fonctionnalités conservées

✅ **Toutes les requêtes SQL** identiques à Streamlit
✅ **Même logique métier** (calculs, agrégations, pivots)
✅ **Tous les graphiques** (bar, pie, heatmap, histogrammes)
✅ **Même navigation** par onglets
✅ **Filtres identiques** avec synchronisation temps réel
✅ **Liens externes** vers ELTO (https://elto.nidec-asi-online.com)

## 📁 Templates importants

### index.html
Template principal avec :
- Header avec logos
- Filtres globaux (sites, dates, types, moments)
- KPIs summary
- Navigation tabs
- Container pour contenu dynamique

### tab_overview.html
Dashboard principal avec :
- Défauts actifs (cartes colorées)
- Transactions suspectes
- Tentatives multiples
- Alertes
- Top 10 sites (graphiques Chart.js)

### Autres tabs
Templates similaires pour les 12 autres onglets (à compléter selon les besoins)

## 🔧 Développement

### Ajouter un nouveau template :
1. Créer `web/templates/tab_*.html`
2. Ajouter le handler dans `internal/handlers/handlers.go`
3. Enregistrer la route dans `RegisterRoutes()`

### Modifier les graphiques :
Les graphiques Chart.js sont définis dans les `<script>` des templates.
Exemple : `tab_overview.html` pour les graphiques du dashboard.

## 📝 TODO / Améliorations

Les templates suivants sont à finaliser :
- [ ] tab_general.html
- [ ] tab_comparison.html
- [ ] tab_pdc_details.html
- [ ] tab_stats.html
- [ ] tab_projection.html
- [ ] tab_attempts.html
- [ ] tab_suspicious.html
- [ ] tab_error_moment.html
- [ ] tab_error_specific.html
- [ ] tab_alerts.html
- [ ] tab_evolution.html
- [ ] tab_defects.html

## 🐛 Debugging

### Logs
Le serveur affiche des logs détaillés :
- ✅ Connexion MySQL réussie
- 🔄 Refresh du cache
- ⚠️ Erreurs SQL
- 📊 Chargement des données

### Endpoints utiles
- `GET /` - Page principale
- `POST /api/filters` - Filtrer les données
- `POST /api/kpis` - Récupérer les KPIs
- `POST /tabs/{tab_name}` - Charger un onglet
- `POST /api/refresh-cache` - Forcer le refresh du cache

## 📦 Build Production

```bash
# Build avec optimisations
make build-prod

# Le binaire est dans bin/monitoring
# Déployer avec les dossiers web/ et go.mod
```

## 🔒 Sécurité

⚠️ **Important** : Les credentials MySQL sont en dur dans le code pour POC.
En production, utilisez des variables d'environnement :

```go
dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
    os.Getenv("DB_USER"),
    os.Getenv("DB_PASS"),
    os.Getenv("DB_HOST"),
    os.Getenv("DB_NAME"),
)
```

## 📞 Support

Pour toute question ou amélioration, consulter le code source ou la documentation Go/HTMX.

## 📄 Licence

Propriétaire - NIDEC/ELTO
