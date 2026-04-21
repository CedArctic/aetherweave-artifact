package sync

import (
	"bytes"
	"context"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p"
	p2ptypes "github.com/OffchainLabs/prysm/v6/beacon-chain/p2p/types"
	pb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	snarktypes "github.com/iden3/go-rapidsnark/types"
)

// sendRPCAWRequest sends an AetherWeave request to a peer.
func (s *Service) sendRPCAWRequest(ctx context.Context, request *pb.Request, peerID peer.ID, nonces []Nonce) error {
	// Start a context with a timeout
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	// Stream topic string
	topic := p2p.RPCAWHeartbeatV1

	// Open stream, send request and defer closing the stream
	stream, err := s.cfg.p2p.Send(ctx, request, topic, peerID)
	if err != nil {
		s.aw.table.records_m.RLock()
		defer s.aw.table.records_m.RUnlock()
		log.WithError(err).WithFields(logrus.Fields{"peerID": peerID, "NetworkRecord": s.aw.table.records[PublicKeyHash(peerID)].GetNetRecord()}).Error("Failed to send Request")
		return err
	}
	defer closeStream(stream, log)

	// Read status code in the response
	code, errMsg, err := ReadStatusCode(stream, s.cfg.p2p.Encoding())
	if err != nil {
		log.WithError(err).Error("Aetherweave peer responded with error. Not increasing bad response counter.")
		// TODO: Re-enable this for production
		// s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		return err
	}
	if code == responseCodeResourceUnavailable {
		log.WithError(err).WithField("code", code).Error("Aetherweave peer rate-limited our request.")
		return errors.New(errMsg)
	}
	if code != 0 {
		log.WithError(err).WithField("code", code).Error("Aetherweave peer responded with non zero status code. Not increasing bad response counter.")
		// TODO: Re-enable this for production
		// s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(peerID)
		return errors.New(errMsg)
	}

	// Decode response
	msg := &pb.Response{}
	if err := s.cfg.p2p.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
		log.Println("failed to decode AetherWeave response:", err)
		return err
	}

	log.WithFields(logrus.Fields{"peerID": peerID, "records": len(msg.GetRecords()) + len(msg.GetRecordsB())}).Info("Received Response to Aetherweave Request")

	// Process received PeerRecords and get any SlashProofs that occur from Commitment collisions
	msg_records := msg.GetRecords()
	builtSlashProofs := make([]*pb.SlashProof, 0)
	for _, peer_record := range msg_records {
		// For every record, unmarshal its public key
		pubkey, aw_peerID, err := processMarshalledPubkey(peer_record.GetNetRecord().GetPublicKey().GetPubkey())
		if err != nil {
			log.WithError(err).Error("Failed to get aw_peerID from public key")
			continue
		}
		// Process the peer record
		slashproof, err := s.aw.processPeerRecord(peer_record, nonces[0], pubkey, aw_peerID, false, true)
		if err != nil {
			log.WithError(err).Error("Failed to process peer record. Skipping.")
			continue
		}
		if slashproof != nil {
			builtSlashProofs = append(builtSlashProofs, slashproof)
		}
	}
	if DUAL_TABLES && len(nonces) == 2 && len(msg.GetRecordsB()) > 0 {
		for _, peer_record := range msg.GetRecordsB() {
			// For every record, unmarshal its public key
			pubkey, aw_peerID, err := processMarshalledPubkey(peer_record.GetNetRecord().GetPublicKey().GetPubkey())
			if err != nil {
				log.WithError(err).Error("Failed to get aw_peerID from public key")
				continue
			}
			slashproof, err := s.aw.processPeerRecord(peer_record, nonces[1], pubkey, aw_peerID, false, false)
			if err != nil {
				log.WithError(err).Error("Failed to process peer record. Skipping.")
				continue
			}
			if slashproof != nil {
				builtSlashProofs = append(builtSlashProofs, slashproof)
			}
		}
	}

	// Submit newly built SlashProofs
	if len(builtSlashProofs) > 0 {
		log.WithField("slash_proofs_count", len(builtSlashProofs)).Info("Submitting slashproofs")
		if err := submitSlashProofs(s.aw.ethClient, s.aw.contract, builtSlashProofs, s.aw.eth_pubkey, s.aw.eth_privkey); err != nil {
			log.WithError(err).Error("Error submitting slashproofs")
		}
	}

	// Process received SlashProofs
	s.aw.table.processSlashProofs(msg.GetSlashProofs(), s.aw.sh_vkey_bytes)

	return nil

}

