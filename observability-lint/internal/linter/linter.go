// Package linter implements the linter analyzer.
package linter

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/ettle/strcase"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Options are options for the linter analyzer.
type Options struct {
	NoMixedArgs    bool     // Enforce not mixing key-value pairs and attributes (default true).
	KVOnly         bool     // Enforce using key-value pairs only (overrides NoMixedArgs, incompatible with AttrOnly).
	AttrOnly       bool     // Enforce using attributes only (overrides NoMixedArgs, incompatible with KVOnly).
	NoGlobal       string   // Enforce not using global loggers ("all" or "default").
	ContextOnly    string   // Enforce using methods that accept a context ("all" or "scope").
	StaticMsg      bool     // Enforce using static log messages.
	NoRawKeys      bool     // Enforce using constants instead of raw keys.
	KeyNamingCase  string   // Enforce a single key naming convention ("snake", "kebab", "camel", or "pascal").
	ForbiddenKeys  []string // Enforce not using specific keys.
	ArgsOnSepLines bool     // Enforce putting arguments on separate lines.
	SkipGenerated  bool     // Enforce skipping generated files
	SkipTest       bool     // Enforce skipping test files
}

// New creates a new linter analyzer.
func New(opts *Options) *analysis.Analyzer {
	if opts == nil {
		opts = &Options{NoMixedArgs: true}
	}
	return &analysis.Analyzer{
		Name:     "linter",
		Doc:      "ensure consistent code style when using log/slog",
		Flags:    flags(opts),
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			if opts.KVOnly && opts.AttrOnly {
				return nil, fmt.Errorf("linter: Options.KVOnly and Options.AttrOnly: %w", errIncompatible)
			}

			switch opts.NoGlobal {
			case "", "all", "default":
			default:
				return nil, fmt.Errorf("linter: Options.NoGlobal=%s: %w", opts.NoGlobal, errInvalidValue)
			}

			switch opts.ContextOnly {
			case "", "all", "scope":
			default:
				return nil, fmt.Errorf("linter: Options.ContextOnly=%s: %w", opts.ContextOnly, errInvalidValue)
			}

			switch opts.KeyNamingCase {
			case "", snakeCase, kebabCase, camelCase, pascalCase:
			default:
				return nil, fmt.Errorf("linter: Options.KeyNamingCase=%s: %w", opts.KeyNamingCase, errInvalidValue)
			}
			preRun(pass, opts)
			run(pass, opts)
			return nil, nil
		},
	}
}

var (
	errIncompatible = errors.New("incompatible options")
	errInvalidValue = errors.New("invalid value")

	testSuffixes      = []string{"_test.go"}
	generatedSuffixes = []string{"_generated.go", "_gen.go", ".gen.go", ".pb.go", ".pb.gw.go"}
)

func flags(opts *Options) flag.FlagSet {
	fset := flag.NewFlagSet("linter", flag.ContinueOnError)

	boolVar := func(value *bool, name, usage string) {
		fset.Func(name, usage, func(s string) error {
			v, err := strconv.ParseBool(s)
			*value = v
			return err
		})
	}

	strVar := func(value *string, name, usage string) {
		fset.Func(name, usage, func(s string) error {
			*value = s
			return nil
		})
	}

	boolVar(&opts.NoMixedArgs, "no-mixed-args", "enforce not mixing key-value pairs and attributes (default true)")
	boolVar(&opts.KVOnly, "kv-only", "enforce using key-value pairs only (overrides -no-mixed-args, incompatible with -attr-only)")
	boolVar(&opts.AttrOnly, "attr-only", "enforce using attributes only (overrides -no-mixed-args, incompatible with -kv-only)")
	strVar(&opts.ContextOnly, "context-only", "enforce using methods that accept a context (all|scope)")
	boolVar(&opts.StaticMsg, "static-msg", "enforce using static log messages")
	boolVar(&opts.NoRawKeys, "no-raw-keys", "enforce using constants instead of raw keys")
	strVar(&opts.KeyNamingCase, "key-naming-case", "enforce a single key naming convention (snake|kebab|camel|pascal)")
	boolVar(&opts.ArgsOnSepLines, "args-on-sep-lines", "enforce putting arguments on separate lines")
	boolVar(&opts.SkipGenerated, "skip-generated", "enforce skipping generated files")
	boolVar(&opts.SkipTest, "skip-test", "enforce skipping test files")

	fset.Func("forbidden-keys", "enforce not using specific keys (comma-separated)", func(s string) error {
		opts.ForbiddenKeys = append(opts.ForbiddenKeys, strings.Split(s, ",")...)
		return nil
	})

	return *fset
}

