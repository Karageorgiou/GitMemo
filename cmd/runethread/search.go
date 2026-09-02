package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Karageorgiou/GitMemo/internal/indexer"
)

func runSearch(args []string) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "Runethread repository root")
	limit := fs.Int("limit", 10, "maximum number of results")
	asJSON := fs.Bool("json", false, "emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "search requires a query or exact memory UUID")
		return 2
	}
	if *limit < 1 || *limit > 100 {
		fmt.Fprintln(os.Stderr, "search --limit must be between 1 and 100")
		return 2
	}

	query := strings.Join(fs.Args(), " ")
	results, err := indexer.Search(*root, query, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "search failed:", err)
		return 1
	}
	if *asJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "search output failed:", err)
			return 1
		}
		fmt.Println(string(data))
		return 0
	}
	if len(results) == 0 {
		fmt.Println("No indexed memories matched.")
		return 0
	}
	for _, result := range results {
		fmt.Printf("%s  %s  %s/%s  %s\n", result.ID, scoreLabel(result.Score), result.Type, result.Lifecycle, result.Title)
		fmt.Printf("  %s\n", result.ContentPath)
		if len(result.MatchedTerms) > 0 {
			fmt.Printf("  matched: %s\n", strings.Join(result.MatchedTerms, ", "))
		}
		if result.Summary != "" {
			fmt.Printf("  %s\n", result.Summary)
		}
	}
	return 0
}

func scoreLabel(score int) string {
	if score == 0 {
		return "exact"
	}
	return fmt.Sprintf("score=%d", score)
}
