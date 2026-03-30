<div align="center">

# ⚡ Distributed Task Processing System

### A Scalable Background Job Processing System

*Built with Go · Redis · REST APIs · Cloud Deployment*

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square\&logo=go\&logoColor=white)](https://golang.org)
[![Redis](https://img.shields.io/badge/Redis-Queue-DC382D?style=flat-square\&logo=redis\&logoColor=white)](https://redis.io)
[![Vercel](https://img.shields.io/badge/Frontend-Vercel-black?style=flat-square\&logo=vercel)](https://vercel.com)
[![Render](https://img.shields.io/badge/Backend-Render-46E3B7?style=flat-square\&logo=render)](https://render.com)

**🌐 Frontend:** https://distributed-task-processing-system-q2zu9224f.vercel.app
**📡 Producer API:** https://workqueue-producer-kz2k.onrender.com/enqueue

</div>

---

## 🧠 Problem Statement

Modern applications often perform tasks like sending emails, generating PDFs, or resizing images directly inside API calls. This leads to:

* Slow response times ❌
* Blocking operations ❌
* Poor user experience ❌

This project solves it using **asynchronous background processing**.

---

## ⚡ How It Works

```
User Request
    ↓
API responds instantly (<5ms) ✅
    ↓
Task pushed to Redis Queue
    ↓
Worker processes task in background
    ↓
Task completed ✅
```

---

## 🏗️ Architecture

```
Frontend (Vercel)
        ↓
Producer API (Render)
        ↓
Redis Queue (Upstash)
        ↓
Worker(s) (Local / Cloud)
```

---

## ✨ Features

* ⚡ Instant API response (non-blocking)
* 🔀 Concurrent execution using goroutines
* 🔁 Retry mechanism for failed jobs
* 📊 Metrics endpoint for monitoring
* 🧩 Flexible task payload system
* 🌐 Fully deployed distributed architecture
* 🧪 cURL & UI based testing

---

## ⚙️ Tech Stack

| Layer      | Technology                          |
| ---------- | ----------------------------------- |
| Backend    | Go (Golang)                         |
| Queue      | Redis (Upstash)                     |
| API        | REST                                |
| Frontend   | HTML, CSS, JavaScript               |
| Deployment | Render (Backend), Vercel (Frontend) |

---

## 📡 API Reference

### 🔹 Enqueue Task

```
POST /enqueue
```

👉 Live:
https://workqueue-producer-kz2k.onrender.com/enqueue

---

### Request Example

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

---

### Response

```
Task of type 'send_email' has been successfully added to the queue
```

---

## 📊 Metrics Endpoint (Worker)

```
GET /metrics
```

> ⚠️ Available only when worker is running

---

## 🧩 Supported Task Types

| Task         | Payload Fields |
| ------------ | -------------- |
| send_email   | to, subject    |
| resize_image | new_x, new_y   |
| generate_pdf | optional       |

---

## 🚀 Run Locally

### 1. Clone Repo

```bash
git clone https://github.com/thisisanubhav/distributed-task-processing-system.git
cd distributed-task-processing-system
```

---

### 2. Create Config File

Create `config.env`

```env
REDIS_URL=redis://localhost:6379
PORT_PRODUCER=8080
PORT_WORKER=8081
```

---

### 3. Start Redis

```bash
brew services start redis
```

---

### 4. Run Producer

```bash
go run cmd/producer/main.go
```

---

### 5. Run Worker

```bash
go run cmd/worker/worker_main_cors.go
```

---

### 6. Open UI

```bash
open frontend/index.html
```

---

## 🧪 Test with cURL

```bash
curl -X POST https://workqueue-producer-kz2k.onrender.com/enqueue \
-H "Content-Type: application/json" \
-d '{
  "type": "send_email",
  "payload": {
    "to": "test@gmail.com",
    "subject": "Hello from cloud 🚀"
  },
  "retries": 2
}'
```

---

## 📁 Project Structure

```
distributed-task-processing-system/
├── cmd/
│   ├── producer/
│   └── worker/
├── internal/
│   ├── task/
│   ├── worker/
│   └── logger/
├── frontend/
│   └── index.html
├── config.env
├── go.mod
└── README.md
```

---

## 🔑 Key Concepts

* Producer–Consumer Model
* Asynchronous Processing
* Goroutines (Concurrency)
* Redis Queue (RPUSH, BLPOP)
* Retry Handling
* Distributed System Design

---

## 🧠 Design Decisions

* Redis Queue → Simple and fast
* BLPOP → Efficient blocking consumption
* Goroutines → Lightweight concurrency
* Service separation → Scalable system

---

## ⚠️ Notes

* Worker currently runs locally (can be deployed later)
* Metrics depend on worker availability
* Redis is hosted on Upstash

---

## 🔥 Future Improvements

* Deploy worker on cloud
* Add authentication
* Dead-letter queue
* Priority queues
* Real-time updates

---

## 👨‍💻 Developed By

**Anubhav Harsh Sinha**

---

## ⭐ If you like this project

Give it a ⭐ on GitHub!
