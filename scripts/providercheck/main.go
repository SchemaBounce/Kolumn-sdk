//go:build ignore

// Command providercheck type-checks Go modules against the real core.Provider
// contract. It backs scripts/check-interface-compliance.sh and
// scripts/validate-schemas.sh and is run with:
//
//	go run scripts/providercheck/main.go interfaces|schemas [module-dir...]
//
// With no module directories it checks the module in the current directory.
//
// interfaces:
//   - core.Provider must still be exactly the 4-method contract documented in
//     core/provider.go and README.md (Configure, Schema, CallFunction, Close
//     with their current signatures).
//   - Every named type that claims to be a provider must implement
//     core.Provider. A type claims it when its method set has CallFunction
//     with core.Provider's exact signature (the unified dispatch entry point),
//     or has all four method names (a provider with a wrong signature).
//     Assignments such as `var _ core.Provider = (*T)(nil)` are already
//     enforced by the compiler; a package that does not type-check fails here.
//
// schemas:
//   - The struct core.Provider.Schema() returns must have the core
//     compatibility fields with their JSON names.
//   - Every composite literal of that struct in non-test code must set Name
//     and Version (helpers/validation.ValidateSchema rejects a schema without
//     them), and every string literal in its SupportedFunctions must be a
//     function the unified dispatcher accepts.
//
// It uses only the standard library: `go list -export` builds the packages
// and the type information comes from the compiler's export data.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const corePath = "github.com/schemabounce/kolumn/sdk/core"

// providerContract is core.Provider as documented in core/provider.go and
// README.md. Changing the interface breaks every provider, so a change here
// must be deliberate.
var providerContract = map[string]string{
	"Configure":    "(context.Context, map[string]interface{}) error",
	"Schema":       "() (*core.Schema, error)",
	"CallFunction": "(context.Context, string, []byte) ([]byte, error)",
	"Close":        "() error",
}

// schemaFields are the core compatibility fields of the provider schema and
// their JSON names.
var schemaFields = map[string]string{
	"SupportedFunctions": "supported_functions",
	"ResourceTypes":      "resource_types",
	"ConfigSchema":       "config_schema",
}

// dispatchFunctions are the function names core.UnifiedDispatcher.Dispatch
// accepts (core/provider.go).
var dispatchFunctions = map[string]bool{
	"CreateResource":    true,
	"ReadResource":      true,
	"UpdateResource":    true,
	"DeleteResource":    true,
	"DiscoverResources": true,
	"DiscoverDatabase":  true,
	"Ping":              true,
}

type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Export     string
	DepOnly    bool
	Module     *struct{ Main bool }
}

type checker struct {
	fset     *token.FileSet
	imp      types.ImporterFrom
	provider *types.Interface
	problems []string
}

