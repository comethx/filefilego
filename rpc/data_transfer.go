package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/filefilego/filefilego/block"
	"github.com/filefilego/filefilego/common"
	"github.com/filefilego/filefilego/common/currency"
	"github.com/filefilego/filefilego/common/hexutil"
	"github.com/filefilego/filefilego/contract"
	ffgcrypto "github.com/filefilego/filefilego/crypto"
	"github.com/filefilego/filefilego/keystore"
	dataquery "github.com/filefilego/filefilego/node/protocols/data_query"
	dataverification "github.com/filefilego/filefilego/node/protocols/data_verification"
	"github.com/filefilego/filefilego/node/protocols/messages"
	"github.com/filefilego/filefilego/transaction"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// PublisherNodesFinder is an interface that specifies finding nodes and publishing a message to the network functionalities.
type PublisherNodesFinder interface {
	NetworkMessagePublisher
	FindPeers(ctx context.Context, peerIDs []peer.ID) []peer.AddrInfo
}

// DataTransferAPI represents the data transfer rpc service which includes data query and verification protocols.
type DataTransferAPI struct {
	host                     host.Host
	dataQueryProtocol        dataquery.Interface
	dataVerificationProtocol dataverification.Interface
	publisherNodesFinder     PublisherNodesFinder
	contractStore            contract.Interface
	keystore                 keystore.KeyAuthorizer
}

// NewDataTransferAPI creates a new data transfer API to be served using JSONRPC.
func NewDataTransferAPI(host host.Host, dataQueryProtocol dataquery.Interface, dataVerificationProtocol dataverification.Interface, publisherNodeFinder PublisherNodesFinder, contractStore contract.Interface, keystore keystore.KeyAuthorizer) (*DataTransferAPI, error) {
	if host == nil {
		return nil, errors.New("host is nil")
	}

	if dataQueryProtocol == nil {
		return nil, errors.New("data query protocol is nil")
	}

	if dataVerificationProtocol == nil {
		return nil, errors.New("data verification protocol is nil")
	}

	if publisherNodeFinder == nil {
		return nil, errors.New("publisherNodeFinder is nil")
	}

	if contractStore == nil {
		return nil, errors.New("contractStore is nil")
	}

	if keystore == nil {
		return nil, errors.New("keystore is nil")
	}

	return &DataTransferAPI{
		host:                     host,
		dataQueryProtocol:        dataQueryProtocol,
		dataVerificationProtocol: dataVerificationProtocol,
		publisherNodesFinder:     publisherNodeFinder,
		contractStore:            contractStore,
		keystore:                 keystore,
	}, nil
}

// SendDataQueryRequestArgs is a data query request argument.
type SendDataQueryRequestArgs struct {
	// FileHashes is a list of comma-separated file hashes.
	FileHashes string `json:"file_hashes"`
}

// SendDataQueryRequestResponse is a data query hash response.
type SendDataQueryRequestResponse struct {
	Hash string `json:"hash"`
}

// SendDataQueryRequest sends a data query request to the network.
func (api *DataTransferAPI) SendDataQueryRequest(r *http.Request, args *SendDataQueryRequestArgs, response *SendDataQueryRequestResponse) error {
	if args.FileHashes == "" {
		return errors.New("no file's in the request")
	}

	list := strings.Split(args.FileHashes, ",")
	request := messages.DataQueryRequest{
		FileHashes:   make([][]byte, 0),
		FromPeerAddr: api.host.ID().String(),
		Timestamp:    time.Now().Unix(),
	}

	for _, v := range list {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		fileHash, err := hexutil.DecodeNoPrefix(trimmed)
		if err != nil {
			return fmt.Errorf("failed to decode file hash: %w", err)
		}
		request.FileHashes = append(request.FileHashes, fileHash)
	}

	requestHash := request.GetHash()
	request.Hash = make([]byte, len(requestHash))
	copy(request.Hash, requestHash)

	err := request.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate data query request: %w", err)
	}

	requestHashHex := hexutil.Encode(requestHash)
	err = api.dataQueryProtocol.PutQueryHistory(requestHashHex, request)
	if err != nil {
		return fmt.Errorf("failed to insert data query request: %w", err)
	}

	requestProto := messages.ToDataQueryRequestProto(request)

	payload := messages.GossipPayload{
		Message: &messages.GossipPayload_Query{
			Query: requestProto,
		},
	}

	payloadBytes, err := proto.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("failed to marshal data query gossip payload: %w", err)
	}

	if err := api.publisherNodesFinder.PublishMessageToNetwork(r.Context(), payloadBytes); err != nil {
		return fmt.Errorf("failed to publish data query to network: %w", err)
	}

	response.Hash = requestHashHex

	return nil
}

// GetDownloadContractArgs represent the args.
type GetDownloadContractArgs struct {
	ContractHash string `json:"contract_hash"`
}

// DownloadContractJSON is a download contract in JSON.
type DownloadContractJSON struct {
	FileHosterResponse         DataQueryResponseJSON `json:"file_hoster_response"`
	FileRequesterNodePublicKey string                `json:"file_requester_node_public_key"`
	FileHashesNeeded           []string              `json:"file_hashes_needed"`
	FileHashesNeededSizes      []uint64              `json:"file_hashes_needed_sizes"`
	VerifierPublicKey          string                `json:"verifier_public_key"`
	VerifierFees               string                `json:"verifier_fees"`
	ContractHash               string                `json:"contract_hash"`
	VerifierSignature          string                `json:"verifier_signature"`
}

// GetDownloadContractResponse represents the response.
type GetDownloadContractResponse struct {
	Contract DownloadContractJSON `json:"contract"`
}

