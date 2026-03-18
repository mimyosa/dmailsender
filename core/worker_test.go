package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartSendCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mail := MailConfig{
		MailFrom:     "test@example.com",
		RcptTo:       "dest@example.com",
		Subject:      "Test",
		Body:         "body",
		ContentType:  "text/plain",
		MailNumber:   1000,
		ThreadNumber: 1,
		IntervalMs:   10,
	}

	server := ServerConfig{
		SMTP: "localhost",
		Port: 25,
	}

	var resultCount int64

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	StartSend(ctx, server, "", mail, nil,
		nil,
		func(r SendResult) {
			atomic.AddInt64(&resultCount, 1)
		},
		nil,
	)

	count := atomic.LoadInt64(&resultCount)
	if count >= int64(mail.MailNumber) {
		t.Errorf("expected cancellation to stop early, but got %d results out of %d", count, mail.MailNumber)
	}
}

func TestStartSendCallbackCounts(t *testing.T) {
	ctx := context.Background()

	mail := MailConfig{
		MailFrom:     "test@example.com",
		RcptTo:       "dest@example.com",
		Subject:      "Test",
		Body:         "body",
		ContentType:  "text/plain",
		MailNumber:   3,
		ThreadNumber: 1,
		IntervalMs:   0,
	}

	server := ServerConfig{
		SMTP: "localhost",
		Port: 25,
	}

	var resultCount int64
	var progressCount int64

	StartSend(ctx, server, "", mail, nil,
		func(p ProgressEvent) {
			atomic.AddInt64(&progressCount, 1)
		},
		func(r SendResult) {
			atomic.AddInt64(&resultCount, 1)
		},
		nil,
	)

	rc := atomic.LoadInt64(&resultCount)
	pc := atomic.LoadInt64(&progressCount)

	if rc != 3 {
		t.Errorf("expected 3 results, got %d", rc)
	}
	if pc != 3 {
		t.Errorf("expected 3 progress events, got %d", pc)
	}
}
