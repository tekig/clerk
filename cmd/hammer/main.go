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
	"time"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app(ctx); err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
}

func app(ctx context.Context) error {
	host := flag.String("host", "localhost:50051", "target recorder")
	flag.Parse()

	conn, err := grpc.NewClient(
		*host,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	defer conn.Close()

	client := pb.NewRecorderClient(conn)

	t1 := time.Now()
	n := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		id := uuid.New()
		var b = make([]byte, 4*1024)
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

		n++
		if n%1000 == 0 {
			d := time.Since(t1)

			log.Printf("lastBlock=%s (%s), avg=%s, rpc=%f", base64.RawStdEncoding.EncodeToString(id[:]), id.String(), d/time.Duration(1000), 1000/d.Seconds())

			t1 = time.Now()
		}
	}
}
