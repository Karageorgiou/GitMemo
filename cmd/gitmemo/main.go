package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Karageorgiou/GitMemo/internal/indexer"
	"github.com/Karageorgiou/GitMemo/internal/starter"
	"github.com/Karageorgiou/GitMemo/internal/validation"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "index":
		return runIndex(args[1:])
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := "."
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "init accepts at most one target directory")
		return 2
	}
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	if err := starter.Init(root); err != nil {
		fmt.Fprintln(os.Stderr, "init failed:", err)
		return 1
	}
	fmt.Printf("Initialized GitMemo memory repository at %s\n", root)
	return 0
}

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := "."
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "validate accepts at most one repository root")
		return 2
	}
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	issues := validation.Validate(root)
	if *asJSON {
		data, _ := validation.MarshalJSONReport(root, issues)
		fmt.Println(string(data))
	} else {
		fmt.Println(validation.RenderText(issues))
	}
	if validation.HasErrors(issues) {
		return 1
	}
	return 0
}

func runIndex(args []string) int {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	check := fs.Bool("check", false, "fail if committed indexes are stale")
	write := fs.Bool("write", false, "regenerate committed indexes")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *check == *write {
		fmt.Fprintln(os.Stderr, "index requires exactly one of --check or --write")
		return 2
	}
	root := "."
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "index accepts at most one repository root")
		return 2
	}
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	if *write {
		if err := indexer.Write(root); err != nil {
			fmt.Fprintln(os.Stderr, "index write failed:", err)
			return 1
		}
		fmt.Println("GitMemo indexes regenerated.")
		return 0
	}
	stale, err := indexer.Check(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "index check failed:", err)
		return 1
	}
	if len(stale) > 0 {
		fmt.Fprintln(os.Stderr, "GitMemo indexes are stale:")
		for _, p := range stale {
			fmt.Fprintln(os.Stderr, "-", p)
		}
		return 1
	}
	fmt.Println("GitMemo indexes are current.")
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "GitMemo repository tooling\n\nUsage:\n  gitmemo init [dir]\n  gitmemo validate [--json] [root]\n  gitmemo index --check [root]\n  gitmemo index --write [root]")
}