const (
	forbiddenLogFuncMessage   = "should be used only observability/logger module"
	forbiddenErrorFuncMessage = "should be used only observability/errors module"
)

var forbiddenFuncs = map[string]string{
	"log/slog.With":                   forbiddenLogFuncMessage,
	"log/slog.Log":                    forbiddenLogFuncMessage,
	"log/slog.LogAttrs":               forbiddenLogFuncMessage,
	"log/slog.Debug":                  forbiddenLogFuncMessage,
	"log/slog.Info":                   forbiddenLogFuncMessage,
	"log/slog.Warn":                   forbiddenLogFuncMessage,
	"log/slog.Error":                  forbiddenLogFuncMessage,
	"log/slog.DebugContext":           forbiddenLogFuncMessage,
	"log/slog.InfoContext":            forbiddenLogFuncMessage,
	"log/slog.WarnContext":            forbiddenLogFuncMessage,
	"log/slog.ErrorContext":           forbiddenLogFuncMessage,
	"(*log/slog.Logger).With":         forbiddenLogFuncMessage,
	"(*log/slog.Logger).Log":          forbiddenLogFuncMessage,
	"(*log/slog.Logger).LogAttrs":     forbiddenLogFuncMessage,
	"(*log/slog.Logger).Debug":        forbiddenLogFuncMessage,
	"(*log/slog.Logger).Info":         forbiddenLogFuncMessage,
	"(*log/slog.Logger).Warn":         forbiddenLogFuncMessage,
	"(*log/slog.Logger).Error":        forbiddenLogFuncMessage,
	"(*log/slog.Logger).DebugContext": forbiddenLogFuncMessage,
	"(*log/slog.Logger).InfoContext":  forbiddenLogFuncMessage,
	"(*log/slog.Logger).WarnContext":  forbiddenLogFuncMessage,
	"(*log/slog.Logger).ErrorContext": forbiddenLogFuncMessage,
	"errors.New":                      forbiddenErrorFuncMessage,
	"fmt.Errorf":                      forbiddenErrorFuncMessage,
}

