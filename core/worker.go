package core

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// StartSend runs the mail send loop with concurrency control.
// It calls onProgress and onResult callbacks for each completed send.
// The caller can cancel via ctx to stop early.
func StartSend(
	ctx context.Context,
	server ServerConfig,
	password string,
	mail MailConfig,
	attachments []string,
	onProgress func(ProgressEvent),
	onResult func(SendResult),
	onLog func(direction, line string),
) {
	sem := make(chan struct{}, mail.ThreadNumber)
	var wg sync.WaitGroup

	var sent int64
	var failed int64
	total := mail.MailNumber

	for i := 0; i < mail.MailNumber; i++ {
		// Check cancellation before acquiring semaphore
		select {
		case <-ctx.Done():
			break
		default:
		}

		// Check again after select — break only exits select, not for
		if ctx.Err() != nil {
			break
		}

		// Acquire semaphore
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			// Compute actual from/to/subject for this index
			from := mail.MailFrom
			if mail.NumberingMailFrom {
				from = applyNumbering(from, idx)
			}
			rcpt := mail.RcptTo
			if mail.NumberingRcptTo {
				rcpt = applyNumbering(rcpt, idx)
			}
			subject := mail.Subject
			if mail.NumberingSubject {
				subject = applyNumberingSubject(subject, idx)
			}

			err := SendOne(server, password, mail, idx, attachments, onLog)
			result := SendResult{
				Index:   idx,
				Success: err == nil,
				From:    from,
				To:      rcpt,
				Subject: subject,
			}
			if err != nil {
				result.Error = err.Error()
				atomic.AddInt64(&failed, 1)
			} else {
				atomic.AddInt64(&sent, 1)
			}

			if onResult != nil {
				onResult(result)
			}

			if onProgress != nil {
				onProgress(ProgressEvent{
					Sent:   int(atomic.LoadInt64(&sent)),
					Failed: int(atomic.LoadInt64(&failed)),
					Total:  total,
				})
			}

			if mail.IntervalMs > 0 {
				time.Sleep(time.Duration(mail.IntervalMs) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}

// StartSendEML sends .eml files with concurrency control.
// mailNumber controls how many times the files are sent (cycling through emlFiles).
func StartSendEML(
	ctx context.Context,
	server ServerConfig,
	password string,
	emlFiles []string,
	from, rcpt string,
	numberingFrom, numberingTo bool,
	useHeaderEnvelope bool,
	updateMessageID bool,
	customHeaders []Header,
	mailNumber int,
	threadNumber int,
	intervalMs int,
	onProgress func(ProgressEvent),
	onResult func(SendResult),
	onLog func(direction, line string),
) {
	if mailNumber <= 0 {
		mailNumber = len(emlFiles)
	}
	if len(emlFiles) == 0 {
		return
	}

	if onLog != nil {
		onLog("info", fmt.Sprintf("EML Send starting: count=%d, files=%d, threads=%d", mailNumber, len(emlFiles), threadNumber))
	}

	sem := make(chan struct{}, threadNumber)
	var wg sync.WaitGroup

	var sent int64
	var failed int64
	total := mailNumber

	for i := 0; i < mailNumber; i++ {
		select {
		case <-ctx.Done():
		default:
		}
		if ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
		case sem <- struct{}{}:
		}
		if ctx.Err() != nil {
			break
		}

		emlPath := emlFiles[i%len(emlFiles)]

		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Apply numbering for this iteration
			actualFrom := from
			if numberingFrom && from != "" {
				actualFrom = applyNumbering(from, idx)
			}
			actualRcpt := rcpt
			if numberingTo && rcpt != "" {
				actualRcpt = applyNumbering(rcpt, idx)
			}

			err := SendEML(server, password, actualFrom, actualRcpt, path, useHeaderEnvelope, updateMessageID, customHeaders, onLog)
			result := SendResult{
				Index:   idx,
				Success: err == nil,
				From:    actualFrom,
				To:      actualRcpt,
				Subject: filepath.Base(path),
			}
			if err != nil {
				result.Error = err.Error()
				atomic.AddInt64(&failed, 1)
			} else {
				atomic.AddInt64(&sent, 1)
			}

			if onResult != nil {
				onResult(result)
			}
			if onProgress != nil {
				onProgress(ProgressEvent{
					Sent:   int(atomic.LoadInt64(&sent)),
					Failed: int(atomic.LoadInt64(&failed)),
					Total:  total,
				})
			}

			if intervalMs > 0 {
				time.Sleep(time.Duration(intervalMs) * time.Millisecond)
			}
		}(i, emlPath)
	}

	wg.Wait()
}
