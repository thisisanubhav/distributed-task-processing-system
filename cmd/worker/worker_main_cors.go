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

// ✅ FIX 1: int64 for atomic ops — plain int++ across goroutines = data race
var jobs_done   int64
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

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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

	// ✅ FIX 2: pass rdb into handler so it can query live queue length
	http.HandleFunc("/metrics", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		metrics_handler(w, r, rdb)
	}))

	// ✅ FIX 3: HTTP server in goroutine so wg.Wait() is reachable
	go func() {
		log.Println("Metrics server on port", PORT)
		if err := http.ListenAndServe(PORT, nil); err != nil {
			log.Fatal("Metrics server error: ", err)
		}
	}()

	log.Printf("Started %d workers\n", n)
	wg.Wait()
}

func metrics_handler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	// ✅ FIX 4: query Redis directly — this is the REAL live queue length
	// The old code had total_jobs_in_queue as a global that was never updated,
	// so it was always 0. This gives the true count every single request.
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

	// ✅ FIX 5: use json.NewEncoder — no string concatenation
	// ✅ FIX 6: atomic.LoadInt64 to safely read while goroutines write
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		TotalJobsInQueue: queueLen,
		JobsDone:         atomic.LoadInt64(&jobs_done),
		JobsFailed:       atomic.LoadInt64(&jobs_failed),
	})
}

func Run_Worker(id int, rdb *redis.Client, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[Worker %d] Started\n", id)

	for {
		res, err := rdb.BLPop(ctx, 0, "task_queue").Result()
		if err != nil {
			log.Printf("[Worker %d] Redis error: %v\n", id, err)
			// ✅ FIX 7: count Redis errors as failures too
			atomic.AddInt64(&jobs_failed, 1)
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(res[1]), &t); err != nil {
			log.Printf("[Worker %d] Parse error: %v\n", id, err)
			atomic.AddInt64(&jobs_failed, 1)
			continue
		}

		log.Printf("[Worker %d] Processing: type=%s\n", id, t.Type)

		if err := Process_Task(id, t); err != nil {
			log.Printf("[Worker %d] Failed: %v\n", id, err)
			// ✅ FIX 8: atomic increment — safe across goroutines
			atomic.AddInt64(&jobs_failed, 1)
		} else {
			log.Printf("[Worker %d] Done: type=%s\n", id, t.Type)
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
		log.Printf("[Worker %d] Sent email to %v\n", workerID, t.Payload["to"])
		return nil
	case "resize_image":
		time.Sleep(1 * time.Second)
		log.Printf("[Worker %d] Resized image\n", workerID)
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
