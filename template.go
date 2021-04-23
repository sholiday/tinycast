package tinycast

import "embed"

// Templates holds the HTML/JavaScript/CSS/Images for the web app.
//go:embed templates
//go:embed templates/favicon/*
var Templates embed.FS