// GetDownloadContract returns a contract from the memmory.
func (api *DataTransferAPI) GetDownloadContract(r *http.Request, args *GetDownloadContractArgs, response *GetDownloadContractResponse) error {
	if args.ContractHash == "" {
		return errors.New("contract hash is empty")
	}

	downloadContract, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("failed to get contract: %w", err)
	}

	dqrJSON := DataQueryResponseJSON{
		FromPeerAddr:          downloadContract.FileHosterResponse.FromPeerAddr,
		FeesPerByte:           downloadContract.FileHosterResponse.FeesPerByte,
		HashDataQueryRequest:  hexutil.Encode(downloadContract.FileHosterResponse.HashDataQueryRequest),
		PublicKey:             hexutil.Encode(downloadContract.FileHosterResponse.PublicKey),
		Signature:             hexutil.Encode(downloadContract.FileHosterResponse.Signature),
		FileHashes:            make([]string, len(downloadContract.FileHosterResponse.FileHashes)),
		FileHashesSizes:       downloadContract.FileHosterResponse.FileHashesSizes,
		UnavailableFileHashes: make([]string, len(downloadContract.FileHosterResponse.UnavailableFileHashes)),
		Timestamp:             downloadContract.FileHosterResponse.Timestamp,
	}

	for i, j := range downloadContract.FileHosterResponse.FileHashes {
		dqrJSON.FileHashes[i] = hexutil.EncodeNoPrefix(j)
	}

	for i, j := range downloadContract.FileHosterResponse.UnavailableFileHashes {
		dqrJSON.UnavailableFileHashes[i] = hexutil.EncodeNoPrefix(j)
	}

	jsonContract := DownloadContractJSON{
		FileHosterResponse:         dqrJSON,
		FileRequesterNodePublicKey: hexutil.Encode(downloadContract.FileRequesterNodePublicKey),
		FileHashesNeeded:           make([]string, len(downloadContract.FileHashesNeeded)),
		FileHashesNeededSizes:      make([]uint64, len(downloadContract.FileHashesNeededSizes)),
		VerifierPublicKey:          hexutil.Encode(downloadContract.VerifierPublicKey),
		VerifierFees:               downloadContract.VerifierFees,
		ContractHash:               hexutil.Encode(downloadContract.ContractHash),
		VerifierSignature:          hexutil.Encode(downloadContract.VerifierSignature),
	}

	for i, j := range downloadContract.FileHashesNeeded {
		jsonContract.FileHashesNeeded[i] = hexutil.EncodeNoPrefix(j)
	}

	copy(jsonContract.FileHashesNeededSizes, downloadContract.FileHashesNeededSizes)
	response.Contract = jsonContract

	return nil
}

// CheckDataQueryResponseArgs is a data query response arg.
type CheckDataQueryResponseArgs struct {
	DataQueryRequestHash string `json:"data_query_request_hash"`
}

// CheckDataQueryResponse is a data query response payload.
type CheckDataQueryResponse struct {
	Responses []DataQueryResponseJSON `json:"responses"`
}

// DataQueryResponseJSON represents a json payload which represents a DataQueryResponse.
type DataQueryResponseJSON struct {
	FromPeerAddr          string   `json:"from_peer_addr"`
	FeesPerByte           string   `json:"fees_per_byte"`
	HashDataQueryRequest  string   `json:"hash_data_query_request"`
	PublicKey             string   `json:"public_key"`
	Signature             string   `json:"signature"`
	FileHashes            []string `json:"file_hashes"`
	FileHashesSizes       []uint64 `json:"file_hashes_sizes"`
	UnavailableFileHashes []string `json:"unavailable_file_hashes"`
	Timestamp             int64    `json:"timestamp"`
}

// CheckDataQueryResponse returns a list of data query responses.
func (api *DataTransferAPI) CheckDataQueryResponse(r *http.Request, args *CheckDataQueryResponseArgs, response *CheckDataQueryResponse) error {
	if args.DataQueryRequestHash == "" {
		return errors.New("data query hash is empty")
	}

	responses, _ := api.dataQueryProtocol.GetQueryResponse(args.DataQueryRequestHash)
	response.Responses = make([]DataQueryResponseJSON, 0)

	for _, v := range responses {
		dqrJSON := DataQueryResponseJSON{
			FromPeerAddr:          v.FromPeerAddr,
			FeesPerByte:           v.FeesPerByte,
			HashDataQueryRequest:  hexutil.Encode(v.HashDataQueryRequest),
			PublicKey:             hexutil.Encode(v.PublicKey),
			Signature:             hexutil.Encode(v.Signature),
			FileHashes:            make([]string, len(v.FileHashes)),
			FileHashesSizes:       v.FileHashesSizes,
			UnavailableFileHashes: make([]string, len(v.UnavailableFileHashes)),
			Timestamp:             v.Timestamp,
		}

		for i, j := range v.FileHashes {
			dqrJSON.FileHashes[i] = hexutil.EncodeNoPrefix(j)
		}

		for i, j := range v.UnavailableFileHashes {
			dqrJSON.UnavailableFileHashes[i] = hexutil.EncodeNoPrefix(j)
		}

		response.Responses = append(response.Responses, dqrJSON)
	}

	return nil
}

// RequestDataQueryResponseFromVerifiers returns a list of data query responses by contacting the verifiers.
func (api *DataTransferAPI) RequestDataQueryResponseFromVerifiers(r *http.Request, args *CheckDataQueryResponseArgs, response *CheckDataQueryResponse) error {
	if args.DataQueryRequestHash == "" {
		return errors.New("data query hash is empty")
	}

	dataQueryRequestHashBytes, err := hexutil.Decode(args.DataQueryRequestHash)
	if err != nil {
		return fmt.Errorf("failed to decode data query request hash: %w", err)
	}

	verfiers := block.GetBlockVerifiers()
	peerIDs := make([]peer.ID, 0)
	for _, v := range verfiers {
		publicKey, err := ffgcrypto.PublicKeyFromHex(v.PublicKey)
		if err != nil {
			continue
		}

		peerID, err := peer.IDFromPublicKey(publicKey)
		if err != nil {
			continue
		}
		peerIDs = append(peerIDs, peerID)
	}

	addrsInfos := api.publisherNodesFinder.FindPeers(r.Context(), peerIDs)
	dqrTransferRequest := &messages.DataQueryResponseTransferProto{Hash: dataQueryRequestHashBytes}
	if len(addrsInfos) > 0 {
		var wg sync.WaitGroup
		for _, addInfo := range addrsInfos {
			wg.Add(1)
			go func(peerID peer.ID) {
				defer wg.Done()
				_ = api.dataQueryProtocol.RequestDataQueryResponseTransfer(r.Context(), peerID, dqrTransferRequest)
			}(addInfo.ID)
		}
		wg.Wait()
	}

	// query again the inmem store to check if data query response from verifiers populated the store
	responses, _ := api.dataQueryProtocol.GetQueryResponse(args.DataQueryRequestHash)
	response.Responses = make([]DataQueryResponseJSON, 0)

	for _, v := range responses {
		dqrJSON := DataQueryResponseJSON{
			FromPeerAddr:          v.FromPeerAddr,
			FeesPerByte:           v.FeesPerByte,
			HashDataQueryRequest:  hexutil.Encode(v.HashDataQueryRequest),
			PublicKey:             hexutil.Encode(v.PublicKey),
			Signature:             hexutil.Encode(v.Signature),
			FileHashes:            make([]string, len(v.FileHashes)),
			FileHashesSizes:       v.FileHashesSizes,
			UnavailableFileHashes: make([]string, len(v.UnavailableFileHashes)),
			Timestamp:             v.Timestamp,
		}

		for i, j := range v.FileHashes {
			dqrJSON.FileHashes[i] = hexutil.EncodeNoPrefix(j)
		}

		for i, j := range v.UnavailableFileHashes {
			dqrJSON.UnavailableFileHashes[i] = hexutil.EncodeNoPrefix(j)
		}

		response.Responses = append(response.Responses, dqrJSON)
	}

	return nil
}

