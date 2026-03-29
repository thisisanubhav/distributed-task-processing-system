package main

import (
	"WorkQueue/internal/task"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

func connectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Could not parse Redis URL:", err)
	}
	return redis.NewClient(opt)
}

// corsMiddleware wraps any handler and adds CORS headers so browsers
// (including the dashboard) can call this service cross-origin.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight request — browsers send OPTIONS before POST
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

	PORT := ":" + os.Getenv("PORT_PRODUCER")
	if PORT == ":" {
		log.Fatal("PORT_PRODUCER is not set in config.env")
	}

	rdb = connectRedis()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Cannot reach Redis: ", err)
	}
	log.Println("Connected to Redis successfully")

	// ✅ Wrap handler with CORS middleware
	http.HandleFunc("/enqueue", corsMiddleware(post_handler))

	log.Println("Starting producer server on port", PORT)
	if err = http.ListenAndServe(PORT, nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func post_handler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST request accepted", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	var t task.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if t.Type == "" {
		http.Error(w, "Bad Request: 'type' field is required", http.StatusBadRequest)
		return
	}

	if err := validatePayload(t); err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	b, err := json.Marshal(t)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	res, err := rdb.RPush(ctx, "task_queue", b).Result()
	if err != nil {
		http.Error(w, "Internal server error: could not enqueue task", http.StatusInternalServerError)
		return
	}

	log.Printf("Task enqueued: type=%s queue_length=%d\n", t.Type, res)
	fmt.Fprintf(w, "Task of type '%s' has been successfully added to the queue", t.Type)
}

func validatePayload(t task.Task) error {
	switch t.Type {
	case "send_email":
		if t.Payload["to"] == nil || t.Payload["subject"] == nil {
			return fmt.Errorf("send_email requires 'to' and 'subject' in payload")
		}
	case "resize_image":
		if t.Payload["new_x"] == nil || t.Payload["new_y"] == nil {
			return fmt.Errorf("resize_image requires 'new_x' and 'new_y' in payload")
		}
	case "generate_pdf":
		// no required fields
	default:
		return fmt.Errorf("unsupported task type: '%s'", t.Type)
	}
	return nil
}
