package main

import (
	"fmt"
	"sort"

	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
)

func main() {
	bank, err := services.LoadBankTransactions("/Users/will/ITWill/JBS/jbs-internal-portal/docs/data/reconciliation-samples/Statement_1002_Feb_2026 (2).xls")
	if err != nil {
		fmt.Println("bank err:", err)
		return
	}
	fmt.Printf("Bank total transactions loaded: %d\n\n", len(bank))

	// Sort by date then amount
	sort.Slice(bank, func(i, j int) bool {
		if bank[i].Date.Equal(bank[j].Date) {
			return bank[i].Amount < bank[j].Amount
		}
		return bank[i].Date.Before(bank[j].Date)
	})

	fmt.Println("=== ALL bank transactions ===")
	for _, t := range bank {
		desc := t.Description
		if len(desc) > 50 {
			desc = desc[:50]
		}
		fmt.Printf("  %s  %9.2f  %s  [%s]\n",
			t.Date.Format("2006-01-02"), t.Amount, desc, t.CardmemberName)
	}
}