// PauseFileDownloadArgs represent args.
type PauseFileDownloadArgs struct {
	ContractHash string `json:"contract_hash"`
	FileHash     string `json:"file_hash"`
}

// PauseFileDownloadResponse represent response.
type PauseFileDownloadResponse struct{}

// PauseFileDownload pauses a file download.
func (api *DataTransferAPI) PauseFileDownload(r *http.Request, args *PauseFileDownloadArgs, response *PauseFileDownloadResponse) error {
	_, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	_, err = hexutil.DecodeNoPrefix(args.FileHash)
	if err != nil {
		return fmt.Errorf("failed to decode file hash: %w", err)
	}

	err = api.contractStore.CancelContractFileDownloadContexts(args.ContractHash + args.FileHash)
	if err != nil {
		return fmt.Errorf("failed to cancel download contexts: %w", err)
	}

	return nil
}

// DownloadFileArgs represent args.
type DownloadFileArgs struct {
	ContractHash string `json:"contract_hash"`
	FileHash     string `json:"file_hash"`
	ReDownload   bool   `json:"re_download"`
}

// DownloadFileArgs represents a response.
type DownloadFileResponse struct {
	Status string `json:"status"`
}

// DownloadFile downloads a file from a contract.
func (api *DataTransferAPI) DownloadFile(r *http.Request, args *DownloadFileArgs, response *DownloadFileResponse) error {
	downloadContract, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	fileHash, err := hexutil.DecodeNoPrefix(args.FileHash)
	if err != nil {
		return fmt.Errorf("failed to decode file hash: %w", err)
	}

	fileHoster, err := peer.Decode(downloadContract.FileHosterResponse.FromPeerAddr)
	if err != nil {
		return fmt.Errorf("failed to decode file hoster's peer id: %w", err)
	}

	fileSize := uint64(0)
	for i, v := range downloadContract.FileHashesNeeded {
		if bytes.Equal(v, fileHash) {
			fileSize = downloadContract.FileHashesNeededSizes[i]
		}
	}

	if fileSize == 0 {
		return fmt.Errorf("file size is zero")
	}

	// trigger a file initialization by seting the size of the file
	api.contractStore.SetFileSize(args.ContractHash, fileHash, fileSize)

	go func() {
		if args.ReDownload {
			// cancel all pending contexts
			_ = api.contractStore.CancelContractFileDownloadContexts(args.ContractHash + args.FileHash)

			// delete all the downloaded file parts
			fileParts := api.contractStore.GetDownoadedFilePartInfos(args.ContractHash, fileHash)
			for _, v := range fileParts {
				err := os.Remove(v.DestinationFilePath)
				if err != nil {
					log.Warnf("failed to remove old downloaded file part %s : %v", v.DestinationFilePath, err)
				}
			}

			// reset the file bytes transfered
			err := api.contractStore.ResetTransferedBytes(args.ContractHash, fileHash)
			if err != nil {
				log.Warnf("failed to rest file transfered bytes: %v", err)
			}
		}

		// create initial file ranges
		fileRanges := createFileRanges(int64(fileSize))

		// create file ranges from downloaded file parts
		fileRangesFromDownloadedParts, err := getDownloadedPartsInfo(filepath.Join(api.dataVerificationProtocol.GetDownloadDirectory(), args.ContractHash))
		if err == nil && len(fileRanges) == len(fileRangesFromDownloadedParts) {
			// if no error
			for i, v := range fileRanges {
				for _, fr := range fileRangesFromDownloadedParts {
					if v.from == fr.from && v.to == fr.to {
						fileRanges[i].availableSize = fr.availableSize
					}
				}
			}
		}

		wg := sync.WaitGroup{}
		for _, v := range fileRanges {
			v := v

			// skip the parts which are already downloaded
			if v.to-v.from+1 == v.availableSize {
				continue
			}

			wg.Add(1)
			go func(fileRange FileRanges) {
				request := &messages.FileTransferInfoProto{
					ContractHash: downloadContract.ContractHash,
					FileHash:     fileHash,
					FileSize:     fileSize,
					From:         fileRange.from + fileRange.availableSize,
					To:           fileRange.to,
				}

				ctxWithCancel, cancel := context.WithCancel(context.Background())
				api.contractStore.SetContractFileDownloadContexts(args.ContractHash+args.FileHash, contract.ContextFileDownloadData{
					From:   fileRange.from + fileRange.availableSize,
					To:     fileRange.to,
					Ctx:    ctxWithCancel,
					Cancel: cancel,
				})

				fileHashHex := hexutil.EncodeNoPrefix(request.FileHash)
				fileNameWithPart := fmt.Sprintf("%s_part_%d_%d", fileHashHex, fileRange.from, fileRange.to)
				destinationFilePath := filepath.Join(api.dataVerificationProtocol.GetDownloadDirectory(), hexutil.Encode(request.ContractHash), fileNameWithPart)

				_, err := api.dataVerificationProtocol.RequestFileTransfer(ctxWithCancel, destinationFilePath, fileNameWithPart, fileHoster, request)
				// if the context wasnt canceled set the error
				if err != nil && !errors.Is(err, context.Canceled) {
					fileHashHex := hexutil.EncodeNoPrefix(fileHash)
					fileNameWithPart := fmt.Sprintf("%s_part_%d_%d", fileHashHex, fileRange.from, fileRange.to)
					api.contractStore.SetFilePartDownloadError(args.ContractHash, fileHash, fileNameWithPart, err.Error())
				}

				wg.Done()
			}(v)
		}
		wg.Wait()

		// check if all file parts have been downloaded
		totalDownloaded := api.contractStore.GetTransferedBytes(args.ContractHash, fileHash)
		if totalDownloaded != fileSize {
			api.contractStore.SetError(args.ContractHash, fileHash, fmt.Sprintf("total downloaded parts size (%d) is not equal to the file size (%d)", totalDownloaded, fileSize))
			return
		}

		// reassemble all file parts
		filePartInfos := api.contractStore.GetDownoadedFilePartInfos(args.ContractHash, fileHash)
		outputFilePath := filepath.Join(filepath.Dir(filePartInfos[0].DestinationFilePath), args.FileHash)
		log.Info("outputfile path ", outputFilePath)
		fileParts := make([]string, len(filePartInfos))
		for i, v := range filePartInfos {
			fileParts[i] = v.DestinationFilePath
		}
		err = common.ConcatenateFiles(outputFilePath, fileParts)
		if err != nil {
			api.contractStore.SetError(args.ContractHash, fileHash, fmt.Sprintf("failed to concatenate downloaded file parts: %s", err.Error()))
			return
		}

		// delete the part files
		for _, v := range fileParts {
			err := os.Remove(v)
			if err != nil {
				log.Warnf("failed to remove file %s : %v", v, err)
			}
		}
	}()

	response.Status = "started"

	return nil
}

