package yaitracker

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS

//go:embed static/*
var StaticFS embed.FS
