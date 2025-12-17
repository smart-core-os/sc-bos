package main

import (
	csv2 "encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	historyconfig "github.com/smart-core-os/sc-bos/pkg/auto/history/config"
)

var (
	printNames = flag.Bool("print-names", false, "Print generated auto source names as a list")
	prefix     = flag.String("prefix", "bc-01/auto/history/", "Prefix for auto names")
)

func init() {
	flag.Parse()
}

// Doesn't write to file - just prints to stdout for you to copy manually
func main() {
	f, err := os.Open("devices.csv")

	if err != nil {
		panic(err)
	}

	csv := csv2.NewReader(f)

	records, err := csv.ReadAll()
	if err != nil {
		panic(err)
	}

	var autoCfg []historyconfig.Root

	var names []string

	for _, record := range records[1:] {
		lockerId := path.Base(record[0])
		name, err := url.JoinPath(*prefix, lockerId)
		if err != nil {
			panic(err)
		}
		autoCfg = append(autoCfg, historyconfig.Root{
			Config: auto.Config{
				Name: name,
				Type: "history",
			},
			Source: &historyconfig.Source{
				Name:  record[0],
				Trait: "smartcore.bos.Allocation",
			},
			Storage: &historyconfig.Storage{
				Type: "postgres",
			},
		})
		names = append(names, record[0])
	}

	a, err := json.MarshalIndent(autoCfg, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(a))

	if *printNames {
		fmt.Println(strings.Join(names, "\",\""))
	}
}
