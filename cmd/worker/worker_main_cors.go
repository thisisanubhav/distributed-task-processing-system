package main

import (
	"WorkQueue/internal/task"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var jobs_done int64
var jobs_failed int64

var ctx = context.Background()

func connectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Could not parse Redis URL:", err)
	}
	return redis.NewClient(opt)
}

// corsMiddleware wraps any handler and adds CORS headers so the dashboard
// can call /metrics from the browser without being blocked.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func main() {
	err := godotenv.Load("config.env")
	if err != nil {
		log.Fatal("Error loading config.env: ", err)
	}

	PORT := ":" + os.Getenv("PORT_WORKER")
	if PORT == ":" {
		log.Fatal("PORT_WORKER is not set in config.env")
	}

	rdb := connectRedis()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Cannot reach Redis: ", err)
	}
	log.Println("Connected to Redis successfully")

	var wg sync.WaitGroup
	n := 3

	for i := 0; i < n; i++ {
		wg.Add(1)
		go Run_Worker(i+1, rdb, ctx, &wg)
	}

	// ✅ Wrap handler with CORS middleware
	http.HandleFunc("/metrics", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		metrics_handler(w, r, rdb)
	}))

	// HTTP server in goroutine so wg.Wait() is reachable
	go func() {
		log.Println("Metrics server starting on port", PORT)
		if err := http.ListenAndServe(PORT, nil); err != nil {
			log.Fatal("Metrics server error: ", err)
		}
	}()

	log.Printf("Started %d workers, waiting for jobs...\n", n)
	wg.Wait()
	log.Println("All workers stopped. Shutting down.")
}

func metrics_handler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET request allowed", http.StatusMethodNotAllowed)
		return
	}

	queueLen, err := rdb.LLen(ctx, "task_queue").Result()
	if err != nil {
		http.Error(w, "Could not fetch queue length", http.StatusInternalServerError)
		return
	}

	type MetricsResponse struct {
		TotalJobsInQueue int64 `json:"total_jobs_in_queue"`
		JobsDone         int64 `json:"jobs_done"`
		JobsFailed       int64 `json:"jobs_failed"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		TotalJobsInQueue: queueLen,
		JobsDone:         atomic.LoadInt64(&jobs_done),
		JobsFailed:       atomic.LoadInt64(&jobs_failed),
	})
}

func Run_Worker(id int, rdb *redis.Client, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[Worker %d] Started, waiting for jobs...\n", id)

	for {
		res, err := rdb.BLPop(ctx, 0, "task_queue").Result()
		if err != nil {
			log.Printf("[Worker %d] Redis error: %v\n", id, err)
			atomic.AddInt64(&jobs_failed, 1)
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(res[1]), &t); err != nil {
			log.Printf("[Worker %d] Failed to parse task: %v\n", id, err)
			atomic.AddInt64(&jobs_failed, 1)
			continue
		}

		log.Printf("[Worker %d] Processing task: type=%s\n", id, t.Type)

		if err := Process_Task(id, t); err != nil {
			log.Printf("[Worker %d] Task failed: %v\n", id, err)
			atomic.AddInt64(&jobs_failed, 1)
		} else {
			log.Printf("[Worker %d] Task completed: type=%s\n", id, t.Type)
			atomic.AddInt64(&jobs_done, 1)
		}
	}
}

func Process_Task(workerID int, t task.Task) error {
	if t.Payload == nil {
		return fmt.Errorf("payload is empty")
	}

	switch t.Type {
	case "send_email":
		time.Sleep(2 * time.Second)
		log.Printf("[Worker %d] Sent email to %v, subject: %v\n",
			workerID, t.Payload["to"], t.Payload["subject"])
		return nil

	case "resize_image":
		time.Sleep(1 * time.Second)
		log.Printf("[Worker %d] Resized image to x=%v y=%v\n",
			workerID, t.Payload["new_x"], t.Payload["new_y"])
		return nil

	case "generate_pdf":
		time.Sleep(3 * time.Second)
		log.Printf("[Worker %d] Generated PDF\n", workerID)
		return nil

	case "":
		return fmt.Errorf("task type is empty")

	default:
		return fmt.Errorf("unsupported task type: '%s'", t.Type)
	}
}
