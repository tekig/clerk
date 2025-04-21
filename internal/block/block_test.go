package block

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
)

func Test(t *testing.T) {
	if err := test(); err != nil {
		t.Error(err)
	}
}

func test() error {
	if err := os.Mkdir("tmp", os.ModePerm); err != nil {
		return fmt.Errorf("tmp: %w", err)
	}

	block, err := NewBlock("tmp", MaxChunkSize(5*1024*1024))
	if err != nil {
		return fmt.Errorf("block: %w", err)
	}

	firstid := uuid.New()
	if _, err := block.Write(&pb.Event{
		Id: firstid[:],
		Attributes: []*pb.Attribute{
			{
				Key: "key",
				Value: &pb.Attribute_AsInt64{
					AsInt64: int64(123),
				},
			}, {
				Key: "text",
				Value: &pb.Attribute_AsString{
					AsString: "Хороший вопрос! Название структуры зависит от контекста (язык программирования, область применения), но вот несколько универсальных вариантов:",
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	for i := range 100000 {
		id := uuid.New()
		if _, err := block.Write(&pb.Event{
			Id: id[:],
			Attributes: []*pb.Attribute{
				{
					Key: "key",
					Value: &pb.Attribute_AsInt64{
						AsInt64: int64(i),
					},
				}, {
					Key: "text",
					Value: &pb.Attribute_AsString{
						AsString: "Хороший вопрос! Название структуры зависит от контекста (язык программирования, область применения), но вот несколько универсальных вариантов:",
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("write event: %w", err)
		}
	}

	{
		event, err := block.Search(context.Background(), firstid)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		_ = event
	}
	{
		// force compress all data
		if err := block.chunk.compress.Flush(); err != nil {
			return fmt.Errorf("flush: %w", err)
		}

		var randID uuid.UUID
		for k := range block.index.chunkIDs {
			randID = k
			break
		}

		event, err := block.Search(context.Background(), randID)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		_ = event
	}

	if err := block.Close(); err != nil {
		return fmt.Errorf("close block")
	}

	return nil
}
