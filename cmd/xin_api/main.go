package main

import (
	_ "Xin-api/config"
	"Xin-api/internal/store/postgresql"
)

func main() {
	_ = postgresql.NewPostgres()
}
