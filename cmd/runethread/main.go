package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/starter"
	"github.com/runethread/core/internal/trust"
	"github.com/runethread/core/internal/upgrader"
	"github.com/runethread/core/internal/validation"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		if isInteractive(os.Stdin) {
			return runWizard()
		}
		usage()
		return 2
	}
	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "index":
		return runIndex(args[1:])
	case "search":
		return runSearch(args[1:])
	case "get":
		return runGet(args[1:])
	case "prepare":
		return runPrepare(args[1:])
	case "apply":
		return runApply(args[1:])
	case "withdraw":
		return runWithdraw(args[1:])
	case "status":
		return runStatus(args[1:])
	case "trust":
		return runTrust(args[1:])
	case "version":
		fmt.Println(buildinfo.ReleaseVersion)
		return 0
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
	fmt.Printf("Initialized Runethread memory repository at %s\n", root)
	return 0
}

func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := "."
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "upgrade accepts at most one repository root")
		return 2
	}
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	result, err := upgrader.Apply(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "upgrade failed:", err)
		return 1
	}
	if result.AlreadyCurrent {
		fmt.Printf("Runethread repository is already current for %s (contract %d).\n", result.ToVersion, result.ToContract)
		return 0
	}
	fmt.Printf("Upgraded Runethread repository from %s / contract %d to %s / contract %d.\n", result.FromVersion, result.FromContract, result.ToVersion, result.ToContract)
	if len(result.ChangedPaths) > 0 {
		fmt.Println("Changed managed/generated paths:")
		for _, path := range result.ChangedPaths {
			fmt.Println("-", path)
		}
	}
	fmt.Println("Repository validation passed after upgrade.")
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
	markStale := fs.Bool("mark-stale", false, "mark generated indexes as potentially incomplete")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	selected := 0
	for _, value := range []bool{*check, *write, *markStale} {
		if value {
			selected++
		}
	}
	if selected != 1 {
		fmt.Fprintln(os.Stderr, "index requires exactly one of --check, --write, or --mark-stale")
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
	if *markStale {
		if err := indexer.MarkStale(root); err != nil {
			fmt.Fprintln(os.Stderr, "index stale marker failed:", err)
			return 1
		}
		fmt.Println("Runethread indexes marked stale; canonical memories remain authoritative.")
		return 0
	}
	if *write {
		if err := indexer.Write(root); err != nil {
			fmt.Fprintln(os.Stderr, "index write failed:", err)
			return 1
		}
		fmt.Println("Runethread indexes regenerated.")
		return 0
	}
	stale, err := indexer.Check(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "index check failed:", err)
		return 1
	}
	if len(stale) > 0 {
		fmt.Fprintln(os.Stderr, "Runethread indexes are stale:")
		for _, p := range stale {
			fmt.Fprintln(os.Stderr, "-", p)
		}
		return 1
	}
	fmt.Println("Runethread indexes are current.")
	return 0
}

func runTrust(args []string) int {
	if len(args) == 0 || args[0] != "version" {
		fmt.Fprintln(os.Stderr, "trust requires subcommand: trust version [root]")
		return 2
	}
	if len(args) > 2 {
		fmt.Fprintln(os.Stderr, "trust version accepts at most one repository root")
		return 2
	}
	root := "."
	if len(args) == 2 {
		root = args[1]
	}
	version, err := trust.ReadPinnedVersion(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "trust version failed:", err)
		return 1
	}
	fmt.Println(version)
	return 0
}

func runWizard() int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Runethread %s\n", buildinfo.ReleaseVersion)
	fmt.Println("Create a private, user-owned memory repository for your AI assistant.")
	fmt.Print("\nDirectory [runethread-memory]: ")
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		fmt.Fprintln(os.Stderr, "could not read directory:", err)
		return 1
	}
	target := strings.TrimSpace(line)
	if target == "" {
		target = "runethread-memory"
	}
	if err := starter.Init(target); err != nil {
		fmt.Fprintln(os.Stderr, "init failed:", err)
		waitForEnter(reader)
		return 1
	}

	gitReady := initializeGit(target)
	fmt.Printf("\nCreated Runethread memory repository at %s.\n", target)
	if gitReady {
		fmt.Println("Initialized a local Git repository on branch main.")
	} else {
		fmt.Println("Git was not initialized automatically. Install Git and run `git init -b main` inside the new directory.")
	}
	fmt.Println("\nNext:")
	fmt.Println("1. Create an empty PRIVATE repository on GitHub/GitLab/etc. and push this directory to it.")
	fmt.Println("2. Give your AI assistant access to that private repository.")
	fmt.Println("3. In any chat, use `Runethread: store ...` to write and `Runethread: search ...` to retrieve.")
	fmt.Println("\nThe repository itself contains the full operating instructions for the assistant.")
	waitForEnter(reader)
	return 0
}

func initializeGit(root string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.Command("git", "-C", root, "init", "-b", "main")
	if err := cmd.Run(); err == nil {
		return true
	}
	cmd = exec.Command("git", "-C", root, "init")
	return cmd.Run() == nil
}

func waitForEnter(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to close...")
	_, _ = reader.ReadString('\n')
}

func isInteractive(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "Runethread repository tooling\n\nUsage:\n  runethread init [dir]\n  runethread upgrade [root]\n  runethread validate [--json] [root]\n  runethread search [--root DIR] [--limit N] [--json] <query-or-uuid>\n  runethread get [--root DIR] <uuid>\n  runethread prepare [--root DIR] [--request FILE|-]\n  runethread apply [--root DIR] [--request FILE|-]\n  runethread withdraw [--root DIR] [--request FILE|-]\n  runethread status [--root DIR]\n  runethread index --check [root]\n  runethread index --write [root]\n  runethread index --mark-stale [root]\n  runethread trust version [root]\n  runethread version")
}
