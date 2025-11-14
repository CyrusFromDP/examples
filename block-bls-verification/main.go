// main.go
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/herumi/bls-eth-go-binary/bls"
)

// RPC response wrappers
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// validator info from debug_getValidatorInfo
type ValidatorSetInfo struct {
	BlsKey      string `json:"blsKey"`
	IdentityKey string `json:"identityKey"`
	Staking     string `json:"staking"`
	ValidatorID string `json:"validatorID"`
}

// debug_getValidatorInfo
type ValidatorInfo struct {
	BlockNumber  string             `json:"blockNumber"`
	ValidatorSet []ValidatorSetInfo `json:"validatorSet"`
}

// debug_getBlockProof
type BlockProof struct {
	BlockNumber            string   `json:"blockNumber"`
	BlockProofHash         string   `json:"blockProofHash"`
	BlsAggregatedSignature string   `json:"blsAggregatedSignature"`
	SignedBlsKeys          []string `json:"signedBlsKeys"`
}

// helpers
func trim0x(s string) string {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return s[2:]
	}
	return s
}

func hexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(trim0x(strings.TrimSpace(s)))
}

// rpcPost posts a JSON-RPC request and returns the "result" raw json
func rpcPost(url, method string, params interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r rpcResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("unmarshal rpc response: %w (body=%s)", err, string(body))
	}
	if r.Error != nil {
		return nil, fmt.Errorf("rpc error: %d %s", r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}

func fetchBlockNumber(rpcURL string) (string, error) {
	resultRaw, err := rpcPost(rpcURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		return "0x0", fmt.Errorf("rpc call eth_blockNumber failed: %w", err)
	}

	var hexStr string
	if err := json.Unmarshal(resultRaw, &hexStr); err != nil {
		return "0x0", fmt.Errorf("parse eth_blockNumber result failed: %w", err)
	}

	return hexStr, nil

}

func fetchValidators(rpcURL string, height interface{}) ([]ValidatorSetInfo, error) {
	resultRaw, err := rpcPost(rpcURL, "debug_getValidatorInfo", []interface{}{height})
	if err != nil {
		return nil, err
	}

	var vInfo ValidatorInfo
	if err := json.Unmarshal(resultRaw, &vInfo); err != nil {
		return nil, err
	}

	validatorSetInfo := vInfo.ValidatorSet
	return validatorSetInfo, nil
}

func fetchBlockProof(rpcURL string, height interface{}) (*BlockProof, error) {
	resultRaw, err := rpcPost(rpcURL, "debug_getBlockProof", []interface{}{height})
	if err != nil {
		return nil, err
	}
	var bp BlockProof
	if err := json.Unmarshal(resultRaw, &bp); err != nil {
		return nil, fmt.Errorf("parse block proof: %w", err)
	}
	return &bp, nil
}

func parseStake(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	b := new(big.Int)
	if _, ok := b.SetString(s, 0); !ok {
		if strings.HasPrefix(strings.ToLower(s), "0x") {
			if _, ok2 := b.SetString(trim0x(s), 16); ok2 {
				return b, nil
			}
		}
		return nil, fmt.Errorf("cannot parse stake value: %s", s)
	}
	return b, nil
}

func main() {
	// CLI flags
	rpcURL := flag.String("rpc", "https://atlantic.dplabs-internal.com/", "JSON-RPC endpoint")
	height := flag.String("height", "latest", "block height or 'latest'")
	// optional: timeout or other flags can be added
	flag.Parse()

	if *height == "latest" {
		// get latest block number
		blockNumberHex, err := fetchBlockNumber(*rpcURL)
		if err != nil {
			log.Fatalf("fetch blockNumber failed: %v", err)
		}
		height = &blockNumberHex
	}

	// init bls
	if err := bls.Init(bls.BLS12_381); err != nil {
		log.Fatalf("bls init failed: %v", err)
	}

	fmt.Println("RPC:", *rpcURL, " height:", *height)

	// 1) fetch validators
	validators, err := fetchValidators(*rpcURL, *height)
	if err != nil {
		log.Fatalf("fetch validators failed: %v", err)
	}
	fmt.Printf("fetched %d validators\n", len(validators))

	// build map from bls_key -> ValidatorSetInfo
	validatorMap := make(map[string]ValidatorSetInfo)
	totalStake := new(big.Int)
	for _, v := range validators {
		k := strings.ToLower(trim0x(v.BlsKey))
		validatorMap[k] = v

		st, err := parseStake(v.Staking)
		if err != nil {
			log.Fatalf("parse staking for %s failed: %v", v.BlsKey, err)
		}
		totalStake.Add(totalStake, st)
	}
	fmt.Printf("total staking (all validators) = %s\n", totalStake.String())

	// 2) fetch block proof
	bp, err := fetchBlockProof(*rpcURL, *height)
	if err != nil {
		log.Fatalf("fetch block proof failed: %v", err)
	}
	fmt.Printf("block_proof_hash: %s\n", bp.BlockProofHash)
	fmt.Printf("aggregated_signature: %s\n", bp.BlsAggregatedSignature)
	fmt.Printf("signed keys count: %d\n", len(bp.SignedBlsKeys))

	// 3) ensure every signed key exists in validator info, and aggregate their stakes & pubkeys
	signedStakeSum := new(big.Int)
	var aggPub bls.PublicKey
	var aggPubInitialized bool

	for i, pk := range bp.SignedBlsKeys {
		key := strings.ToLower(trim0x(pk))
		v, ok := validatorMap[key]
		if !ok {
			log.Fatalf("signed bls key not found in validator info: %s, exit", pk)
			return
		}

		// parse stake and add
		st, err := parseStake(v.Staking)
		if err != nil {
			log.Fatalf("parse staking for signed key %s failed: %v", pk, err)
		}
		signedStakeSum.Add(signedStakeSum, st)

		// parse public key bytes
		pkBytes, err := hexToBytes(v.BlsKey)
		if err != nil {
			log.Fatalf("parse pubkey hex failed for %s: %v", v.BlsKey, err)
		}
		var pk bls.PublicKey
		if err := pk.Deserialize(pkBytes); err != nil {
			log.Fatalf("Deserialize pubkey failed for %s: %v", v.BlsKey, err)
		}

		if !aggPubInitialized {
			aggPub = pk
			aggPubInitialized = true
		} else {
			aggPub.Add(&pk)
		}

		fmt.Printf("signed[%d] %s stake=%s\n", i, key, st.String())
	}

	fmt.Printf("signed stake sum = %s\n", signedStakeSum.String())

	// 4) verify aggregate signature
	// parse signature
	sigBytes, err := hexToBytes(bp.BlsAggregatedSignature)
	if err != nil {
		log.Fatalf("parse aggregated signature hex failed: %v", err)
	}
	var aggSig bls.Sign
	if err := aggSig.Deserialize(sigBytes); err != nil {
		// if err := aggSig.Deserialize(sigBytes[4:]); err != nil {
		log.Fatalf("Deserialize aggregated signature failed: %v", err)
	}

	// parse message (block_proof_hash)
	msgBytes, err := hexToBytes(bp.BlockProofHash)
	if err != nil {
		log.Fatalf("parse block_proof_hash failed: %v", err)
	}

	// verify: aggregated pubkey vs signature on msg
	ok := aggSig.VerifyByte(&aggPub, msgBytes)
	if ok {
		fmt.Println("✅ BLS VerifySig Pass")
	} else {
		fmt.Println("❌ BLS VerifySig Failed")
	}

	// 5) check > 2/3 threshold
	// condition: signedStakeSum * 3 > totalStake * 2  (strictly greater than 2/3)
	left := new(big.Int).Mul(signedStakeSum, big.NewInt(3))
	right := new(big.Int).Mul(totalStake, big.NewInt(2))
	if left.Cmp(right) == 1 {
		fmt.Println("✅ Signed stake exceeds 2/3 total stake")
	} else {
		fmt.Println("❌ Signed stake is below 2/3 total stake")
	}

	// exit code: if both signature ok and >2/3 then success
	if ok && left.Cmp(right) == 1 {
		fmt.Printf("Block %s verify pass\n", *height)
		os.Exit(0)
	}
	os.Exit(2)
}
