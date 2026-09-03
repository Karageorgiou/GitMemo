package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/runethread/core/internal/memoryservice"
)

var (
	serviceCLIStdin  io.Reader = os.Stdin
	serviceCLIStdout io.Writer = os.Stdout
	serviceCLIStderr io.Writer = os.Stderr
)

type serviceCLIError struct {
	Error serviceCLIErrorBody `json:"error"`
}

type serviceCLIErrorBody struct {
	Code      string `json:"code"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

func runGet(args []string) int {
	fs := serviceFlagSet("get")
	root := fs.String("root", ".", "Runethread repository root")
	_ = fs.Bool("json", false, "emit machine-readable JSON (service commands are always JSON)")
	if err := fs.Parse(args); err != nil {
		return writeCLIArgumentError("get", err)
	}
	if fs.NArg() != 1 {
		return writeCLIArgumentError("get", errors.New("get requires exactly one memory UUID"))
	}
	svc, err := memoryservice.Open(*root)
	if err != nil {
		return writeCLIServiceError("get", err)
	}
	result, err := svc.Get(context.Background(), fs.Arg(0))
	if err != nil {
		return writeCLIServiceError("get", err)
	}
	return writeCLIResult("get", result)
}

func runPrepare(args []string) int {
	fs := serviceFlagSet("prepare")
	root := fs.String("root", ".", "Runethread repository root")
	_ = fs.Bool("json", false, "emit machine-readable JSON (service commands are always JSON)")
	requestPath := fs.String("request", "-", "JSON request file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return writeCLIArgumentError("prepare", err)
	}
	if fs.NArg() != 0 {
		return writeCLIArgumentError("prepare", errors.New("prepare accepts no positional arguments"))
	}
	var request memoryservice.PrepareMutationRequest
	if err := decodeCLIRequest(*requestPath, &request); err != nil {
		return writeCLIArgumentError("prepare", err)
	}
	svc, err := memoryservice.Open(*root)
	if err != nil {
		return writeCLIServiceError("prepare", err)
	}
	result, err := svc.PrepareMutation(context.Background(), request)
	if err != nil {
		return writeCLIServiceError("prepare", err)
	}
	return writeCLIResult("prepare", result)
}

func runApply(args []string) int {
	fs := serviceFlagSet("apply")
	root := fs.String("root", ".", "Runethread repository root")
	_ = fs.Bool("json", false, "emit machine-readable JSON (service commands are always JSON)")
	requestPath := fs.String("request", "-", "JSON request file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return writeCLIArgumentError("apply", err)
	}
	if fs.NArg() != 0 {
		return writeCLIArgumentError("apply", errors.New("apply accepts no positional arguments"))
	}
	var request memoryservice.ApplyMutationRequest
	if err := decodeCLIRequest(*requestPath, &request); err != nil {
		return writeCLIArgumentError("apply", err)
	}
	svc, err := memoryservice.Open(*root)
	if err != nil {
		return writeCLIServiceError("apply", err)
	}
	result, err := svc.ApplyMutation(context.Background(), request)
	if err != nil {
		return writeCLIServiceError("apply", err)
	}
	return writeCLIResult("apply", result)
}

func runWithdraw(args []string) int {
	fs := serviceFlagSet("withdraw")
	root := fs.String("root", ".", "Runethread repository root")
	_ = fs.Bool("json", false, "emit machine-readable JSON (service commands are always JSON)")
	requestPath := fs.String("request", "-", "JSON request file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return writeCLIArgumentError("withdraw", err)
	}
	if fs.NArg() != 0 {
		return writeCLIArgumentError("withdraw", errors.New("withdraw accepts no positional arguments"))
	}
	var request memoryservice.WithdrawRequest
	if err := decodeCLIRequest(*requestPath, &request); err != nil {
		return writeCLIArgumentError("withdraw", err)
	}
	svc, err := memoryservice.Open(*root)
	if err != nil {
		return writeCLIServiceError("withdraw", err)
	}
	result, err := svc.Withdraw(context.Background(), request)
	if err != nil {
		return writeCLIServiceError("withdraw", err)
	}
	return writeCLIResult("withdraw", result)
}

func runStatus(args []string) int {
	fs := serviceFlagSet("status")
	root := fs.String("root", ".", "Runethread repository root")
	_ = fs.Bool("json", false, "emit machine-readable JSON (service commands are always JSON)")
	if err := fs.Parse(args); err != nil {
		return writeCLIArgumentError("status", err)
	}
	if fs.NArg() != 0 {
		return writeCLIArgumentError("status", errors.New("status accepts no positional arguments"))
	}
	svc, err := memoryservice.Open(*root)
	if err != nil {
		return writeCLIServiceError("status", err)
	}
	result, err := svc.Status(context.Background())
	if err != nil {
		return writeCLIServiceError("status", err)
	}
	return writeCLIResult("status", result)
}

func serviceFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func decodeCLIRequest(path string, target any) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("--request must not be empty")
	}
	var (
		reader io.Reader
		file   *os.File
	)
	if path == "-" {
		reader = serviceCLIStdin
	} else {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return fmt.Errorf("open request %s: %w", path, err)
		}
		defer file.Close()
		reader = file
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON request: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON request must contain exactly one value")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return nil
}

func writeCLIResult(operation string, result any) int {
	encoder := json.NewEncoder(serviceCLIStdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		writeCLIError(serviceCLIErrorBody{Code: "output_error", Operation: operation, Message: err.Error()})
		return 1
	}
	return 0
}

func writeCLIArgumentError(operation string, err error) int {
	writeCLIError(serviceCLIErrorBody{Code: memoryservice.CodeInvalidArgument, Operation: operation, Message: err.Error()})
	return 2
}

func writeCLIServiceError(operation string, err error) int {
	var serviceErr *memoryservice.Error
	if errors.As(err, &serviceErr) {
		writeCLIError(serviceCLIErrorBody{Code: serviceErr.Code, Operation: serviceErr.Operation, Message: serviceErr.Message})
		return 1
	}
	writeCLIError(serviceCLIErrorBody{Code: memoryservice.CodeRepository, Operation: operation, Message: err.Error()})
	return 1
}

func writeCLIError(body serviceCLIErrorBody) {
	encoder := json.NewEncoder(serviceCLIStderr)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(serviceCLIError{Error: body})
}
