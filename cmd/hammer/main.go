package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app(ctx); err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
}

func app(ctx context.Context) error {
	host := flag.String("host", "localhost:50051", "target recorder")
	concurrency := flag.String("concurrency", "1", "number of parallel records")
	flag.Parse()

	conc, err := strconv.Atoi(*concurrency)
	if err != nil {
		return fmt.Errorf("atoi concurrency: %w", err)
	}

	_ = conc

	h := New(*host)

	var (
		wg       sync.WaitGroup
		onceErr  sync.Once
		causeErr error
	)
	for range conc {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := h.hammer(ctx); err != nil {
				onceErr.Do(func() {
					causeErr = err
				})
			}
		}()
	}

	wg.Wait()

	return causeErr
}

type Hammer struct {
	host  string
	total uint64
	batch time.Time
	mu    sync.Mutex
}

func New(host string) *Hammer {
	return &Hammer{
		host:  host,
		batch: time.Now(),
	}
}

func (h *Hammer) hammer(ctx context.Context) error {
	conn, err := grpc.NewClient(
		h.host,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	defer conn.Close()

	client := pb.NewRecorderClient(conn)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		id := uuid.New()
		b := make([]byte, 4*1024)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("rand: %w", err)
		}
		str := hex.EncodeToString(b)

		if _, err := client.CreateEvents(ctx, &pb.CreateEventsRequest{
			Events: []*pb.Event{
				{
					Id: id[:],
					Attributes: []*pb.Attribute{
						{
							Key: "string",
							Value: &pb.Attribute_AsString{
								AsString: str,
							},
						}, {
							Key: "int",
							Value: &pb.Attribute_AsInt64{
								AsInt64: rand.Int63(),
							},
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("create events: %w", err)
		}

		n := atomic.AddUint64(&h.total, 1)
		if n%1000 == 0 {
			h.mu.Lock()

			d := time.Since(h.batch)

			log.Printf("lastBlock=%s (%s), avg=%s, rpc=%f", base64.RawStdEncoding.EncodeToString(id[:]), id.String(), d/time.Duration(1000), 1000/d.Seconds())

			h.batch = time.Now()

			h.mu.Unlock()
		}
	}
}