func getDownloadedPartsInfo(downloadedPartsFolder string) ([]FileRanges, error) {
	filesRanges := make([]FileRanges, 0)
	f, err := os.Open(downloadedPartsFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to open downloaded parts folder: %w", err)
	}

	fileInfo, err := f.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded parts folder content: %w", err)
	}
	f.Close()

	for _, file := range fileInfo {
		fileParts := strings.Split(file.Name(), "_part_")
		if len(fileParts) == 2 {
			fromToParts := strings.Split(fileParts[1], "_")
			if len(fromToParts) == 2 {
				from, err := strconv.ParseInt(fromToParts[0], 10, 64)
				if err != nil {
					continue
				}
				to, err := strconv.ParseInt(fromToParts[1], 10, 64)
				if err != nil {
					continue
				}

				tmpRange := FileRanges{
					from:          from,
					to:            to,
					availableSize: file.Size(),
				}
				filesRanges = append(filesRanges, tmpRange)
			}
		}
	}

	sort.Slice(filesRanges, func(i, j int) bool { return filesRanges[i].to < filesRanges[j].to })

	return filesRanges, nil
}

type FileRanges struct {
	from          int64
	to            int64
	availableSize int64
}

func createFileRanges(fileSize int64) []FileRanges {
	numWorkers := int64(4)
	chunkSize := fileSize / numWorkers
	ranges := make([]FileRanges, 0)

	if chunkSize == 0 {
		ranges = append(ranges, FileRanges{
			from: 0,
			to:   fileSize - 1,
		})
		return ranges
	}

	for i := 0; i < int(numWorkers); i++ {
		start := int64(i) * chunkSize
		end := int64(0)
		if i == int(numWorkers-1) {
			end = fileSize - 1
		} else {
			end = start + chunkSize - 1
		}

		ranges = append(ranges, FileRanges{
			from: start,
			to:   end,
		})
	}

	return ranges
}

// DownloadFileProgressArgs represent args.
type DownloadFileProgressArgs struct {
	ContractHash string `json:"contract_hash"`
	FileHash     string `json:"file_hash"`
}

// DownloadFileProgressResponse represents the response of a download file progress.
type DownloadFileProgressResponse struct {
	Error           string `json:"error"`
	BytesTransfered uint64 `json:"bytes_transfered"`
}

