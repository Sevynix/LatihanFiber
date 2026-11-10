package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"

	"tugas2/app/model"
	"tugas2/config"
	"tugas2/database"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "id baris outbox")
	copies := flag.Int("copies", 5, "jumlah salinan task")
	flag.Parse()
	if *outboxID <= 0 || *copies <= 0 {
		fmt.Fprintln(os.Stderr, "pemakaian: go run ./tools/redeliver -outbox <id> -copies <n>")
		os.Exit(2)
	}

	config.LoadEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx,
		`SELECT event_type, payload FROM outbox WHERE id = $1`, *outboxID).
		Scan(&eventType, &payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "membaca outbox:", err)
		os.Exit(1)
	}

	body, err := json.Marshal(model.NotificationJob{
		OutboxID: *outboxID, EventType: eventType, Payload: payload,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "menyusun task:", err)
		os.Exit(1)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	for i := 1; i <= *copies; i++ {
		task := asynq.NewTask(model.TaskSendNotification, body,
			asynq.Queue(model.QueueNotifications), asynq.MaxRetry(5))
		if _, err := client.EnqueueContext(ctx, task); err != nil {
			fmt.Fprintln(os.Stderr, "enqueue:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("%d salinan task untuk outbox %d dikirim ke Redis\n", *copies, *outboxID)
}
