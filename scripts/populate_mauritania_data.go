package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Mauritanian cities and their quartiers
var mauritaniaData = map[string][]string{
	"Nouakchott": {
		"Tevragh Zeina", "Teyarett", "Ksar", "Toujounine", "Sebkha", 
		"El Mina", "Arafat", "Riyad", "Dar Naim", "Tevragh Zeina Nord",
		"Tevragh Zeina Sud", "Teyarett Nord", "Teyarett Sud", "Ksar Nord",
		"Ksar Sud", "Toujounine Nord", "Toujounine Sud", "Sebkha Nord",
		"Sebkha Sud", "El Mina Nord", "El Mina Sud", "Arafat Nord",
		"Arafat Sud", "Riyad Nord", "Riyad Sud", "Dar Naim Nord",
		"Dar Naim Sud",
	},
	"Nouadhibou": {
		"Centre Ville", "Cansado", "Chami", "Port", "Aéroport",
		"Baie du Lévrier", "Cap Blanc", "Plage", "Zone Industrielle",
		"Résidentiel", "Commercial", "Administratif",
	},
	"Rosso": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Rive Gauche",
		"Rive Droite", "Port", "Gare", "Marché", "Administratif",
		"Résidentiel", "Commercial",
	},
	"Kaédi": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Rive Gauche",
		"Rive Droite", "Port", "Gare", "Marché", "Administratif",
		"Résidentiel", "Commercial", "Universitaire",
	},
	"Kiffa": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Marché",
		"Administratif", "Résidentiel", "Commercial", "Industriel",
	},
	"Atar": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Aéroport",
		"Marché", "Administratif", "Résidentiel", "Commercial",
		"Touristique", "Oasis",
	},
	"Zouérat": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Mine",
		"Résidentiel", "Commercial", "Administratif", "Industriel",
	},
	"Aioun": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Marché",
		"Administratif", "Résidentiel", "Commercial",
	},
	"Boutilimit": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Marché",
		"Administratif", "Résidentiel", "Commercial",
	},
	"Selibaby": {
		"Centre", "Nord", "Sud", "Est", "Ouest", "Marché",
		"Administratif", "Résidentiel", "Commercial",
	},
}

func main() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://khalil:44441318@127.0.0.1/fmbq?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully!")

	// Insert cities
	cityIDs := make(map[string]string)
	for cityName, quartiers := range mauritaniaData {
		cityID := uuid.New().String()
		
		// Determine region based on city
		region := getRegion(cityName)
		
		_, err := db.Exec(`
			INSERT INTO cities (id, name, name_ar, region, is_active) 
			VALUES ($1, $2, $3, $4, $5)
		`, cityID, cityName, cityName, region, true)
		
		if err != nil {
			log.Printf("Error inserting city %s: %v", cityName, err)
			continue
		}
		
		cityIDs[cityName] = cityID
		fmt.Printf("Inserted city: %s (Region: %s)\n", cityName, region)
		
		// Insert quartiers for this city
		for _, quartierName := range quartiers {
			quartierID := uuid.New().String()
			
			_, err := db.Exec(`
				INSERT INTO quartiers (id, city_id, name, name_ar, is_active) 
				VALUES ($1, $2, $3, $4, $5)
			`, quartierID, cityID, quartierName, quartierName, true)
			
			if err != nil {
				log.Printf("Error inserting quartier %s for city %s: %v", quartierName, cityName, err)
				continue
			}
			
			fmt.Printf("  - Inserted quartier: %s\n", quartierName)
		}
	}

	fmt.Println("Data insertion completed!")
}

func getRegion(cityName string) string {
	regions := map[string]string{
		"Nouakchott": "Nouakchott",
		"Nouadhibou": "Dakhlet Nouadhibou",
		"Rosso":      "Trarza",
		"Kaédi":      "Gorgol",
		"Kiffa":      "Assaba",
		"Atar":       "Adrar",
		"Zouérat":    "Tiris Zemmour",
		"Aioun":      "Hodh El Gharbi",
		"Boutilimit": "Trarza",
		"Selibaby":   "Guidimaka",
	}
	
	if region, exists := regions[cityName]; exists {
		return region
	}
	return "Autre"
}