// aw_requestRPCHandler handles incoming AetherWeave Requests
func (s *Service) aw_requestRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {

	log := log.WithField("handler", "status")

	// Set connection timeouts and deadlines
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	// log.Info("Setting RPC Stream Deadline")
	SetRPCStreamDeadlines(stream)

	// Parse Request
	request, ok := msg.(*pb.Request)
	if !ok {
		err := errors.New("message is not type *pb.Request")
		log.WithError(err).Error("Failed to parse to *pb.Request")
		return err
	}

	// Validate request and apply rate limit
	// TODO: Re-enable rate limiting. Temporarily disabled for debugging
	// log.WithField("SenderRecord", request.SenderRecord).Info("aw_requestRPCHandler limiter: skipping adding stream to rate limiter")
	// if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
	// 	log.WithError(err).Error("Rate limit error during validateRequest()")
	// 	return err
	// }
	// s.rateLimiter.add(stream, 1)

	// Get sender peerID
	remotePeer := stream.Conn().RemotePeer()
	log.WithFields(logrus.Fields{"peerID": remotePeer, "nonces": request.GetNonces()}).Info("Received Request")

	// Get sender Aetherweave peerID and pubkey
	pubkey, aw_peerID, err := processMarshalledPubkey(request.GetSenderRecord().GetPublicKey().GetPubkey())
	if err != nil {
		log.WithError(err).Error("Failed to get aw_peerID from public key")
		return err
	}

	// Validate request
	if err := s.aw.validateRequest(ctx, request, remotePeer, pubkey, aw_peerID); err != nil {

		var respCode byte
		switch {
		case errors.Is(err, p2ptypes.ErrGeneric):
			respCode = responseCodeServerError
		case errors.Is(err, p2ptypes.ErrCommitmentServed):
			log.WithError(err).Warn("Refusing request. Commitment has already been served.")
			respCode = responseCodeResourceUnavailable
		default:
			respCode = responseCodeInvalidRequest
			log.WithError(err).Warn("Was supposed to increase bad peer score. Skipping for now.")
			// TODO: Re-enable this after debugging. Currently disabled to debug disconnections
			// s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(remotePeer)
		}

		if !errors.Is(err, p2ptypes.ErrCommitmentServed) {
			log.WithFields(logrus.Fields{
				"peer":  remotePeer,
				"error": err,
			}).Error("Request validation error")
		}

		originalErr := err
		resp, err := s.generateErrorResponse(respCode, err.Error())
		if err != nil {
			log.WithError(err).Debug("Could not generate a response error")
		} else if _, err := stream.Write(resp); err != nil && !isUnwantedError(err) {
			// The peer may already be ignoring us, as we disagree on fork version, so log this as debug only.
			log.WithError(err).Debug("Could not write to stream")
		}
		closeStreamAndWait(stream, log)
		return originalErr
	}
	log.WithField("remotePeer", remotePeer).Info("Request passed validation")

	// Validate and attempt to inject the sender's record
	peer_rec := &pb.PeerRecord{NetRecord: request.GetSenderRecord(), Commitments: []*pb.CommitmentRecord{request.GetCommitmentRecord()}}

	req_slashProof, err := s.aw.processPeerRecord(peer_rec, s.aw.table.nonce_pub, pubkey, aw_peerID, true, true)
	if err != nil {
		resp, err := s.generateErrorResponse(responseCodeInvalidRequest, err.Error())
		if err != nil {
			log.WithError(err).Debug("Could not generate a response error")
		} else if _, err := stream.Write(resp); err != nil && !isUnwantedError(err) {
			log.WithError(err).Debug("Could not write to stream")
		}
		closeStreamAndWait(stream, log)
		return errors.New("Remote peer network record failed validation during injection.")
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil && !isUnwantedError(err) {
		log.WithError(err).Debug("Could not write response code to stream")
	}

	// Build response records
	responseRecords := []*pb.PeerRecord{}
	responseRecords_B := []*pb.PeerRecord{}

	s.aw.table.records_m.RLock()
	for aw_peerID, _ := range s.aw.table.idx_pub {
		peerRecord, ok := s.aw.table.records[aw_peerID]
		if !ok {
			log.WithFields(logrus.Fields{"aw_peerID": aw_peerID}).Warn("Index aw_peerID not in records table. Skipping including it in Response.")
			continue
		}
		recScore := score(request.SenderRecord.PublicKey.Pubkey, peerRecord.NetRecord.PublicKey.Pubkey, Nonce(request.Nonces[0]))
		if float64(recScore) < float64(TABLE_SIZE)/float64(AW_NODES_NUM) {
			responseRecords = append(responseRecords, peerRecord)
		}
		if len(request.Nonces) > 1 && DUAL_TABLES {
			recScore_B := score(request.SenderRecord.PublicKey.Pubkey, peerRecord.NetRecord.PublicKey.Pubkey, Nonce(request.Nonces[1]))
			if float64(recScore_B) < float64(TABLE_SIZE)/float64(AW_NODES_NUM) {
				responseRecords_B = append(responseRecords_B, peerRecord)
			}
		}
	}
	s.aw.table.records_m.RUnlock()

	// Fetch most recent slashproofs from blacklist table
	slashProofs := []*pb.SlashProof{}
	blacklist := []BlacklistEntry{}
	for _, entry := range s.aw.table.blacklist {
		blacklist = append(blacklist, entry)
	}
	sort.Slice(blacklist, func(i, j int) bool { return blacklist[i].timestamp.After(blacklist[j].timestamp) })
	for _, entry := range blacklist[:min(len(blacklist), MAX_RES_SLASHPROOFS)] {
		slashProofs = append(slashProofs, entry.slash_proof)
	}

	// Build response
	response := &pb.Response{
		Records:     responseRecords,
		RecordsB:    responseRecords_B,
		SlashProofs: slashProofs,
	}

	// Send response and close stream
	if _, err := s.cfg.p2p.Encoding().EncodeWithMaxLength(stream, response); err != nil && !isUnwantedError(err) {
		log.WithError(err).Debug("Could not write Response to stream")
		return err
	}
	closeStream(stream, log)

	// Keep track of the Commitments for which we have served a Response
	comm_root_b := request.GetCommitmentRecord().GetSlashShare()
	var comm_root Hash
	copy(comm_root[:], comm_root_b)
	s.aw.table.markCommitmentServed(comm_root, s.aw.round_number)

	// Submit any SlashProofs that occur from the new peer
	if req_slashProof != nil {
		if err := submitSlashProofs(s.aw.ethClient, s.aw.contract, []*pb.SlashProof{req_slashProof}, s.aw.eth_pubkey, s.aw.eth_privkey); err != nil {
			log.Error("Failed to submit SlashProofs while responding to request")
		}
	}

	log.WithField("peerID", remotePeer).Info("Processed Request")

	return nil
}

func (aw *Aetherweave) validateRequest(ctx context.Context, req *pb.Request, streamPeerID peer.ID, recordPubKey crypto.PubKey, recordPeerID peer.ID) error {
	if req == nil || req.SenderRecord == nil || req.CommitmentOpening == nil || req.CommitmentRecord == nil {
		return errors.New("incomplete request")
	}

	// Check if we've served a Request with this Commitment root before
	slash_share_b := req.GetCommitmentRecord().GetRootHash().GetHash()
	var slash_share Hash
	copy(slash_share[:], slash_share_b)
	if aw.table.checkCommitmentServed(slash_share) {
		return p2ptypes.ErrCommitmentServed
	}

	// Extract sender NetRecord and public key, and calculate peerID
	netRecord := req.SenderRecord

	// Assert that pubkey is BJJ key
	bjj_pubkey, ok := recordPubKey.(*crypto.BJJPublicKey)
	if !ok {
		return errors.New("Failed to assert BabyJubJub public key")
	}
	pubkey_x, pubkey_y := bjj_pubkey.GetXY()

	// Verify that the peerID from the libp2p stream matches that of the Request
	// TODO: disabled for now since we're using Babyjubjub for network records, but ECDSA for libp2p
	// if recordPeerID != streamPeerID {
	// 	return errors.New("stream peerID does not match sender record peerID")
	// }

	// Check if Request sender has been slashed
	if aw.table.isPeerBlacklisted(PublicKeyHash(recordPeerID)) {
		return errors.New("Request sender is slashed, discarding Request")
	}

	// Verify that CommitmentOpening matches the CommitmentRecord
	if !bytes.Equal(req.CommitmentRecord.RootHash.Hash, req.CommitmentOpening.ParentHash.Hash) {
		return errors.New("CommitmentRecord hash does not match CommitmentOpening")
	}

	// Verify CommitmentRecord
	commitment := req.CommitmentRecord
	if commitment.RootHash == nil {
		return errors.New("invalid commitment record")
	}

	// Check epoch freshness
	if commitment.RoundNumber < uint64(aw.round_number)-1 || commitment.RoundNumber > uint64(aw.round_number)+1 {
		return errors.New("request commitment has invalid round number")
	}

	// Verify that the peer has included our publickey in its commitment
	ok, err := verify_commitment_opening(req.CommitmentOpening, aw.node_pubkey)
	if err != nil || !ok {
		return errors.Wrap(err, "invalid commitment opening")
	}

	// Verify CommitmentRecord, including ZKP

	// Unpack share proof data
	zkp_d, err := ZKPToProofData(commitment.ShareProof)
	if err != nil {
		return errors.Wrap(err, "failed to convert share proof ZKP protobuf to ProofData ZKP")
	}

	// Reconstruct public signals from the protobufs what we received
	// Public signals are decimal strings of [ pubkeyX, pubkeyY, commitment_root, epoch, slashshare ]
	commitment_root_bi := new(big.Int).SetBytes(commitment.RootHash.Hash)
	share := new(big.Int).SetBytes(commitment.SlashShare)
	zkp_pub := []string{pubkey_x.String(), pubkey_y.String(), commitment_root_bi.String(), strconv.FormatUint(commitment.RoundNumber, 10), share.String()}
	zkp := snarktypes.ZKProof{Proof: zkp_d, PubSignals: zkp_pub}

	// Check share proof
	err = verifyZKProof(zkp, aw.sh_vkey_bytes)
	if err != nil {
		return errors.Wrap(err, "share proof verification failed")
	}

	// Check freshness
	timestamp := int64(netRecord.Timestamp)
	now := time.Now().Unix()
	if now < timestamp || now-timestamp > AW_ROUND_TIME {
		return errors.New("request too old or from the future")
	}

	// Check that the sender has not created requests to more than the allowed peers in the given round
	if len(req.CommitmentOpening.Proof) > int(math.Ceil(math.Log2(float64(AW_REQ_NUM)))) {
		return errors.Errorf("commitment proof too deep: %d > ceil(log2(%d))", len(req.CommitmentOpening.Proof), AW_REQ_NUM)
	}

	return nil
}

// Validate a NetworkRecord by ensuring a proper sc_root, valid PoS commitment and sender signature.
func validateNetworkRecord(n *pb.NetworkRecord, sc_roots *SCRootsTable, round_n RoundNumber, st_vkey_bytes []byte) (bool, error) {
	if n == nil || n.PublicKey == nil || n.ProofOfStake == nil || n.Multiaddr == nil || n.Signature == nil {
		return false, errors.New("incomplete NetworkRecord")
	}

	// Unmarshal public key
	pubkey, err := crypto.UnmarshalPublicKey(n.PublicKey.Pubkey)
	if err != nil {
		return false, errors.Wrap(err, "invalid public key in NetworkRecord")
	}

	// Assert that pubkey is a BJJ key
	bjj_pubkey, ok := pubkey.(*crypto.BJJPublicKey)
	if !ok {
		return false, errors.New("Failed to assert BabyJubJub public key")
	}
	pubkey_x, pubkey_y := bjj_pubkey.GetXY()

	// Unpack proof of stake proof data
	// zkp_d := snarktypes.ProofData{}
	// err = json.Unmarshal(n.ProofOfStake, &zkp_d)
	// if err != nil {
	// 	return false, errors.Wrap(err, "failed to unmarshal proof of stake data")
	// }
	zkp_d, err := ZKPToProofData(n.ProofOfStake)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert proof of stake ZKP protobuf to ProofData ZKP")
	}

	// Get proof Merkle root in hex
	merkle_root := n.MerkleRoot
	merkle_root_bi := new(big.Int).SetBytes(merkle_root.Hash)

	// Verify that the smart contract root is valid
	sc_root_valid := sc_roots.scRootValid(Hash(merkle_root.Hash), round_n)
	if !sc_root_valid {
		return false, errors.New("Smart contract root for proof of stake is not valid")
	}

	// Reconstruct public signals from the protobufs what we received
	// Public signals are decimal strings of [ pubkeyX, pubkeyY, merkle root ]
	zkp_pub := []string{pubkey_x.String(), pubkey_y.String(), merkle_root_bi.String()}
	zkp := snarktypes.ZKProof{Proof: zkp_d, PubSignals: zkp_pub}

	// Check proof of stake
	err = verifyZKProof(zkp, st_vkey_bytes)
	if err != nil {
		return false, errors.Wrap(err, "proof of stake verification failed")
	}

	// Verify NetworkRecord signature
	ok, err = verifyAWMessage(n, pubkey)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("invalid NetworkRecord signature")
	}

	return true, nil
}