// DownloadFileProgress returns the download progress of a file.
func (api *DataTransferAPI) DownloadFileProgress(r *http.Request, args *DownloadFileProgressArgs, response *DownloadFileProgressResponse) error {
	fileHash, err := hexutil.DecodeNoPrefix(args.FileHash)
	if err != nil {
		return fmt.Errorf("failed to decode file hash: %w", err)
	}

	fileInfo, err := api.contractStore.GetContractFileInfo(args.ContractHash, fileHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	response.BytesTransfered = api.contractStore.GetTransferedBytes(args.ContractHash, fileHash)
	response.Error = fileInfo.Error

	return nil
}

// SendFileMerkleTreeNodesToVerifierArgs represents args.
type SendFileMerkleTreeNodesToVerifierArgs struct {
	ContractHash string `json:"contract_hash"`
	FileHash     string `json:"file_hash"`
}

// SendFileMerkleTreeNodesToVerifierResponse represents a struct.
type SendFileMerkleTreeNodesToVerifierResponse struct {
	Success bool `json:"success"`
}

// SendFileMerkleTreeNodesToVerifier sends the merkle tree nodes of a downloaded encrypted file to verifier from the file downloader.
func (api *DataTransferAPI) SendFileMerkleTreeNodesToVerifier(r *http.Request, args *SendFileMerkleTreeNodesToVerifierArgs, response *SendFileMerkleTreeNodesToVerifierResponse) error {
	fileHash, err := hexutil.DecodeNoPrefix(args.FileHash)
	if err != nil {
		return fmt.Errorf("failed to decode file hash: %w", err)
	}

	downloadContract, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	fileInfo, err := api.contractStore.GetContractFileInfo(args.ContractHash, fileHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	transferedBytes := api.contractStore.GetTransferedBytes(args.ContractHash, fileHash)
	if fileInfo.Error != "" {
		return fmt.Errorf("contract file info failure: %s", fileInfo.Error)
	}

	if fileInfo.FileSize != transferedBytes {
		return fmt.Errorf("file wasn't fully transfered: size: %d, transfered: %d", fileInfo.FileSize, transferedBytes)
	}

	totalDesiredSegments, _ := api.dataVerificationProtocol.GetMerkleTreeFileSegmentsEncryptionPercentage()
	downloadDir := api.dataVerificationProtocol.GetDownloadDirectory()
	fileHashWithPrefix := hexutil.EncodeNoPrefix(fileHash)
	destinationFilePath := filepath.Join(downloadDir, args.ContractHash, fileHashWithPrefix)

	orderedSlice := make([]int, totalDesiredSegments)
	for i := 0; i < totalDesiredSegments; i++ {
		orderedSlice[i] = i
	}

	merkleNodes, err := common.HashFileBlockSegments(destinationFilePath, totalDesiredSegments, orderedSlice)
	if err != nil {
		return fmt.Errorf("failed to hash downloaded file block segments: %w", err)
	}

	contractHash, err := hexutil.Decode(args.ContractHash)
	if err != nil {
		return fmt.Errorf("failed to decode contract hash: %w", err)
	}

	merkleRequest := &messages.MerkleTreeNodesOfFileContractProto{
		ContractHash:    contractHash,
		FileHash:        fileHash,
		MerkleTreeNodes: make([][]byte, len(merkleNodes)),
	}

	for i, v := range merkleNodes {
		merkleRequest.MerkleTreeNodes[i] = make([]byte, len(v.X))
		copy(merkleRequest.MerkleTreeNodes[i], v.X)
	}

	publicKey, err := ffgcrypto.PublicKeyFromBytes(downloadContract.VerifierPublicKey)
	if err != nil {
		return fmt.Errorf("failed to get verifier's public key: %w", err)
	}

	verifierID, err := peer.IDFromPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to get verifier's peer id: %w", err)
	}

	err = api.dataVerificationProtocol.SendFileMerkleTreeNodesToVerifier(context.Background(), verifierID, merkleRequest)
	if err != nil {
		return fmt.Errorf("failed to send merkle tree nodes to verifier: %w", err)
	}

	response.Success = true

	return nil
}

// RequestEncryptionDataFromVerifierArgs represents args.
type RequestEncryptionDataFromVerifierArgs struct {
	ContractHash         string   `json:"contract_hash"`
	FileHashes           []string `json:"file_hashes"`
	FileMerkleRootHashes []string `json:"file_merkle_root_hashes"`
	RestoredFilePaths    []string `json:"restored_file_paths"`
}

// RequestEncryptionDataFromVerifierResponse represents the response.
type RequestEncryptionDataFromVerifierResponse struct {
	DecryptedFilePaths []string `json:"decrypted_file_paths"`
}

// RequestEncryptionDataFromVerifierAndDecrypt requires encryption data from verifier and decrypts.
func (api *DataTransferAPI) RequestEncryptionDataFromVerifierAndDecrypt(r *http.Request, args *RequestEncryptionDataFromVerifierArgs, response *RequestEncryptionDataFromVerifierResponse) error {
	if len(args.FileHashes) != len(args.FileMerkleRootHashes) {
		return errors.New("size of merkle root hashes and the file hashes are not equal")
	}
	downloadContract, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	encRequest := &messages.KeyIVRequestsProto{
		KeyIvs: make([]*messages.KeyIVProto, 0),
	}

	for idx, v := range args.FileHashes {
		fileHash, err := hexutil.DecodeNoPrefix(v)
		if err != nil {
			return fmt.Errorf("failed to decode file hash: %w", err)
		}

		fileInfo, err := api.contractStore.GetContractFileInfo(args.ContractHash, fileHash)
		if err != nil {
			return fmt.Errorf("contract not found: %w", err)
		}

		transferedBytes := api.contractStore.GetTransferedBytes(args.ContractHash, fileHash)
		if fileInfo.Error != "" {
			return fmt.Errorf("contract file info failure: %s", fileInfo.Error)
		}

		if fileInfo.FileSize != transferedBytes {
			return fmt.Errorf("file wasn't fully transfered: size: %d, transfered: %d", fileInfo.FileSize, transferedBytes)
		}

		contractHashBytes, err := hexutil.Decode(args.ContractHash)
		if err != nil {
			return fmt.Errorf("failed to decode contract hash: %w", err)
		}

		merkleRootHash, err := hexutil.Decode(args.FileMerkleRootHashes[idx])
		if err != nil {
			return fmt.Errorf("failed to decode merkle root hash: %w", err)
		}

		encRequest.KeyIvs = append(encRequest.KeyIvs, &messages.KeyIVProto{
			ContractHash:       contractHashBytes,
			FileHash:           fileHash,
			FileMerkleRootHash: merkleRootHash,
		})
	}

	publicKey, err := ffgcrypto.PublicKeyFromBytes(downloadContract.VerifierPublicKey)
	if err != nil {
		return fmt.Errorf("failed to get verifier's public key: %w", err)
	}

	verifierID, err := peer.IDFromPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to get verifier's peer id: %w", err)
	}

	encryptionData, err := api.dataVerificationProtocol.RequestEncryptionData(context.Background(), verifierID, encRequest)
	if err != nil {
		return fmt.Errorf("failed to request decryption data from verifier: %w", err)
	}

	response.DecryptedFilePaths = make([]string, 0)
	for i, v := range encryptionData.KeyIvRandomizedFileSegments {
		foundIdx := -1
		for j, w := range encRequest.KeyIvs {
			if bytes.Equal(v.FileHash, w.FileHash) {
				foundIdx = j
				break
			}
		}

		if foundIdx == -1 {
			return fmt.Errorf("decryption data doesn't contain the requested file hash: %s", hexutil.Encode(v.FileHash))
		}

		randomizedSegsFromKey := make([]int, len(encryptionData.KeyIvRandomizedFileSegments[i].RandomizedSegments))
		for i, v := range encryptionData.KeyIvRandomizedFileSegments[i].RandomizedSegments {
			randomizedSegsFromKey[i] = int(v)
		}

		fileInfo, err := api.contractStore.GetContractFileInfo(args.ContractHash, v.FileHash)
		if err != nil {
			return fmt.Errorf("failed to get a file of a contract: %w", err)
		}

		if fileInfo.FileDecryptionStatus == contract.FileDecrypting {
			continue
		}

		api.contractStore.SetFileDecryptionStatus(args.ContractHash, v.FileHash, contract.FileDecrypting)
		outputPathOfFile := args.RestoredFilePaths[foundIdx]
		inputEncryptedFilePath := filepath.Join(api.dataVerificationProtocol.GetDownloadDirectory(), hexutil.Encode(v.ContractHash), hexutil.EncodeNoPrefix(v.FileHash))
		decryptedPath, err := api.dataVerificationProtocol.DecryptFile(inputEncryptedFilePath, outputPathOfFile, encryptionData.KeyIvRandomizedFileSegments[i].Key, encryptionData.KeyIvRandomizedFileSegments[i].Iv, common.EncryptionType(encryptionData.KeyIvRandomizedFileSegments[i].EncryptionType), randomizedSegsFromKey, fileInfo.FileDecryptionStatus == contract.FileDecrypted)
		if err != nil {
			api.contractStore.SetFileDecryptionStatus(args.ContractHash, v.FileHash, contract.FileDecryptionError)
			return fmt.Errorf("failed to decrypt file %s with message: %w", hexutil.EncodeNoPrefix(v.FileHash), err)
		}
		api.contractStore.SetFileDecryptionStatus(args.ContractHash, v.FileHash, contract.FileDecrypted)
		response.DecryptedFilePaths = append(response.DecryptedFilePaths, decryptedPath)
	}

	return nil
}