func main() {
	if len(os.Args) < 2 || (os.Args[1] != "interfaces" && os.Args[1] != "schemas") {
		fmt.Fprintln(os.Stderr, "usage: providercheck interfaces|schemas [module-dir...]")
		os.Exit(2)
	}
	mode := os.Args[1]
	dirs := os.Args[2:]
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	start, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	failed := false
	for _, dir := range dirs {
		if err := os.Chdir(filepath.Join(start, dir)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Printf("module %s\n", dir)
		problems, err := checkModule(mode, dir)
		if err != nil {
			fmt.Printf("  ERROR %v\n", err)
			failed = true
			continue
		}
		for _, p := range problems {
			fmt.Printf("  FAIL %s\n", p)
		}
		if len(problems) > 0 {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func checkModule(mode, dir string) ([]string, error) {
	pkgs, exports, err := listPackages()
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	lookup := func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(file)
	}
	c := &checker{fset: fset, imp: importer.ForCompiler(fset, "gc", lookup).(types.ImporterFrom)}

	core, err := c.imp.ImportFrom(corePath, ".", 0)
	if err != nil {
		return nil, fmt.Errorf("type-checking %s: %w", corePath, err)
	}
	obj := core.Scope().Lookup("Provider")
	if obj == nil {
		return nil, fmt.Errorf("%s.Provider not found", corePath)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%s.Provider is not an interface", corePath)
	}
	c.provider = iface

	switch mode {
	case "interfaces":
		c.checkContract()
		for _, p := range pkgs {
			if err := c.checkImplementations(p); err != nil {
				return nil, err
			}
		}
		fmt.Printf("  checked %d package(s) for core.Provider implementations\n", len(pkgs))
	case "schemas":
		schema := c.checkSchemaType()
		if schema == nil {
			break
		}
		literals := 0
		for _, p := range pkgs {
			n, err := c.checkSchemaLiterals(p, schema)
			if err != nil {
				return nil, err
			}
			literals += n
		}
		fmt.Printf("  checked %d %s literal(s) in %d package(s)\n", literals, schema.Obj().Name(), len(pkgs))
	}
	return c.problems, nil
}

// listPackages returns the module's own packages and the export data file of
// every package they depend on, plus core itself (a module that does not
// import core is still checked against it). `go list -export` compiles the
// packages, so a package that does not build fails here.
func listPackages() ([]listedPackage, map[string]string, error) {
	cmd := exec.Command("go", "list", "-e=false", "-export", "-deps",
		"-json=ImportPath,Dir,GoFiles,Export,DepOnly,Module", "./...", corePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	var pkgs []listedPackage
	exports := map[string]string{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			return nil, nil, fmt.Errorf("decoding go list output: %w", err)
		}
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
		if !p.DepOnly && p.Module != nil && p.Module.Main && len(p.GoFiles) > 0 {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, exports, nil
}

func (c *checker) report(pos token.Pos, format string, args ...any) {
	where := ""
	if pos.IsValid() {
		p := c.fset.Position(pos)
		if rel, err := filepath.Rel(mustGetwd(), p.Filename); err == nil {
			p.Filename = rel
		}
		where = fmt.Sprintf("%s:%d: ", p.Filename, p.Line)
	}
	c.problems = append(c.problems, where+fmt.Sprintf(format, args...))
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

// signature renders a method signature without parameter names, qualifying
// types by package name, e.g. "(context.Context, string) error".
func signature(sig *types.Signature) string {
	q := func(p *types.Package) string { return p.Name() }
	tuple := func(t *types.Tuple) []string {
		var s []string
		for i := 0; i < t.Len(); i++ {
			s = append(s, strings.ReplaceAll(types.TypeString(t.At(i).Type(), q), "any", "interface{}"))
		}
		return s
	}
	out := "(" + strings.Join(tuple(sig.Params()), ", ") + ")"
	switch res := tuple(sig.Results()); len(res) {
	case 0:
	case 1:
		out += " " + res[0]
	default:
		out += " (" + strings.Join(res, ", ") + ")"
	}
	return out
}

// checkContract verifies core.Provider is still the documented contract.
func (c *checker) checkContract() {
	got := map[string]string{}
	for i := 0; i < c.provider.NumMethods(); i++ {
		m := c.provider.Method(i)
		got[m.Name()] = signature(m.Type().(*types.Signature))
	}
	for name, want := range providerContract {
		switch sig, ok := got[name]; {
		case !ok:
			c.report(token.NoPos, "core.Provider no longer has %s%s", name, want)
		case sig != want:
			c.report(token.NoPos, "core.Provider.%s is %s%s, contract is %s%s", name, name, sig, name, want)
		}
	}
	for name, sig := range got {
		if _, ok := providerContract[name]; !ok {
			c.report(token.NoPos, "core.Provider gained %s%s; the contract is exactly 4 methods", name, sig)
		}
	}
	if len(c.problems) == 0 {
		fmt.Println("  core.Provider matches the 4-method contract")
	}
}

// claimsProvider reports whether a type presents itself as a core.Provider.
func (c *checker) claimsProvider(mset *types.MethodSet) bool {
	call := c.provider.Method(indexOf(c.provider, "CallFunction"))
	if sel := mset.Lookup(nil, "CallFunction"); sel != nil {
		if types.Identical(sel.Type(), call.Type()) {
			return true
		}
	}
	for i := 0; i < c.provider.NumMethods(); i++ {
		if mset.Lookup(nil, c.provider.Method(i).Name()) == nil {
			return false
		}
	}
	return true
}

func indexOf(iface *types.Interface, name string) int {
	for i := 0; i < iface.NumMethods(); i++ {
		if iface.Method(i).Name() == name {
			return i
		}
	}
	panic("core.Provider has no method " + name)
}

func (c *checker) checkImplementations(p listedPackage) error {
	pkg, err := c.imp.ImportFrom(p.ImportPath, p.Dir, 0)
	if err != nil {
		return fmt.Errorf("type-checking %s: %w", p.ImportPath, err)
	}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || tn.IsAlias() {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, isIface := named.Underlying().(*types.Interface); isIface {
			continue
		}
		ptr := types.NewPointer(named)
		if !c.claimsProvider(types.NewMethodSet(ptr)) {
			continue
		}
		if types.Implements(ptr, c.provider) {
			fmt.Printf("  ok   %s.%s implements core.Provider\n", p.ImportPath, name)
			continue
		}
		m, wrongType := types.MissingMethod(ptr, c.provider, true)
		want := signature(m.Type().(*types.Signature))
		if wrongType {
			c.report(tn.Pos(), "%s claims to be a core.Provider but %s has the wrong signature (want %s%s)", name, m.Name(), m.Name(), want)
		} else {
			c.report(tn.Pos(), "%s claims to be a core.Provider but is missing %s%s", name, m.Name(), want)
		}
	}
	return nil
}

// checkSchemaType returns the struct core.Provider.Schema() returns, after
// checking it still carries the core compatibility fields.
func (c *checker) checkSchemaType() *types.Named {
	m := c.provider.Method(indexOf(c.provider, "Schema"))
	res := m.Type().(*types.Signature).Results()
	if res.Len() == 0 {
		c.report(token.NoPos, "core.Provider.Schema returns nothing")
		return nil
	}
	t := res.At(0).Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		c.report(token.NoPos, "core.Provider.Schema returns %s, not a named struct", t)
		return nil
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		c.report(token.NoPos, "core.Provider.Schema returns %s, which is not a struct", named)
		return nil
	}
	fields := map[string]string{}
	for i := 0; i < st.NumFields(); i++ {
		tag := reflect.StructTag(st.Tag(i)).Get("json")
		fields[st.Field(i).Name()] = strings.Split(tag, ",")[0]
	}
	for _, name := range sortedKeys(schemaFields) {
		switch jsonName, ok := fields[name]; {
		case !ok:
			c.report(named.Obj().Pos(), "%s has no %s field", named.Obj().Name(), name)
		case jsonName != schemaFields[name]:
			c.report(named.Obj().Pos(), "%s.%s has JSON name %q, core expects %q", named.Obj().Name(), name, jsonName, schemaFields[name])
		}
	}
	if len(c.problems) == 0 {
		fmt.Printf("  %s has the core compatibility fields\n", types.TypeString(named, nil))
	}
	return named
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkSchemaLiterals checks every composite literal of the schema type in
// one package and returns how many it found.
func (c *checker) checkSchemaLiterals(p listedPackage, schema *types.Named) (int, error) {
	var files []*ast.File
	for _, name := range p.GoFiles {
		f, err := parser.ParseFile(c.fset, filepath.Join(p.Dir, name), nil, 0)
		if err != nil {
			return 0, err
		}
		files = append(files, f)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
	conf := types.Config{Importer: c.imp}
	if _, err := conf.Check(p.ImportPath, c.fset, files, info); err != nil {
		return 0, fmt.Errorf("type-checking %s: %w", p.ImportPath, err)
	}

	isSchema := func(t types.Type) bool {
		n, ok := t.(*types.Named)
		// Compare by path and name: the core package checked here and the
		// one the importer loaded are distinct objects.
		return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == schema.Obj().Pkg().Path() &&
			n.Obj().Name() == schema.Obj().Name()
	}

	count := 0
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isSchema(info.TypeOf(lit)) {
				return true
			}
			count++
			set := map[string]ast.Expr{}
			for _, elt := range lit.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						set[id.Name] = kv.Value
					}
				}
			}
			for _, required := range []string{"Name", "Version"} {
				if _, ok := set[required]; !ok {
					c.report(lit.Pos(), "%s literal does not set %s", schema.Obj().Name(), required)
				}
			}
			if fns, ok := set["SupportedFunctions"].(*ast.CompositeLit); ok {
				for _, e := range fns.Elts {
					tv := info.Types[e]
					if tv.Value == nil || tv.Value.Kind() != constant.String {
						continue
					}
					if fn := constant.StringVal(tv.Value); !dispatchFunctions[fn] {
						c.report(e.Pos(), "SupportedFunctions lists %q, which the unified dispatcher does not accept", fn)
					}
				}
			}
			return true
		})
	}
	return count, nil
}
