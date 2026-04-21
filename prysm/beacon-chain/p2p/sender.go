package p2p

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v6/monitoring/tracing"
	"github.com/OffchainLabs/prysm/v6/monitoring/tracing/trace"
	"github.com/kr/pretty"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/sirupsen/logrus"
)

// Send a message to a specific peer. The returned stream may be used for reading, but has been
// closed for writing.
//
// When done, the caller must Close or Reset on the stream.
func (s *Service) Send(ctx context.Context, message interface{}, baseTopic string, pid peer.ID) (network.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "p2p.Send")
	defer span.End()
	if err := VerifyTopicMapping(baseTopic, message); err != nil {
		wr_err := fmt.Errorf("failed to verify topic mapping. pid=%v. err=%w", pid, err)
		return nil, wr_err
	}
	topic := baseTopic + s.Encoding().ProtocolSuffix()
	span.SetAttributes(trace.StringAttribute("topic", topic))

	log.WithFields(logrus.Fields{
		"topic":   topic,
		"request": pretty.Sprint(message),
	}).Tracef("Sending RPC request to peer %s", pid.String())

	// Apply max dial timeout when opening a new stream.
	ctx, cancel := context.WithTimeout(ctx, maxDialTimeout)
	defer cancel()

	stream, err := s.host.NewStream(ctx, pid, protocol.ID(topic))
	if err != nil {
		wr_err := fmt.Errorf("failed to open new stream. pid=%v. err=%w", pid, err)
		tracing.AnnotateError(span, wr_err)
		return nil, wr_err
	}
	// do not encode anything if we are sending a metadata request
	if baseTopic != RPCMetaDataTopicV1 && baseTopic != RPCMetaDataTopicV2 {
		castedMsg, ok := message.(ssz.Marshaler)
		if !ok {
			return nil, errors.Errorf("%T does not support the ssz marshaller interface", message)
		}
		if _, err := s.Encoding().EncodeWithMaxLength(stream, castedMsg); err != nil {
			wr_err := fmt.Errorf("failed to encode message. pid=%v. err=%w", pid, err)
			tracing.AnnotateError(span, wr_err)
			_err := stream.Reset()
			_ = _err
			return nil, wr_err
		}
	}

	// Close stream for writing.
	if err := stream.CloseWrite(); err != nil {
		wr_err := fmt.Errorf("failed to close stream writer. pid=%v. err=%w", pid, err)
		tracing.AnnotateError(span, wr_err)
		_err := stream.Reset()
		_ = _err
		return nil, wr_err
	}

	return stream, nil
}