// SendContractToFileHosterAndVerifierArgs represents the args.
type SendContractToFileHosterAndVerifierArgs struct {
	ContractHash string `json:"contract_hash"`
}

// SendContractToFileHosterAndVerifierResponse represents a response.
type SendContractToFileHosterAndVerifierResponse struct {
	Success bool `json:"success"`
}

// SendContractToFileHosterAndVerifier sends the contract to file hoster and verifier.
func (api *DataTransferAPI) SendContractToFileHosterAndVerifier(r *http.Request, args *SendContractToFileHosterAndVerifierArgs, response *SendContractToFileHosterAndVerifierResponse) error {
	downloadContract, err := api.contractStore.GetContract(args.ContractHash)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	publicKeyVerifier, err := ffgcrypto.PublicKeyFromBytes(downloadContract.VerifierPublicKey)
	if err != nil {
		return fmt.Errorf("failed to get verifier's public key: %w", err)
	}

	verifierID, err := peer.IDFromPublicKey(publicKeyVerifier)
	if err != nil {
		return fmt.Errorf("failed to get verifier's peer id: %w", err)
	}

	publicKeyHoster, err := ffgcrypto.PublicKeyFromBytes(downloadContract.FileHosterResponse.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to get file hoster's public key: %w", err)
	}

	fileHosterID, err := peer.IDFromPublicKey(publicKeyHoster)
	if err != nil {
		return fmt.Errorf("failed to get file hoster's peer id: %w", err)
	}

	err = api.dataVerificationProtocol.TransferContract(r.Context(), verifierID, downloadContract)
	if err != nil {
		return fmt.Errorf("failed to send contract to verifier: %w", err)
	}

	err = api.dataVerificationProtocol.TransferContract(r.Context(), fileHosterID, downloadContract)
	if err != nil {
		return fmt.Errorf("failed to send contract to file hoster: %w", err)
	}

	response.Success = true
	return nil
}

// CreateTransactionDataPayloadFromContractHashesArgs represent the function args.
type CreateTransactionDataPayloadFromContractHashesArgs struct {
	AccessToken             string   `json:"access_token"`
	ContractHashes          []string `json:"contract_hashes"`
	CurrentNounce           string   `json:"current_nounce"`
	TransactionFeesToBeUsed string   `json:"transaction_fees_to_be_used"`
}

// CreateTransactionDataPayloadFromContractHashesResponse represent the function response.
type CreateTransactionDataPayloadFromContractHashesResponse struct {
	TransactionDataBytesHex  []string `json:"transaction_data_bytes_hex"`
	TotalFeesForTransactions string   `json:"total_fees_for_transaction"`
}

// CreateTransactionsWithDataPayloadFromContractHashes given a list of contract hashes it creates the transactions and its data payloads.
func (api *DataTransferAPI) CreateTransactionsWithDataPayloadFromContractHashes(r *http.Request, args *CreateTransactionDataPayloadFromContractHashesArgs, response *CreateTransactionDataPayloadFromContractHashesResponse) error {
	ok, key, err := api.keystore.Authorized(args.AccessToken)
	if err != nil || !ok {
		return fmt.Errorf("failed to authorize access token %v", err)
	}
	response.TransactionDataBytesHex = make([]string, 0)
	currentNounce, err := hexutil.DecodeUint64(args.CurrentNounce)
	if err != nil {
		return fmt.Errorf("failed to decode current nounce: %w", err)
	}

	transactionFees, err := hexutil.DecodeBig(args.TransactionFeesToBeUsed)
	if err != nil {
		return fmt.Errorf("failed to decode transaction fees to be used for transaction: %w", err)
	}
	allTransactionFess := big.NewInt(0)
	for _, v := range args.ContractHashes {
		currentNounce++
		downloadContract, err := api.contractStore.GetContract(v)
		if err != nil {
			return fmt.Errorf("failed to get contract: %w", err)
		}

		dcinTX := &messages.DownloadContractInTransactionDataProto{
			ContractHash:               downloadContract.ContractHash,
			FileRequesterNodePublicKey: downloadContract.FileRequesterNodePublicKey,
			FileHosterNodePublicKey:    downloadContract.FileHosterResponse.PublicKey,
			VerifierPublicKey:          downloadContract.VerifierPublicKey,
			VerifierFees:               downloadContract.VerifierFees,
			FileHosterFees:             downloadContract.FileHosterResponse.FeesPerByte,
		}

		contractsEnvelope := &messages.DownloadContractsHashesProto{
			Contracts: []*messages.DownloadContractInTransactionDataProto{dcinTX},
		}

		itemsBytes, err := proto.Marshal(contractsEnvelope)
		if err != nil {
			return fmt.Errorf("failed to marshal contract envelope: %w", err)
		}
		txPayload := transaction.DataPayload{
			Type:    transaction.DataType_DATA_CONTRACT,
			Payload: itemsBytes,
		}
		txPayloadBytes, err := proto.Marshal(&txPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal data payload with contract envelope inside: %w", err)
		}

		dataverifierAddr, err := ffgcrypto.RawPublicToAddress(downloadContract.VerifierPublicKey)
		if err != nil {
			return fmt.Errorf("failed to get the address of data verifier: %w", err)
		}
		publicKeyOfTxSigner, err := key.Key.PublicKey.Raw()
		if err != nil {
			return fmt.Errorf("failed get the public key bytes of unlocked address: %w", err)
		}
		mainChain, _ := hexutil.Decode(transaction.ChainID)

		totalFileSize := uint64(0)
		for _, v := range downloadContract.FileHashesNeededSizes {
			totalFileSize += v
		}
		fileHosterFees, err := hexutil.DecodeBig(downloadContract.FileHosterResponse.FeesPerByte)
		if err != nil {
			return fmt.Errorf("failed to decode file hosters fees: %w", err)
		}

		fileHosterFees = fileHosterFees.Mul(fileHosterFees, big.NewInt(0).SetUint64(totalFileSize))
		verifierFees, err := hexutil.DecodeBig(downloadContract.VerifierFees)
		if err != nil {
			return fmt.Errorf("failed to decode verifier's fees: %w", err)
		}

		totalFees := currency.FFGZero().Add(fileHosterFees, verifierFees)
		allTransactionFess = allTransactionFess.Add(allTransactionFess, totalFees)
		tx := transaction.Transaction{
			PublicKey:       publicKeyOfTxSigner,
			Nounce:          hexutil.EncodeUint64ToBytes(currentNounce),
			Data:            txPayloadBytes,
			From:            key.Key.Address,
			To:              dataverifierAddr,
			Value:           hexutil.EncodeBig(totalFees),
			TransactionFees: hexutil.EncodeBig(transactionFees),
			Chain:           mainChain,
		}
		err = tx.Sign(key.Key.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to sign the transaction with data contract inside: %w", err)
		}
		ok, err = tx.Validate()
		if !ok || err != nil {
			return fmt.Errorf("failed to validate transaction: %w", err)
		}
		JSONTx := JSONTransaction{
			Hash:            hexutil.Encode(tx.Hash),
			Signature:       hexutil.Encode(tx.Signature),
			PublicKey:       hexutil.Encode(tx.PublicKey),
			Nounce:          hexutil.EncodeUint64BytesToHexString(tx.Nounce),
			Data:            hexutil.Encode(tx.Data),
			From:            tx.From,
			To:              tx.To,
			Value:           tx.Value,
			TransactionFees: tx.TransactionFees,
			Chain:           hexutil.Encode(mainChain),
		}

		JSONTxBytes, err := json.Marshal(JSONTx)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON transaction: %w", err)
		}

		response.TransactionDataBytesHex = append(response.TransactionDataBytesHex, string(JSONTxBytes))
	}

	response.TotalFeesForTransactions = hexutil.EncodeBig(allTransactionFess)

	return nil
}

