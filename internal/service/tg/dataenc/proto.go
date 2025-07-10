package dataenc

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

func Compress(
	b []byte,
) []byte {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, err := zw.Write(b)
	if err != nil {
		panic(err) // we are in trouble
	}
	zw.Flush()
	zw.Close()
	return buf.Bytes()
}

func DecompressString(
	str string,
) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("b64 decode: %w", err)
	}

	return DecompressReader(bytes.NewReader(b))
}

func CompressToString(
	b []byte,
) string {
	return base64.RawURLEncoding.EncodeToString(Compress(b))
}

func DecompressReader(
	r io.Reader,
) ([]byte, error) {
	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("new zlib reader: %w", err)
	}

	defer rc.Close()

	decompressed, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	return decompressed, nil
}

func Decompress(
	b []byte,
) ([]byte, error) {
	return DecompressReader(bytes.NewReader(b))
}

func Encode(msg proto.Message) []byte {
	bb, err := proto.Marshal(msg)
	if err != nil {
		panic(err) // we are in trouble
	}

	return bb
}

func EncodeToString(msg proto.Message) string {
	return base64.RawURLEncoding.EncodeToString(Encode(msg))
}

func DecodeString(str string, msg proto.Message) error {
	b, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("b64 decode: %w", err)
	}

	return proto.Unmarshal(b, msg)
}

func Decode(b []byte, msg proto.Message) error {
	return proto.Unmarshal(b, msg)
}
