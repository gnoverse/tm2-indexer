package db

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/std"
)

type Validator struct {
	Addr string `db:"validator_addr"`
	Name string `db:"validator_name"`
}

type Block struct {
	Height int64     `db:"height"`
	Time   time.Time `db:"time"`

	Hash string `db:"hash"`

	NumTXS     int64  `db:"num_txs"`
	TotalTXS   int64  `db:"total_txs"`
	AppVersion string `db:"app_version"`

	DataHash           string `db:"data_hash"`
	LastCommitHash     string `db:"last_commit_hash"`
	ValidatorsHash     string `db:"validators_hash"`
	NextValidatorsHash string `db:"next_validators_hash"`
	ConsensusHash      string `db:"consensus_hash"`
	AppHash            string `db:"app_hash"`
	LastResultsHash    string `db:"last_results_hash"`

	ProposerAddress string `db:"proposer_address"`

	Signatures   []BlockSignature `db:"-"`
	Transactions []Transaction    `db:"-"`
}

type BlockSignature struct {
	BlockHeight   int64  `db:"block_height"`
	ValidatorAddr string `db:"validator_addr"`
	Signed        bool   `db:"signed"`
}

type Transaction struct {
	Height int64 `db:"height"`
	Index  int   `db:"tx_index"`

	Hash string `db:"hash"`
	// Success   bool   `db:"success"`
	GasWanted int64 `db:"gas_wanted"`
	// GasUsed   int64  `db:"gas_used"`
	GasFee int64 `db:"gas_fee"`
	// Content   string `db:"content"`
	Memo string `db:"memo"`

	Messages []Message `db:"-"`
}

type Message struct {
	Height int64  `db:"height"`
	TxHash string `db:"tx_hash"`
	Index  int64  `db:"index"`

	Route  string          `db:"route"`
	Type   string          `db:"type"`
	MsgRaw json.RawMessage `db:"msg_raw"`
}

func NewBlock(block *types.Block, validatorAddrName map[string]string) (*Block, error) {
	signed := make(map[string]bool, len(validatorAddrName))
	for k := range validatorAddrName {
		signed[k] = false
	}

	b := &Block{
		Height:     block.GetHeight(),
		Time:       block.GetTime(),
		Hash:       base64.StdEncoding.EncodeToString(block.Hash()),
		NumTXS:     block.NumTxs,
		TotalTXS:   block.TotalTxs,
		AppVersion: block.AppVersion,

		DataHash:           base64.RawStdEncoding.EncodeToString(block.DataHash),
		LastCommitHash:     base64.RawStdEncoding.EncodeToString(block.LastCommitHash),
		LastResultsHash:    base64.RawStdEncoding.EncodeToString(block.LastResultsHash),
		NextValidatorsHash: base64.RawStdEncoding.EncodeToString(block.NextValidatorsHash),
		ConsensusHash:      base64.RawStdEncoding.EncodeToString(block.ConsensusHash),
		AppHash:            base64.RawStdEncoding.EncodeToString(block.AppHash),

		ProposerAddress: block.ProposerAddress.Bech32().String(),

		Signatures:   make([]BlockSignature, 0, len(signed)),
		Transactions: make([]Transaction, block.NumTxs),
	}

	if _, exist := validatorAddrName[b.ProposerAddress]; !exist {
		return b, fmt.Errorf("block proposer is an unknown validator")
	}

	if len(block.LastCommit.Precommits) != len(validatorAddrName) {
		return b, fmt.Errorf("missing validators informations %d/%d", len(block.LastCommit.Precommits), len(validatorAddrName))
	}
	// Parse signatures
	for _, v := range block.LastCommit.Precommits {
		if v != nil {
			vAddr := v.ValidatorAddress.Bech32().String()
			b.Signatures = append(b.Signatures, BlockSignature{
				BlockHeight:   block.GetHeight(),
				ValidatorAddr: vAddr,
				Signed:        true,
			})
			signed[vAddr] = true
		}
	}

	for vAddr, signed := range signed {
		if !signed {
			b.Signatures = append(b.Signatures, BlockSignature{
				BlockHeight:   block.GetHeight(),
				ValidatorAddr: vAddr,
				Signed:        false,
			})
		}
	}

	// Parse transactions
	for i, tx := range block.Txs {
		var stdTx std.Tx
		err := amino.Unmarshal(tx, &stdTx)
		if err != nil {
			return b, err
		}

		b.Transactions[i].Height = block.GetHeight()
		b.Transactions[i].Hash = base64.RawStdEncoding.EncodeToString(tx.Hash())
		b.Transactions[i].Index = i
		b.Transactions[i].GasFee = stdTx.Fee.GasFee.Amount
		b.Transactions[i].GasWanted = stdTx.Fee.GasWanted
		b.Transactions[i].Memo = stdTx.Memo

		b.Transactions[i].Messages = make([]Message, len(stdTx.Msgs))
		for i2, msg := range stdTx.Msgs {
			b.Transactions[i].Messages[i2].Height = block.GetHeight()
			b.Transactions[i].Messages[i2].TxHash = b.Transactions[i].Hash
			b.Transactions[i].Messages[i2].Route = msg.Route()
			b.Transactions[i].Messages[i2].Type = msg.Type()
			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				return b, err
			}

			b.Transactions[i].Messages[i2].MsgRaw = jsonMsg

		}
	}

	return b, nil
}