// CreateContractsFromDataQueryResponseHashArgs represents args.
type CreateContractsFromDataQueryResponsesArgs struct {
	DataQueryRequestHash string `json:"data_query_request_hash"`
}

// CreateContractsFromDataQueryResponsesResponse represents the response.
type CreateContractsFromDataQueryResponsesResponse struct {
	ContractHashes []string `json:"contract_hashes"`
}

type filesNeededInDataQueryResponse struct {
	response              *messages.DataQueryResponse
	fileHashesNeeded      [][]byte
	fileHashesSizesNeeded []uint64
}

// CreateContractsFromDataQueryResponses creates contracts from the available data query responses.
func (api *DataTransferAPI) CreateContractsFromDataQueryResponses(r *http.Request, args *CreateContractsFromDataQueryResponsesArgs, response *CreateContractsFromDataQueryResponsesResponse) error {
	response.ContractHashes = make([]string, 0)
	requests, ok := api.dataQueryProtocol.GetQueryHistory(args.DataQueryRequestHash)
	if !ok {
		return fmt.Errorf("data query request not found %s", args.DataQueryRequestHash)
	}

	// TODO: data query responses validation of available files hashes and unavailable files hashes
	// they should sum to the total files requested from the data query

	responses, ok := api.dataQueryProtocol.GetQueryResponse(args.DataQueryRequestHash)
	if !ok {
		return fmt.Errorf("data query responses not found %s", args.DataQueryRequestHash)
	}

	filesNeeded, err := getFilesNeededFromDataQueryResponses(requests, responses)
	if err != nil {
		return fmt.Errorf("failed to get files needed from responses: %w", err)
	}

	// TODO: check which one to send and how many contracts from filesNeeded

	requesterPubKeyBytes, err := api.host.Peerstore().PubKey(api.host.ID()).Raw()
	if err != nil {
		return fmt.Errorf("failed to get node's public key bytes %w", err)
	}

	// find all verifiers
	verfiers := block.GetBlockVerifiers()
	peerIDs := make([]peer.ID, 0)
	for _, v := range verfiers {
		publicKey, err := ffgcrypto.PublicKeyFromHex(v.PublicKey)
		if err != nil {
			continue
		}

		peerID, err := peer.IDFromPublicKey(publicKey)
		if err != nil {
			continue
		}
		peerIDs = append(peerIDs, peerID)
	}

	downloadContracts := make([]*messages.DownloadContractProto, 0)
	for _, v := range filesNeeded {
		contract := &messages.DownloadContractProto{
			FileHosterResponse:         messages.ToDataQueryResponseProto(*v.response),
			FileRequesterNodePublicKey: requesterPubKeyBytes,
			FileHashesNeeded:           v.fileHashesNeeded,
			FileHashesNeededSizes:      v.fileHashesSizesNeeded,
		}
		downloadContracts = append(downloadContracts, contract)
	}
	addrsInfos := api.publisherNodesFinder.FindPeers(r.Context(), peerIDs)
	signedDownloadContracts := make([]*messages.DownloadContractProto, 0)
	mux := sync.Mutex{}

	if len(addrsInfos) > 0 {
		var wg sync.WaitGroup
		for _, addInfo := range addrsInfos {
			wg.Add(1)
			go func(peerID peer.ID) {
				defer wg.Done()

				// send contract to verifiers
				for _, unsignedContract := range downloadContracts {
					signedDownloadContract, err := api.dataVerificationProtocol.SendContractToVerifierForAcceptance(context.Background(), peerID, unsignedContract)
					if err != nil {
						return
					}

					verifierPubKey, err := ffgcrypto.PublicKeyFromBytes(signedDownloadContract.VerifierPublicKey)
					if err != nil {
						return
					}

					ok, err := messages.VerifyDownloadContractProto(verifierPubKey, signedDownloadContract)
					if !ok || err != nil {
						return
					}

					mux.Lock()
					signedDownloadContracts = append(signedDownloadContracts, signedDownloadContract)
					mux.Unlock()
				}
			}(addInfo.ID)
		}
		wg.Wait()
	} else {
		return fmt.Errorf("failed to find verifiers in the network")
	}

	// shuffle the result to introduce extra pseudorandom
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(signedDownloadContracts), func(i, j int) {
		signedDownloadContracts[i], signedDownloadContracts[j] = signedDownloadContracts[j], signedDownloadContracts[i]
	})

	// at this stage we should have a list of mixed download contracts from all verifiers
	// find the number of contracts
	selectedSignedDownloadContracts := make([]*messages.DownloadContractProto, 0)
	for _, v := range downloadContracts {
		neededHashOfContract := bytes.Join(
			v.FileHashesNeeded,
			[]byte{},
		)

		for _, signedContract := range signedDownloadContracts {
			signedContractFileHash := bytes.Join(
				signedContract.FileHashesNeeded,
				[]byte{},
			)

			if bytes.Equal(neededHashOfContract, signedContractFileHash) {
				selectedSignedDownloadContracts = append(selectedSignedDownloadContracts, signedContract)
				break
			}
		}
	}

	if len(selectedSignedDownloadContracts) != len(downloadContracts) {
		return errors.New("incomplete number of contracts returned from verifiers")
	}

	for _, v := range selectedSignedDownloadContracts {
		_ = api.contractStore.CreateContract(v)
		response.ContractHashes = append(response.ContractHashes, hexutil.Encode(v.ContractHash))
	}

	return nil
}