var slogFuncs = map[string]struct {
	argsPos          int
	skipContextCheck bool
	msgPos           int
}{
	"github.com/MikL9/observability/logger.Debug": {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger.Info":  {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger.Warn":  {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger.Error": {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger.Fatal": {msgPos: 1, argsPos: 2},
	//"github.com/MikL9/observability/logger.Recovery":              {msgPos: 1, argsPos: 2}, // TODO придумать другую проверку Recovery
	"github.com/MikL9/observability/logger.HandleError":           {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/storage.SetContext":           {msgPos: 1, argsPos: 1},
	"github.com/MikL9/observability/logger/errors.New":            {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger/errors.NewConstError":  {msgPos: 0, argsPos: 1, skipContextCheck: true},
	"github.com/MikL9/observability/logger/errors.Wrap":           {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger/errors.WrapSkip`":      {msgPos: 1, argsPos: 2},
	"github.com/MikL9/observability/logger/errors.WrapPrefix":     {msgPos: 1, argsPos: 3},
	"github.com/MikL9/observability/logger/errors.WrapPrefixSkip": {msgPos: 1, argsPos: 4},
}

var attrFuncs = map[string]struct{}{
	"log/slog.String":   {},
	"log/slog.Int64":    {},
	"log/slog.Int":      {},
	"log/slog.Uint64":   {},
	"log/slog.Float64":  {},
	"log/slog.Bool":     {},
	"log/slog.Time":     {},
	"log/slog.Duration": {},
	"log/slog.Group":    {},
	"log/slog.Any":      {},
}

const (
	snakeCase  = "snake"
	kebabCase  = "kebab"
	camelCase  = "camel"
	pascalCase = "pascal"
)

func preRun(pass *analysis.Pass, opts *Options) {
	visitor := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	filter := []ast.Node{
		(*ast.File)(nil),
		(*ast.StructType)(nil),
		(*ast.GenDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	visitor.Preorder(filter, func(node ast.Node) {
		fn := pass.Fset.File(node.Pos()).Name()

		if opts.SkipTest && hasSuffixes(&pass.IgnoredFiles, fn, testSuffixes) {
			return
		}

		if opts.SkipGenerated && hasSuffixes(&pass.IgnoredFiles, fn, generatedSuffixes) {
			return
		}

		if f, ok := node.(*ast.File); ok {
			if opts.SkipGenerated && hasGeneratedComment(&pass.IgnoredFiles, fn, f) {
				return
			}
		}
	})
}

func run(pass *analysis.Pass, opts *Options) {
	visitor := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	filter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	visitor.Preorder(filter, func(node ast.Node) {
		fn := pass.Fset.File(node.Pos()).Name()
		if slices.Contains(pass.IgnoredFiles, fn) {
			return
		}
		visit(pass, opts, node, nil)
	})
}

// NOTE: stack is nil if Preorder is used.
func visit(pass *analysis.Pass, opts *Options, node ast.Node, stack []ast.Node) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}

	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	if fn == nil {
		return
	}

	name := fn.FullName()
	if errMsg, ok := forbiddenFuncs[name]; ok {
		pass.Reportf(call.Pos(), errMsg)
		return
	}
	funcInfo, ok := slogFuncs[name]
	if !ok {
		return
	}

	// NOTE: "With" functions are not checked for context.Context.
	if !funcInfo.skipContextCheck {
		switch opts.ContextOnly {
		case "all":
			typ := pass.TypesInfo.TypeOf(call.Args[0])
			if typ != nil && typ.String() != "context.Context" {
				pass.Reportf(call.Pos(), "context should not be nil")
			}
		case "scope":
			typ := pass.TypesInfo.TypeOf(call.Args[0])
			if typ != nil && typ.String() != "context.Context" && isContextInScope(pass.TypesInfo, stack) {
				pass.Reportf(call.Pos(), "context should not be nil")
			}
		}
	}

	//msgPos := funcInfo.argsPos - 1
	// NOTE: "With" functions have no message argument and must be skipped.
	if opts.StaticMsg && !isStaticMsg(pass, call.Args[funcInfo.msgPos]) {
		pass.Reportf(call.Pos(), "message should be a string literal or a constant")
	}

	// NOTE: we assume that the arguments have already been validated by govet.
	args := call.Args[funcInfo.argsPos:]
	if len(args) == 0 {
		return
	}

	var keys []ast.Expr
	var attrs []ast.Expr

	for i := 0; i < len(args); i++ {
		typ := pass.TypesInfo.TypeOf(args[i])
		if typ == nil {
			continue
		}

		switch typ.String() {
		case "string":
			keys = append(keys, args[i])
			i++ // skip the value.
		case "log/slog.Attr":
			attrs = append(attrs, args[i])
		case "[]any", "[]log/slog.Attr":
			continue // the last argument may be an unpacked slice, skip it.
		}
	}

	switch {
	case opts.KVOnly && len(attrs) > 0:
		pass.Reportf(call.Pos(), "attributes should not be used")
	case opts.AttrOnly && len(keys) > 0:
		pass.Reportf(call.Pos(), "key-value pairs should not be used")
	case opts.NoMixedArgs && len(attrs) > 0 && len(keys) > 0:
		pass.Reportf(call.Pos(), "key-value pairs and attributes should not be mixed")
	}

	if opts.NoRawKeys {
		forEachKey(pass.TypesInfo, keys, attrs, func(key ast.Expr) {
			if ident, ok := key.(*ast.Ident); !ok || ident.Obj == nil || ident.Obj.Kind != ast.Con {
				pass.Reportf(call.Pos(), "raw keys should not be used")
			}
		})
	}

	checkKeyNamingCase := func(caseFn func(string) string, caseName string) {
		forEachKey(pass.TypesInfo, keys, attrs, func(key ast.Expr) {
			if name, ok := getKeyName(key); ok && name != caseFn(name) {
				pass.Reportf(call.Pos(), "keys should be written in %s", caseName)
			}
		})
	}

	switch opts.KeyNamingCase {
	case snakeCase:
		checkKeyNamingCase(strcase.ToSnake, "snake_case")
	case kebabCase:
		checkKeyNamingCase(strcase.ToKebab, "kebab-case")
	case camelCase:
		checkKeyNamingCase(strcase.ToCamel, "camelCase")
	case pascalCase:
		checkKeyNamingCase(strcase.ToPascal, "PascalCase")
	}

	if len(opts.ForbiddenKeys) > 0 {
		forEachKey(pass.TypesInfo, keys, attrs, func(key ast.Expr) {
			if name, ok := getKeyName(key); ok && slices.Contains(opts.ForbiddenKeys, name) {
				pass.Reportf(call.Pos(), "%q key is forbidden and should not be used", name)
			}
		})
	}

	if opts.ArgsOnSepLines && areArgsOnSameLine(pass.Fset, call, keys, attrs) {
		pass.Reportf(call.Pos(), "arguments should be put on separate lines")
	}
}

func isContextInScope(info *types.Info, stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		decl, ok := stack[i].(*ast.FuncDecl)
		if !ok {
			continue
		}
		params := decl.Type.Params
		if len(params.List) == 0 || len(params.List[0].Names) == 0 {
			continue
		}
		typ := info.TypeOf(params.List[0].Names[0])
		if typ != nil && typ.String() == "context.Context" {
			return true
		}
	}
	return false
}

func isStaticMsg(pass *analysis.Pass, msg ast.Expr) bool {
	if pass.TypesInfo.TypeOf(msg).String() == "error" {
		return true
	}
	switch msg := msg.(type) {
	case *ast.BasicLit: // e.g. slog.Info("msg")
		return msg.Kind == token.STRING
	case *ast.Ident: // e.g. const msg = "msg"; slog.Info(msg)
		return msg.Obj != nil && msg.Obj.Kind == ast.Con
	default:
		return false
	}
}

func forEachKey(info *types.Info, keys, attrs []ast.Expr, fn func(key ast.Expr)) {
	for _, key := range keys {
		fn(key)
	}

	for _, attr := range attrs {
		switch attr := attr.(type) {
		case *ast.CallExpr: // e.g. slog.Int()
			callee := typeutil.StaticCallee(info, attr)
			if callee == nil {
				continue
			}
			if _, ok := attrFuncs[callee.FullName()]; !ok {
				continue
			}
			fn(attr.Args[0])

		case *ast.CompositeLit: // slog.Attr{}
			switch len(attr.Elts) {
			case 1: // slog.Attr{Key: ...} | slog.Attr{Value: ...}
				if kv := attr.Elts[0].(*ast.KeyValueExpr); kv.Key.(*ast.Ident).Name == "Key" {
					fn(kv.Value)
				}
			case 2: // slog.Attr{Key: ..., Value: ...} | slog.Attr{Value: ..., Key: ...} | slog.Attr{..., ...}
				if kv, ok := attr.Elts[0].(*ast.KeyValueExpr); ok && kv.Key.(*ast.Ident).Name == "Key" {
					fn(kv.Value)
				} else if kv, ok := attr.Elts[1].(*ast.KeyValueExpr); ok && kv.Key.(*ast.Ident).Name == "Key" {
					fn(kv.Value)
				} else {
					fn(attr.Elts[0])
				}
			}
		}
	}
}

func getKeyName(key ast.Expr) (string, bool) {
	if ident, ok := key.(*ast.Ident); ok {
		if ident.Obj == nil || ident.Obj.Decl == nil || ident.Obj.Kind != ast.Con {
			return "", false
		}
		if spec, ok := ident.Obj.Decl.(*ast.ValueSpec); ok && len(spec.Values) > 0 {
			// TODO: support len(spec.Values) > 1; e.g. const foo, bar = 1, 2
			key = spec.Values[0]
		}
	}
	if lit, ok := key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		// string literals are always quoted.
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			panic("unreachable")
		}
		return value, true
	}
	return "", false
}

func areArgsOnSameLine(fset *token.FileSet, call ast.Expr, keys, attrs []ast.Expr) bool {
	if len(keys)+len(attrs) <= 1 {
		return false // special case: slog.Info("msg", "key", "value") is ok.
	}

	l := len(keys) + len(attrs) + 1
	args := make([]ast.Expr, 0, l)
	args = append(args, call)
	args = append(args, keys...)
	args = append(args, attrs...)

	lines := make(map[int]struct{}, l)
	for _, arg := range args {
		line := fset.Position(arg.Pos()).Line
		if _, ok := lines[line]; ok {
			return true
		}
		lines[line] = struct{}{}
	}

	return false
}

func hasSuffixes(fset *[]string, fn string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(fn, s) {
			*fset = append(*fset, fn)

			return true
		}
	}

	return false
}

func hasGeneratedComment(fset *[]string, fn string, file *ast.File) bool {
	if ast.IsGenerated(file) {
		*fset = append(*fset, fn)
	}
	return false
}
