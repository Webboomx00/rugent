package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"rugent/functions"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mr-tron/base58"
)

// parseLookupTableData parses the raw lookup table data and returns a slice of addresses
func parseLookupTableData(data []byte) ([]solana.PublicKey, error) {
	// Skip discriminator and metadata (8 bytes + 56 bytes)
	if len(data) < 64 {
		return nil, fmt.Errorf("data too short for lookup table")
	}
	data = data[64:]

	addresses := make([]solana.PublicKey, 0)

	// Each address is 32 bytes
	for len(data) >= 32 {
		var address solana.PublicKey
		copy(address[:], data[:32])
		addresses = append(addresses, address)
		data = data[32:]
	}

	return addresses, nil
}

// PumpFunData represents the decoded instruction data
type PumpFunData struct {
	Amount     uint64
	MaxSolCost uint64
}

func decodePumpFunInstruction(encoded string) (*PumpFunData, error) {
	// Decode base58 string
	data, err := base58.Decode(encoded)
	if err != nil {
		fmt.Println("Eroor decoding PF data: ", err)
	}

	if len(data) < 16 {
		return nil, fmt.Errorf("data too short: expected at least 16 bytes, got %d", len(data))
	}

	// First 8 bytes for amount
	amount := binary.LittleEndian.Uint64(data[0:8])

	// Next 8 bytes for maxSolCost
	maxSolCost := binary.LittleEndian.Uint64(data[8:16])

	return &PumpFunData{
		Amount:     amount,
		MaxSolCost: maxSolCost,
	}, nil
}

func main() {
	var version uint64 = 0
	clientRPC := rpc.New("")
	// raydium tx 3FtEmTMEbmAv3h8QDxTtEdkobyt5YgYnT7hRBGTHSMjz8m8CQaMfEZ7b8SgFQGEsRTCg1HDgUZgRitEEhC2z2iF5
	// photon tx 2m3tp55FX3gPB1kCCcDqxPQ1sVLw4JBBQqcpJsZ9RxJShNne9PiExzKBmkXePmCM7FNgH1rBJQ5EN5cjz9UFYEAH
	signature := solana.MustSignatureFromBase58("3FtEmTMEbmAv3h8QDxTtEdkobyt5YgYnT7hRBGTHSMjz8m8CQaMfEZ7b8SgFQGEsRTCg1HDgUZgRitEEhC2z2iF5")

	tx, errTx := clientRPC.GetTransaction(
		context.TODO(),
		signature,
		&rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			MaxSupportedTransactionVersion: &version,
			Commitment:                     rpc.CommitmentConfirmed,
		},
	)

	if errTx != nil {
		fmt.Println("Error getting transaction:", errTx)
		return
	}

	decoded, err := tx.Transaction.GetTransaction()
	if err != nil {
		fmt.Printf("Error decoding transaction: %v\n", err)
		return
	}

	// Get the complete list of accounts including looked-up addresses
	var completeAccountList []solana.PublicKey

	// First, add all static account keys
	completeAccountList = append(completeAccountList, decoded.Message.AccountKeys...)

	// Then, add all looked-up accounts if available
	if decoded.Message.AddressTableLookups != nil {
		fmt.Println("Address Table Lookups:")
		for _, lookup := range decoded.Message.AddressTableLookups {
			fmt.Printf("Looking up table: %s\n", lookup.AccountKey)

			// Get the lookup table account info
			tableAccount, err := clientRPC.GetAccountInfoWithOpts(
				context.TODO(),
				lookup.AccountKey,
				&rpc.GetAccountInfoOpts{
					Encoding: solana.EncodingBase64,
				},
			)
			if err != nil {
				fmt.Printf("Error fetching lookup table: %v\n", err)
				continue
			}

			// Get the data as bytes
			tableData := tableAccount.Value.Data.GetBinary()

			// Parse the addresses from the lookup table
			addresses, err := parseLookupTableData(tableData)
			if err != nil {
				fmt.Printf("Error parsing lookup table: %v\n", err)
				continue
			}

			// Add writable addresses
			for _, idx := range lookup.WritableIndexes {
				if int(idx) < len(addresses) {
					completeAccountList = append(completeAccountList, addresses[idx])
				}
			}

			// Add readonly addresses
			for _, idx := range lookup.ReadonlyIndexes {
				if int(idx) < len(addresses) {
					completeAccountList = append(completeAccountList, addresses[idx])
				}
			}

			// fmt.Printf("Found %d addresses in lookup table\n", len(addresses))
		}
	}
	// spew.Dump(tx.Meta.InnerInstructions)
	for _, inner := range tx.Meta.InnerInstructions {
		fmt.Printf("\nInner Instructions for Index %d:\n", inner.Index)

		for i, inst := range inner.Instructions {
			// Bounds check for program ID
			if inst.ProgramIDIndex >= uint16(len(completeAccountList)) {
				fmt.Printf("Program ID index %d out of bounds\n", inst.ProgramIDIndex)
				continue
			}

			programID := completeAccountList[inst.ProgramIDIndex]
			fmt.Println(programID)

			// radyium uses solana token progID, pumpfun would use PF prog id 6EF8..
			if programID == solana.TokenProgramID {
				functions.DecodeRaydiumInstruction(i, inst, programID, completeAccountList)
			}
		}
	}

}