type copyResponse struct {
	response messages.DataQueryResponse
	// availableHashes [][]byte
}

func getFilesNeededFromDataQueryResponses(requests messages.DataQueryRequest, responses []messages.DataQueryResponse) ([]filesNeededInDataQueryResponse, error) {
	// find if all file hashes in the request have a data query response
	filesNeeded := make(map[string]filesNeededInDataQueryResponse, 0)
	for _, fhreq := range requests.FileHashes {
		foundFileHashInResponses := false
		for outerIdx, dqresponse := range responses {
			for fhIdx, fhresp := range dqresponse.FileHashes {
				if bytes.Equal(fhreq, fhresp) {
					m, ok := filesNeeded[hexutil.Encode(dqresponse.Signature)]
					if !ok {
						filesNeeded[hexutil.Encode(dqresponse.Signature)] = filesNeededInDataQueryResponse{
							response:              &responses[outerIdx],
							fileHashesNeeded:      [][]byte{fhreq},
							fileHashesSizesNeeded: []uint64{dqresponse.FileHashesSizes[fhIdx]},
						}
					} else {
						m.fileHashesNeeded = append(m.fileHashesNeeded, fhreq)
						m.fileHashesSizesNeeded = append(m.fileHashesSizesNeeded, dqresponse.FileHashesSizes[fhIdx])
						filesNeeded[hexutil.Encode(dqresponse.Signature)] = m
					}

					foundFileHashInResponses = true
					break
				}
			}
		}

		if !foundFileHashInResponses {
			return nil, fmt.Errorf("incomplete data responses: file hash %s was not found in the data query responses", hexutil.Encode(fhreq))
		}
	}

	if len(filesNeeded) == 0 {
		return nil, errors.New("requested files were not found in data query responses")
	}
	// error validation finished

	copyResponses := make([]copyResponse, len(responses))
	for i, v := range responses {
		copyResponses[i] = copyResponse{
			response: v,
		}
	}

	sort.Slice(copyResponses, func(i, j int) bool {
		return len(copyResponses[i].response.UnavailableFileHashes) < len(copyResponses[j].response.UnavailableFileHashes)
	})

	// if first response includes all the files needed return it
	wantedResponses := make([]filesNeededInDataQueryResponse, 0)
	wantedResponses = append(wantedResponses, filesNeededInDataQueryResponse{
		response:              &copyResponses[0].response,
		fileHashesNeeded:      copyResponses[0].response.FileHashes,
		fileHashesSizesNeeded: copyResponses[0].response.FileHashesSizes,
	})
	if len(copyResponses[0].response.UnavailableFileHashes) == 0 {
		return wantedResponses, nil
	}

	// at this stage there is a slice of responses with sorted unavailable files in asc order
	// take those unavailable file hashes, and find where they can be found in another response
	unavailableFileHashes := make([][]byte, len(copyResponses[0].response.UnavailableFileHashes))
	copy(unavailableFileHashes, copyResponses[0].response.UnavailableFileHashes)
	// if there is a response that doesn't include any of the list from unavailableFileHashes then choose it as second contract
	for _, resp := range copyResponses {
		foundResponse := true
		for _, unavailableFile := range unavailableFileHashes {
			for _, ununavailableFileResponse := range resp.response.UnavailableFileHashes {
				if bytes.Equal(unavailableFile, ununavailableFileResponse) {
					foundResponse = false
					break
				}
			}
		}

		if foundResponse {
			wantedResponses = append(wantedResponses, filesNeededInDataQueryResponse{
				response:              &resp.response,
				fileHashesNeeded:      resp.response.FileHashes,
				fileHashesSizesNeeded: resp.response.FileHashesSizes,
			})
			return wantedResponses, nil
		}
	}

	for _, v := range copyResponses {
		tmpCopyResp := filesNeededInDataQueryResponse{
			response:              &v.response,
			fileHashesNeeded:      make([][]byte, 0),
			fileHashesSizesNeeded: make([]uint64, 0),
		}

		for idx, availableHasheCopyResponse := range v.response.FileHashes {
			for _, notavailableFile := range unavailableFileHashes {
				if bytes.Equal(notavailableFile, availableHasheCopyResponse) {
					tmpCopyResp.fileHashesNeeded = append(tmpCopyResp.fileHashesNeeded, availableHasheCopyResponse)
					tmpCopyResp.fileHashesSizesNeeded = append(tmpCopyResp.fileHashesSizesNeeded, v.response.FileHashesSizes[idx])
				}
			}
		}

		if len(tmpCopyResp.fileHashesNeeded) != 0 {
			wantedResponses = append(wantedResponses, tmpCopyResp)
			whichIndexesToRemove := []int{}
			for idxUnavailableHash, unavailableHash := range unavailableFileHashes {
				for _, p := range tmpCopyResp.fileHashesNeeded {
					if bytes.Equal(unavailableHash, p) {
						whichIndexesToRemove = append(whichIndexesToRemove, idxUnavailableHash)
					}
				}
			}

			for _, remove := range whichIndexesToRemove {
				unavailableFileHashes = append(unavailableFileHashes[:remove], unavailableFileHashes[remove+1:]...)
			}
		}
	}

	flattenFilesNeeded := make([][]byte, 0)
	for _, v := range wantedResponses {
		flattenFilesNeeded = append(flattenFilesNeeded, v.fileHashesNeeded...)
	}

	if len(flattenFilesNeeded) != len(requests.FileHashes) {
		return nil, errors.New("failed to coordinate the responses into multiple contracts, found files aren't equal to the requested files")
	}

	return wantedResponses, nil
}
