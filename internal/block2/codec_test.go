package block2

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/tekig/clerk/internal/pb"
	"google.golang.org/protobuf/proto"
)

func TestEncodeDecode(t *testing.T) {
	messages := []*pb.Event{
		{Id: []byte("1"), Attributes: []*pb.Attribute{{Key: "k", Value: &pb.Attribute_AsString{AsString: "v"}}}},
		{Id: []byte("2"), Attributes: []*pb.Attribute{{Key: "k", Value: &pb.Attribute_AsInt64{AsInt64: 42}}}},
		{Id: []byte("3")},
	}

	var buf bytes.Buffer
	for _, m := range messages {
		if err := Encode(m, &buf); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}

	for _, want := range messages {
		got := &pb.Event{}
		if err := Decode(got, &buf); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !proto.Equal(got, want) {
			t.Errorf("decode = %v, want %v", got, want)
		}
	}

	if err := Decode(&pb.Event{}, &buf); !errors.Is(err, io.EOF) {
		t.Errorf("decode after last message: err = %v, want io.EOF", err)
	}
}

func TestDecode_Truncated(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&pb.Event{Id: []byte("0123456789")}, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()
	for _, size := range []int{4, 8, len(data) - 1} {
		err := Decode(&pb.Event{}, bytes.NewReader(data[:size]))
		if err == nil {
			t.Errorf("decode of %d/%d bytes: expected error", size, len(data))
		}
	}
}
