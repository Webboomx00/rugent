package functions

import (
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/gagliardetto/solana-go"
)

type PumpFunData struct {
	Amount     uint64 `json:"amount"`
	MaxSolCost uint64 `json:"maxSolCost"`
}

func DecodePumpFunInstruction(encoded string) (*PumpFunData, error) {
	// Decode base58 string
	data := base58.Decode(encoded)

	if len(data) < 24 {
		return nil, fmt.Errorf("data too short: expected at least 24 bytes, got %d", len(data))
	}

	// Amount is at offset 8 for 8 bytes (same as hex version)
	amount := binary.LittleEndian.Uint64(data[8:16])

	// MaxSolCost is at offset 16 for 8 bytes (same as hex version)
	maxSolCost := binary.LittleEndian.Uint64(data[16:24])

	return &PumpFunData{
		Amount:     amount,
		MaxSolCost: maxSolCost,
	}, nil
}

func DecodeRaydiumInstruction(i int, instruction solana.CompiledInstruction, programId solana.PublicKey, completeAccountList []solana.PublicKey) {
	data := instruction.Data
	if len(data) > 0 {
		discriminator := data[0]
		if discriminator == 3 && len(data) >= 9 {
			amount := binary.LittleEndian.Uint64(data[1:9])

			fmt.Printf("\nTransfer #%d:\n", i+1)

			// Safely get account addresses with bounds checking
			if len(instruction.Accounts) >= 3 {
				for j, accountIdx := range instruction.Accounts {
					if accountIdx >= uint16(len(completeAccountList)) {
						fmt.Printf("Account index %d out of bounds\n", accountIdx)
						continue
					}

					account := completeAccountList[accountIdx]
					switch j {
					case 0:
						fmt.Printf("From: %s\n", account)
					case 1:
						fmt.Printf("To: %s\n", account)
					case 2:
						fmt.Printf("Authority: %s\n", account)
					}
				}
			}
			fmt.Printf("Amount: %d (raw amount, divide by decimals for actual token amount)\n", amount)
		} else {
			fmt.Printf("\nNon-transfer instruction #%d:\n", i+1)
			fmt.Printf("Program: %s\n", programId)
			fmt.Printf("Discriminator: %d\n", discriminator)
			fmt.Printf("Raw data: %v\n", data)

			fmt.Printf("Accounts involved:\n")
			for _, accountIdx := range instruction.Accounts {
				if accountIdx < uint16(len(completeAccountList)) {
					fmt.Printf("- %s\n", completeAccountList[accountIdx])
				} else {
					fmt.Printf("Account index %d out of bounds\n", accountIdx)
				}
			}
		}
	}
}
