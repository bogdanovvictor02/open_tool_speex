package main

import (
	"log"

	"open_tool_speex/internal/config"
	"open_tool_speex/internal/processor"
)

func main() {
	// Parse command line flags
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create processor
	proc := processor.NewProcessor(cfg)

	// Process audio
	if err := proc.Process(); err != nil {
		log.Fatalf("Processing error: %v", err)
	}

	log.Printf("AEC processing completed: %s -> %s", cfg.MicFile, cfg.OutputFile)
}
