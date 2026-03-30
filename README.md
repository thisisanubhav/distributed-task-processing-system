<div align="center">

# ⚡ Distributed Task Processing System

### A production-grade background job queue built with Go and Redis

*Producer · Worker · Redis Queue · Real-time Dashboard · Cloud Deployed*

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![Redis](https://img.shields.io/badge/Redis-Queue-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io)
[![Render](https://img.shields.io/badge/Backend-Render-46E3B7?style=flat-square&logo=render&logoColor=white)](https://render.com)
[![Vercel](https://img.shields.io/badge/Frontend-Vercel-000000?style=flat-square&logo=vercel&logoColor=white)](https://vercel.com)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)

<br/>

| 🌐 Dashboard | 📡 Producer API | 📊 Worker Metrics |
|:---:|:---:|:---:|
| [distributed-task-processing-system.vercel.app](https://distributed-task-processing-system.vercel.app/) | [workqueue-producer-kz2k.onrender.com](https://workqueue-producer-kz2k.onrender.com) | [worker.onrender.com/metrics](https://distributed-task-processing-system-worker.onrender.com/metrics) |

</div>

---

## 🧠 The Problem — A Real Example

Imagine you're building a food delivery app. A user places an order and hits **"Place Order"**.

Behind the scenes, your server needs to:

1. Save the order to the database
2. Send a confirmation email to the user
3. Send an SMS notification
4. Notify the restaurant
5. Generate and send a PDF receipt

**Without a task queue**, all of this happens inside the same API call:

```
User clicks "Place Order"
        ↓
  Server saves order          ~50ms
  Server sends email          ~800ms   ← waiting for Gmail
  Server sends SMS            ~600ms   ← waiting for Twilio
  Server notifies restaurant  ~400ms   ← waiting for their API
  Server generates PDF        ~300ms   ← CPU-heavy work
        ↓
User sees "Order Confirmed"   after ~2.1 seconds  😴
```

Your user stares at a spinner for 2 full seconds. If any one of those external services is slow or down, your entire API hangs — or worse, times out.

---

## ✅ How This System Solves It

With a task queue, your API does one thing: save the order and respond. Everything else is handed off to background workers.

```
User clicks "Place Order"
        ↓
  Server saves order          ~50ms
  Server pushes 4 tasks       ~5ms    ← just adding items to a list
        ↓
User sees "Order Confirmed"   in ~55ms  🚀

Meanwhile, in the background:
  Worker 1 → sends confirmation email
  Worker 2 → sends SMS
  Worker 3 → notifies restaurant + generates PDF
```

**The user gets a response 40× faster.** The slow work still happens — just not in the user's way. And if the email provider is down, only that task fails and retries. Your API keeps running perfectly.

This is the exact pattern used by companies like Uber (job dispatch), Airbnb (email/notifications), and GitHub (CI pipeline triggers).

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Your Application                        │
└──────────────────────────┬───────────────────────────────────┘
                           │  POST /enqueue  (JSON task)
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                    Producer Service                          │
│          Validates → Serialises → RPUSH to Redis             │
│          Returns HTTP 200 instantly ← your app resumes       │
└──────────────────────────┬───────────────────────────────────┘
                           │  RPUSH
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                     Redis Queue                              │
│                   task_queue  (FIFO list)                    │
└──────┬───────────────────┬──────────────────────┬───────────┘
       │ BLPOP             │ BLPOP                │ BLPOP
       ▼                   ▼                      ▼
┌────────────┐      ┌────────────┐        ┌────────────┐
│ Worker #1  │      │ Worker #2  │        │ Worker #3  │
│ goroutine  │      │ goroutine  │        │ goroutine  │
└─────┬──────┘      └─────┬──────┘        └─────┬──────┘
      └────────────────────┴──────────────────────┘
                           │
                           ▼
             ┌─────────────────────────┐
             │     GET /metrics        │
             │  Real-time Dashboard    │
             └─────────────────────────┘
```

---

## ✨ Features

| Feature | Details |
|---------|---------|
| ⚡ **Instant response** | API returns in <5ms — never blocks on background work |
| 🔀 **Concurrent workers** | 3 goroutines process jobs in parallel |
| 🔁 **Retry mechanism** | Configurable retries per job on failure |
| 📊 **Real-time dashboard** | Live metrics, job history, worker status — polls every 3s |
| 🧩 **Modular tasks** | Add a new job type by writing one `case` block |
| 🛡️ **Race-condition safe** | `sync/atomic` counters across goroutines |
| 📝 **Structured logging** | Every job logged with worker ID, type, and result |
| 🌐 **Production deployed** | Live on Render + Vercel, always-on via UptimeRobot |

---

## ⚙️ Tech Stack

| Layer | Technology | Role |
|-------|-----------|------|
| Backend | **Go 1.23** | Producer & Worker services |
| Queue | **Redis (Upstash)** | FIFO task queue via RPUSH / BLPOP |
| Concurrency | **Goroutines + sync/atomic** | Parallel job processing, race-safe counters |
| Frontend | **HTML / CSS / JS** | Real-time admin dashboard |
| Backend Deploy | **Render** | Producer & Worker hosting |
| Frontend Deploy | **Vercel** | Dashboard hosting |

---

## 📡 API Reference

### Enqueue a task

```http
POST https://workqueue-producer-kz2k.onrender.com/enqueue
Content-Type: application/json
```

```json
{
  "type": "send_email",
  "retries": 3,
  "payload": {
    "to": "test@gmail.com",
    "subject": "Hello from WorkQueue 🚀"
  }
}
```

**Response**
```
Task of type 'send_email' has been successfully added to the queue
```

---

### Get live metrics

```http
GET https://distributed-task-processing-system-worker.onrender.com/metrics
```

**Response**
```json
{
  "total_jobs_in_queue": 2,
  "jobs_done": 10,
  "jobs_failed": 1
}
```

---

## 🧩 Supported Task Types

| Type | Required Payload Fields | Processing Time |
|------|------------------------|----------------|
| `send_email` | `to`, `subject` | ~2s |
| `resize_image` | `new_x`, `new_y` | ~1s |
| `generate_pdf` | `title` (optional) | ~3s |

### Adding a new task type

Open `cmd/worker/main.go` and add one `case`:

```go
switch t.Type {
case "send_email":
    // existing...

case "your_new_task":     // ← just add this
    // your logic here
    return nil
}
```

That's it. No config changes, no restarts.

---

## 🚀 Run Locally

### Prerequisites
- Go 1.21+
- Redis running locally

### Setup

```bash
# 1. Clone
git clone https://github.com/thisisanubhav/distributed-task-processing-system.git
cd distributed-task-processing-system

# 2. Create config
cp config.env.example config.env
# Edit config.env — set your Redis URL and ports

# 3. Start Redis (macOS)
brew services start redis

# 4. Run Producer (Terminal 1)
go run cmd/producer/main.go

# 5. Run Worker (Terminal 2)
go run cmd/worker/main.go

# 6. Open dashboard
open frontend/index.html
```

### Test it with curl

```bash
curl -X POST http://localhost:8080/enqueue \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "retries": 3,
    "payload": {
      "to": "test@example.com",
      "subject": "Hello from WorkQueue!"
    }
  }'
```

---

## 🔑 Key Engineering Decisions

**Why `RPUSH` + `BLPOP`?**
`RPUSH` adds to the tail, `BLPOP` pops from the head — correct FIFO ordering. `BLPOP` with timeout `0` blocks until a job arrives, so workers use zero CPU while idle. No polling loop needed.

**Why `sync/atomic` instead of a mutex?**
Three goroutines concurrently increment `jobs_done` and `jobs_failed`. `atomic.AddInt64` is a single CPU instruction — faster and simpler than acquiring a lock.

**Why two separate services?**
Independent scaling. If the queue backs up, you deploy more workers without touching the producer. This mirrors how production systems like Sidekiq and Celery work.

**Why goroutines over OS threads?**
Go goroutines start at ~2KB of stack vs ~8MB for OS threads. You can run thousands simultaneously with negligible overhead.

---

## 📊 Concurrency in action

```
Time →    0s        1s        2s        3s        4s
           │         │         │         │         │
Worker 1  [send_email──────────]         [generate_pdf────────]
Worker 2       [resize_image───]   [send_email──────────]
Worker 3  [generate_pdf──────────────────]   [resize_image───]
```

Three jobs that would take 6s sequentially complete in ~3s in parallel.

---

## 🆚 Compared to Alternatives

Production job queue systems exist in every major language. Here's how this project maps to the ecosystem — and why I built it from scratch instead of using one of them.

| | **This Project** | **Celery** (Python) | **BullMQ** (Node.js) | **Sidekiq** (Ruby) | **Asynq** (Go) |
|---|---|---|---|---|---|
| **Language** | Go | Python | JavaScript | Ruby | Go |
| **Queue backend** | Redis | Redis / RabbitMQ | Redis | Redis | Redis |
| **Concurrency model** | Goroutines | Multiprocessing | Worker threads | Threads | Goroutines |
| **Worker startup** | ~5ms | ~500ms | ~100ms | ~200ms | ~5ms |
| **Memory per worker** | ~2KB | ~50MB | ~30MB | ~20MB | ~2KB |


**Why build it instead of using an existing library?**

Using Celery or BullMQ would have taken 30 minutes. Building it from scratch took considerably longer — and that's the point. Writing the queue logic by hand forced me to actually understand *why* `BLPOP` beats a polling loop, *what* a data race looks like when three goroutines share a counter, and *how* FIFO ordering breaks if you mix `LPUSH` and `BLPOP`. You don't learn those things by calling `queue.add('send_email', payload)`.

---

## 🎓 What I Learned

Building this project from scratch — rather than using an existing library like Celery or BullMQ — forced me to confront problems I would never have encountered otherwise. Here's what genuinely changed how I think about software:

**Concurrency is not parallelism, and both are hard.**
I thought "just use goroutines" was the whole story. Then I ran the Go race detector (`go run -race`) on my first version and watched it flag `jobs_done++` as a data race. Three goroutines were incrementing the same integer at the same time, and the result was silently wrong. Learning the difference between a mutex (which serialises access) and `sync/atomic` (which uses a single CPU instruction) was a turning point — not just for this project, but for how I think about any shared state.

**Blocking is a feature, not a bug.**
My first worker implementation used a polling loop — check Redis every 500ms, sleep, repeat. It worked but wasted CPU doing nothing useful. Switching to `BLPOP` with timeout `0` was eye-opening: the goroutine literally suspends itself at the OS level and wakes up the instant a job arrives, using zero CPU in between. Understanding *why* this works — the operating system's blocking I/O model — made distributed systems feel less magical.

**Distributed systems fail in ways local programs don't.**
When both services run on your laptop, everything works. The moment you deploy to Render, you discover: what happens when the Worker starts before Redis is ready? What if the Producer accepts a request but Redis is temporarily unreachable? I had to add startup health checks (`rdb.Ping`), proper error propagation, and CORS headers — none of which exist in a local-only project. These aren't afterthoughts; they're the actual job of backend engineering.

**Separation of concerns scales.** Running the Producer and Worker as two independent services initially felt like unnecessary complexity for a small project. But when I needed to debug the worker without restarting the API, or think about scaling workers independently, the separation paid off immediately. I now instinctively think about services in terms of their single responsibility and what would need to change if load on one component grew 10×.

**Good tooling reveals bad code.** The Go compiler refusing to compile unused variables, the race detector finding concurrent writes, the linter flagging swallowed errors — these weren't annoyances. They were the compiler teaching me things. Every flag was a real bug I'd written.

---

## 🔥 Roadmap

- [ ] Persistent job history via Redis (replace localStorage)
- [ ] Dead letter queue for failed jobs with retry UI
- [ ] Job priority queues (high / normal / low)
- [ ] Dynamic worker scaling via `POST /workers?count=N`
- [ ] WebSocket push instead of polling
- [ ] Docker Compose for one-command local setup
- [ ] Authentication on dashboard and API

---

## 👨‍💻 Author

**Anubhav Harsh Sinha**

[![LinkedIn](https://img.shields.io/badge/LinkedIn-Anubhav%20Sinha-0A66C2?style=flat-square&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/anubhav-sinha-a70019287/)
[![GitHub](https://img.shields.io/badge/GitHub-thisisanubhav-181717?style=flat-square&logo=github&logoColor=white)](https://github.com/thisisanubhav)

---

<div align="center">

Found this useful? Drop a ⭐ — it helps others find the project.

</div>
