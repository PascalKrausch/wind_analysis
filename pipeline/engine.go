package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"wind_analysis/internal/client"
	"wind_analysis/internal/database"
	"wind_analysis/models"
)

// Engine steuert das Concurrency-Handling, das Channing der Channels und den Bulk-Writer.
type Engine struct {
	cfg         *models.Config
	db          *database.DB
	apiClient   *client.SmartClient
	rateLimiter *client.RateLimiter
}

// NewEngine erstellt eine neue Pipeline-Instanz.
func NewEngine(cfg *models.Config, db *database.DB, apiClient *client.SmartClient, rateLimiter *client.RateLimiter) *Engine {
	return &Engine{
		cfg:         cfg,
		db:          db,
		apiClient:   apiClient,
		rateLimiter: rateLimiter,
	}
}

// Run startet das Fan-Out/Fan-In der Pipeline und blockiert, bis alle Aufgaben erledigt sind.
func Run(e *Engine, ctx context.Context) error {
	startTime := time.Now()

	// Channels initialisieren (mit Puffern für flüssigen Durchsatz)
	taskChan := make(chan models.FetchTask, e.cfg.Pipeline.Concurrency*2)
	resultChan := make(chan []models.WindRecord, e.cfg.Pipeline.Concurrency*2)

	// Atomic-Zähler für einfaches Live-Monitoring
	var totalRecords int64
	var processedTasks int64

	// 1. Task-Generator starten (Produzent)
	tasks, err := GenerateLocationTasks(e.cfg)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der Tasks: %w", err)
	}
	totalTasks := len(tasks)
	log.Printf("🚀 Starte Pipeline mit %d Tasks auf %d Worker-Routinen...", totalTasks, e.cfg.Pipeline.Concurrency)

	go func() {
		defer close(taskChan)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case taskChan <- task:
			}
		}
	}()

	// 2. Worker-Pool starten (Fan-Out)
	var workerWg sync.WaitGroup
	for i := 1; i <= e.cfg.Pipeline.Concurrency; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			e.worker(ctx, workerID, taskChan, resultChan, &processedTasks, totalTasks)
		}(i)
	}

	// Schließer-Routine für resultChan nach Beendigung aller Worker
	go func() {
		workerWg.Wait()
		close(resultChan)
	}()

	// 3. Database Writer Consumer (Fan-In / Bulk Copy)
	// Dieser Stream empfängt fertig aufbereitete WindRecords und schiebt sie via CopyFrom direkt in DB
	var writerWg sync.WaitGroup
	writerWg.Add(1)

	var writeErr error
	go func() {
		defer writerWg.Done()
		for records := range resultChan {
			if len(records) == 0 {
				continue
			}

			err := e.db.SaveBatchWindData(ctx, records)
			if err != nil {
				log.Printf("❌ DB Bulk-Insert Fehler: %v", err)
				writeErr = err
				return
			}
			atomic.AddInt64(&totalRecords, int64(len(records)))
		}
	}()

	// Warten bis der Writer alle Pakete abgearbeitet hat
	writerWg.Wait()

	if writeErr != nil {
		return fmt.Errorf("pipeline abgebrochen aufgrund von DB-Fehlern: %w", writeErr)
	}

	duration := time.Since(startTime)
	log.Printf("✅ Pipeline erfolgreich beendet: %d Datensätze geschrieben in %v (%.0f Rec/s)",
		totalRecords, duration.Round(time.Millisecond), float64(totalRecords)/duration.Seconds())

	return nil
}

// worker arbeitet kontinuierlich Tasks ab, fragt API ab und transformiert Responses.
func (e *Engine) worker(
	ctx context.Context,
	id int,
	tasks <-chan models.FetchTask,
	results chan<- []models.WindRecord,
	processedTasks *int64,
	totalTasks int,
) {
	for task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Rate Limiter Token anfordern
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return // Context storniert
		}

		// API Request über SmartClient (enthält Retry-Mechanismen)
		resp, err := e.apiClient.FetchSeamlessData(ctx, task.Latitude, task.Longitude, task.StartDate, task.EndDate)
		if err != nil {
			log.Printf("[Worker %d] ⚠️ Fehler bei Task %s (%f, %f): %v", id, task.ID, task.Latitude, task.Longitude, err)
			continue
		}

		records, err := e.apiClient.TransformResponseToRecords(task, resp)
		if err != nil {
			log.Printf("[Worker %d] ⚠️ Fehler bei der Transformation der API-Antwort für Task %s (%f, %f): %v", id, task.ID, task.Latitude, task.Longitude, err)
			continue
		}

		// Daten über Result Channel an den DB-Writer senden
		select {
		case <-ctx.Done():
			return
		case results <- records:
		}

		done := atomic.AddInt64(processedTasks, 1)
		if done%10 == 0 || done == int64(totalTasks) {
			log.Printf("Progress: [%d/%d] Tasks abgearbeitet (%.1f%%)", done, totalTasks, (float64(done)/float64(totalTasks))*100)
		}
	}
}
