package handlers

import (
	"fmbq-server/database"
)

var DB *database.DB

func InitializeHandlers(db *database.DB) {
	DB = db
}
