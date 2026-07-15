// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
)

func streamSSE(ctx context.Context, body io.Reader, handle func(data []byte) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(line[6:])
		if data == "[DONE]" {
			return nil
		}
		if err := handle([]byte(data)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func streamNDJSON(ctx context.Context, body io.Reader, handle func(data []byte) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := handle([]byte(line)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func emitStream(ctx context.Context, chunks chan<- ChatCompletionChunk, errs chan<- error, fn func() error) {
	go func() {
		defer close(chunks)
		defer close(errs)
		if err := fn(); err != nil {
			select {
			case errs <- err:
			case <-ctx.Done():
			}
		}
	}()
}

func parseChunkJSON(data []byte, chunk *ChatCompletionChunk) error {
	return json.Unmarshal(data, chunk)
}
