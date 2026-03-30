package main

import (
	"WorkQueue/internal/logger"
	"WorkQueue/internal/task"
	"WorkQueue/internal/worker"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var total_jobs_in_queue int64
var jobs_done int = 0
var jobs_failed int = 0

var ctx = context.Background()

// Connect to Redis
func connectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Could not parse Redis URL:", err)
	}
	return redis.NewClient(opt)
}

func main() {

	// ✅ Load local env (safe for Render too)
	_ = godotenv.Load("config.env")

	// ✅ Render PORT support (IMPORTANT)
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("PORT_WORKER")
	}
	if port == "" {
		port = "8081"
	}
	PORT := ":" + port

	rdb := connectRedis()

	// ✅ Check Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Cannot reach Redis:", err)
	}
	log.Println("Connected to Redis successfully")

	// ✅ Start worker goroutines
	var wg sync.WaitGroup
	numWorkers := 3

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go RunWorker(i, rdb, ctx, &wg)
	}

	log.Printf("Started %d workers, waiting for jobs...\n", numWorkers)

	// ✅ Metrics endpoint (for dashboard)
	http.HandleFunc("/metrics", metricsHandler)

	log.Println("Metrics server starting on port", PORT)

	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		log.Fatal("Metrics server error:", err)
	}

	wg.Wait()
}

// Metrics API
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := task.Metrics{
		Total_jobs_in_queue: total_jobs_in_queue,
		Jobs_done:           jobs_done,
		Jobs_failed:         jobs_failed,
	}

	res, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, "Error encoding metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

// Worker logic
func RunWorker(id int, rdb *redis.Client, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Printf("[Worker %d] Started, waiting for jobs...\n", id)

	for {
		// BLPOP blocks until job arrives
		res, err := rdb.BLPop(ctx, 0, "task_queue").Result()
		if err != nil {
			log.Println("Redis error:", err)
			continue
		}

		total_jobs_in_queue, _ = rdb.LLen(ctx, "task_queue").Result()

		var taskToExecute task.Task

		err = json.Unmarshal([]byte(res[1]), &taskToExecute)
		if err != nil {
			log.Println("Failed to parse task:", err)
			continue
		}

		log.Printf("[Worker %d] Task received: %s\n", id, res[1])

		retriesLeft := taskToExecute.Retries

		errWorker := worker.Process_Task(taskToExecute)

		if errWorker != nil {
			jobs_failed++
			logger.LogFailure(taskToExecute, errWorker)

			retriesLeft--
			log.Printf("[Worker %d] Error: %v. Retrying...\n", id, errWorker)

			if retriesLeft > 0 {
				taskToExecute.Retries = retriesLeft
				rdb.RPush(ctx, "task_queue", taskToExecute)
			} else {
				log.Printf("[Worker %d] Task failed after retries\n", id)
			}
		} else {
			jobs_done++
			logger.LogSuccess(taskToExecute)
			log.Printf("[Worker %d] Task processed successfully\n", id)
		}
	}
}
