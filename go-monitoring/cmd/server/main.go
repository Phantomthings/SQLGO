package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/monitoring/charging-stations/internal/database"
	"github.com/monitoring/charging-stations/internal/handlers"
)

func main() {
	log.Println("🚀 Starting Charging Stations Monitoring Server...")

	// Connexion à la base de données
	db := database.GetDB()
	defer db.Close()

	// Créer le routeur
	r := mux.NewRouter()

	// Enregistrer les handlers
	h := handlers.New(db)
	h.RegisterRoutes(r)

	// Configuration du serveur
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Démarrer le serveur dans une goroutine
	go func() {
		log.Printf("✅ Server listening on http://localhost%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Rafraîchir le cache périodiquement (toutes les heures)
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			log.Println("⏰ Scheduled cache refresh...")
			if err := db.RefreshCache(); err != nil {
				log.Printf("Error refreshing cache: %v", err)
			}
		}
	}()

	// Attendre un signal d'interruption
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ticker.Stop()
}
