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
//     core.Provider, and every missing or mismatched method is reported. A
//     type claims it when its method set has a CallFunction method (any
//     signature: the unified dispatch entry point), at least three of the
//     four method names under any signatures, or two of them with its
//     Configure or Schema matching core.Provider's exact signature. (Close()
//     error alone is too common to count.)
//   - Exception: a type the code explicitly declares as a schema-testing
//     provider with `var _ sdktesting.SchemaProvider = (*T)(nil)`. That
//     interface's Schema() returns *sdktesting.ProviderSchema, so no type can
//     implement both it and core.Provider. Such types are listed as skipped;
//     a type that only happens to match SchemaProvider is still checked.
//   - Assignments such as `var _ core.Provider = (*T)(nil)` are already
//     enforced by the compiler; a package that does not type-check fails here.
//
// schemas:
//   - The struct core.Provider.Schema() returns must have the core
//     compatibility fields with their JSON names.
//   - A keyed literal of that struct that is returned directly
//     (`return &core.Schema{...}`) must set Name and Version
//     (helpers/validation.ValidateSchema rejects a schema without them).
//     Literals stored in a variable are not checked, since fields can be
//     set after construction, and neither is a return that also returns a
//     non-nil error (`return &core.Schema{}, err`).
//   - Every function name given to a SupportedFunctions field must be one
//     core.UnifiedDispatcher.Dispatch accepts (read from its allowedFunctions
//     map in the core sources). Names are followed through string and
//     slice literals, constants, append calls, and local or package string
//     and slice variables in the same package; values produced by other
//     function calls are not followed.
//
// Only a package's non-test files are checked: _test.go files are neither
// type-checked nor searched.
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

const testingPath = "github.com/schemabounce/kolumn/sdk/testing"

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
	all      map[string]listedPackage // every listed package, by import path
	lookup   importer.Lookup
	imp      types.ImporterFrom // shared importer for the schema checks
	provider *types.Interface   // core.Provider as loaded by imp
	problems []string
}

// newImporter returns an importer that reads compiler export data.
//
// Export data records every command package under the path "main", so one
// importer would merge the scopes of all main packages it loads. Each main
// package therefore gets an importer of its own.
func (c *checker) newImporter() types.ImporterFrom {
	return importer.ForCompiler(c.fset, "gc", c.lookup).(types.ImporterFrom)
}

// loadProvider returns core.Provider as seen by imp. Types from different
// importers are never identical, so a package is always compared with the
// core.Provider its own importer loaded.
func loadProvider(imp types.ImporterFrom) (*types.Interface, error) {
	return loadInterface(imp, corePath, "Provider")
}

func loadInterface(imp types.ImporterFrom, path, name string) (*types.Interface, error) {
	pkg, err := imp.ImportFrom(path, ".", 0)
	if err != nil {
		return nil, fmt.Errorf("type-checking %s: %w", path, err)
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("%s.%s not found", path, name)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%s.%s is not an interface", path, name)
	}
	return iface, nil
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
	pkgs, all, exports, err := listPackages()
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
	c := &checker{fset: fset, all: all, lookup: lookup}
	c.imp = c.newImporter()
	if c.provider, err = loadProvider(c.imp); err != nil {
		return nil, err
	}

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
		dispatch, err := c.dispatchFunctions()
		if err != nil {
			return nil, err
		}
		literals := 0
		for _, p := range pkgs {
			n, err := c.checkSchemaLiterals(p, schema, dispatch)
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
func listPackages() ([]listedPackage, map[string]listedPackage, map[string]string, error) {
	cmd := exec.Command("go", "list", "-e=false", "-export", "-deps",
		"-json=ImportPath,Dir,GoFiles,Export,DepOnly,Module", "./...", corePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	var pkgs []listedPackage
	all := map[string]listedPackage{}
	exports := map[string]string{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			return nil, nil, nil, fmt.Errorf("decoding go list output: %w", err)
		}
		all[p.ImportPath] = p
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
		if !p.DepOnly && p.Module != nil && p.Module.Main && len(p.GoFiles) > 0 {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, all, exports, nil
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

// claimsProvider reports whether a type presents itself as a core.Provider:
// it has a CallFunction method under any signature, at least three of the
// interface's method names under any signatures, or two of them with
// Configure or Schema matching core.Provider exactly.
func claimsProvider(mset *types.MethodSet, provider *types.Interface) bool {
	if mset.Lookup(nil, "CallFunction") != nil {
		return true
	}
	found, exact := 0, false
	for i := 0; i < provider.NumMethods(); i++ {
		m := provider.Method(i)
		sel := mset.Lookup(nil, m.Name())
		if sel == nil {
			continue
		}
		found++
		if (m.Name() == "Configure" || m.Name() == "Schema") && types.Identical(sel.Type(), m.Type()) {
			exact = true
		}
	}
	return found >= 3 || (found >= 2 && exact)
}

// declaredSchemaProviders returns the names of types a package explicitly
// declares as sdktesting.SchemaProvider: var _ sdktesting.SchemaProvider = (*T)(nil).
func declaredSchemaProviders(p listedPackage) (map[string]bool, error) {
	declared := map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range p.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(p.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		alias := ""
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == testingPath {
				alias = "testing"
				if imp.Name != nil {
					alias = imp.Name.Name
				}
			}
		}
		if alias == "" {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "_" || len(vs.Values) != 1 {
				return true
			}
			sel, ok := vs.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "SchemaProvider" {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); !ok || x.Name != alias {
				return true
			}
			// (*T)(nil)
			call, ok := vs.Values[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			paren, ok := call.Fun.(*ast.ParenExpr)
			if !ok {
				return true
			}
			if star, ok := paren.X.(*ast.StarExpr); ok {
				if id, ok := star.X.(*ast.Ident); ok {
					declared[id.Name] = true
				}
			}
			return true
		})
	}
	return declared, nil
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
	imp := c.newImporter()
	provider, err := loadProvider(imp)
	if err != nil {
		return err
	}
	declared, err := declaredSchemaProviders(p)
	if err != nil {
		return err
	}
	pkg, err := imp.ImportFrom(p.ImportPath, p.Dir, 0)
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
		mset := types.NewMethodSet(ptr)
		if !claimsProvider(mset, provider) {
			continue
		}
		if types.Implements(ptr, provider) {
			fmt.Printf("  ok   %s.%s implements core.Provider\n", p.ImportPath, name)
			continue
		}
		if declared[name] {
			fmt.Printf("  skip %s.%s is declared an sdktesting.SchemaProvider, which cannot also be a core.Provider\n", p.ImportPath, name)
			continue
		}
		for i := 0; i < provider.NumMethods(); i++ {
			m := provider.Method(i)
			want := signature(m.Type().(*types.Signature))
			switch sel := mset.Lookup(nil, m.Name()); {
			case sel == nil:
				c.report(tn.Pos(), "%s claims to be a core.Provider but is missing %s%s", name, m.Name(), want)
			case !types.Identical(sel.Type(), m.Type()):
				c.report(tn.Pos(), "%s claims to be a core.Provider but %s has the wrong signature (want %s%s)", name, m.Name(), m.Name(), want)
			}
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

// dispatchFunctions reads the function names core.UnifiedDispatcher.Dispatch
// accepts from the allowedFunctions map literal in the core sources, so the
// checker never keeps a second copy of that list.
func (c *checker) dispatchFunctions() (map[string]bool, error) {
	core, ok := c.all[corePath]
	if !ok {
		return nil, fmt.Errorf("%s not listed", corePath)
	}
	fset := token.NewFileSet()
	for _, name := range core.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(core.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Dispatch" || fn.Recv == nil || fn.Body == nil {
				continue
			}
			var names map[string]bool
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				as, ok := n.(*ast.AssignStmt)
				if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
					return true
				}
				if id, ok := as.Lhs[0].(*ast.Ident); !ok || id.Name != "allowedFunctions" {
					return true
				}
				lit, ok := as.Rhs[0].(*ast.CompositeLit)
				if !ok {
					return true
				}
				names = map[string]bool{}
				for _, elt := range lit.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if bl, ok := kv.Key.(*ast.BasicLit); ok && bl.Kind == token.STRING {
							names[strings.Trim(bl.Value, "`\"")] = true
						}
					}
				}
				return false
			})
			if len(names) > 0 {
				return names, nil
			}
		}
	}
	return nil, fmt.Errorf("allowedFunctions map not found in %s Dispatch", corePath)
}

// checkSchemaLiterals checks the schema literals and SupportedFunctions
// values in one package and returns how many schema literals it found.
func (c *checker) checkSchemaLiterals(p listedPackage, schema *types.Named, dispatch map[string]bool) (int, error) {
	var files []*ast.File
	for _, name := range p.GoFiles {
		f, err := parser.ParseFile(c.fset, filepath.Join(p.Dir, name), nil, 0)
		if err != nil {
			return 0, err
		}
		files = append(files, f)
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: c.imp}
	if _, err := conf.Check(p.ImportPath, c.fset, files, info); err != nil {
		return 0, fmt.Errorf("type-checking %s: %w", p.ImportPath, err)
	}

	isSchema := func(t types.Type) bool {
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		n, ok := t.(*types.Named)
		// Compare by path and name: the core package checked here and the
		// one the importer loaded are distinct objects.
		return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == schema.Obj().Pkg().Path() &&
			n.Obj().Name() == schema.Obj().Name()
	}

	// Every value assigned to each variable in the package, so function
	// names can be followed through variables.
	values := map[types.Object][]ast.Expr{}
	objOf := func(id *ast.Ident) types.Object {
		if obj := info.Defs[id]; obj != nil {
			return obj
		}
		return info.Uses[id]
	}
	// Values given to a SupportedFunctions field.
	var supported []ast.Expr
	count := 0

	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if len(n.Lhs) == len(n.Rhs) {
					for i, lhs := range n.Lhs {
						switch l := lhs.(type) {
						case *ast.Ident:
							if obj := objOf(l); obj != nil {
								values[obj] = append(values[obj], n.Rhs[i])
							}
						case *ast.SelectorExpr:
							if l.Sel.Name == "SupportedFunctions" && isSchema(info.TypeOf(l.X)) {
								supported = append(supported, n.Rhs[i])
							}
						}
					}
				}
			case *ast.ValueSpec:
				if len(n.Names) == len(n.Values) {
					for i, id := range n.Names {
						if obj := info.Defs[id]; obj != nil {
							values[obj] = append(values[obj], n.Values[i])
						}
					}
				}
			case *ast.ReturnStmt:
				// An error path (`return &core.Schema{}, err`) need not
				// return a complete schema.
				if returnsError(info, n) {
					return true
				}
				for _, r := range n.Results {
					if u, ok := r.(*ast.UnaryExpr); ok && u.Op == token.AND {
						r = u.X
					}
					lit, ok := r.(*ast.CompositeLit)
					if !ok || !isSchema(info.TypeOf(lit)) {
						continue
					}
					set, keyed := keyedFields(lit)
					if !keyed {
						continue
					}
					for _, required := range []string{"Name", "Version"} {
						if _, ok := set[required]; !ok {
							c.report(lit.Pos(), "returned %s literal does not set %s", schema.Obj().Name(), required)
						}
					}
				}
			case *ast.CompositeLit:
				if !isSchema(info.TypeOf(n)) {
					return true
				}
				count++
				if set, _ := keyedFields(n); set["SupportedFunctions"] != nil {
					supported = append(supported, set["SupportedFunctions"])
				}
			}
			return true
		})
	}

	seen := map[ast.Expr]bool{}
	var names func(e ast.Expr)
	names = func(e ast.Expr) {
		if e == nil || seen[e] {
			return
		}
		seen[e] = true
		check := func(e ast.Expr) {
			if tv := info.Types[e]; tv.Value != nil && tv.Value.Kind() == constant.String {
				if fn := constant.StringVal(tv.Value); !dispatch[fn] {
					c.report(e.Pos(), "SupportedFunctions lists %q, which the unified dispatcher does not accept", fn)
				}
				return
			}
			names(e)
		}
		switch e := e.(type) {
		case *ast.ParenExpr:
			names(e.X)
		case *ast.CompositeLit:
			for _, elt := range e.Elts {
				check(elt)
			}
		case *ast.CallExpr:
			// append(base, names...): follow the base and check the names.
			if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "append" && len(e.Args) > 0 {
				names(e.Args[0])
				for _, a := range e.Args[1:] {
					check(a)
				}
			}
		case *ast.Ident:
			// A variable: check each value assigned to it, so a string
			// variable is checked like a literal and a slice variable is
			// followed.
			if obj := info.Uses[e]; obj != nil {
				for _, v := range values[obj] {
					check(v)
				}
			}
		}
	}
	for _, e := range supported {
		names(e)
	}
	return count, nil
}

// returnsError reports whether a return statement returns a non-nil error
// value next to its other results.
func returnsError(info *types.Info, ret *ast.ReturnStmt) bool {
	errType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	for _, r := range ret.Results {
		tv, ok := info.Types[r]
		if !ok || tv.IsNil() || tv.Type == nil {
			continue
		}
		if types.Implements(tv.Type, errType) {
			return true
		}
	}
	return false
}

// keyedFields returns a literal's keyed fields, and false when the literal is
// positional (it then sets every field).
func keyedFields(lit *ast.CompositeLit) (map[string]ast.Expr, bool) {
	set := map[string]ast.Expr{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, false
		}
		if id, ok := kv.Key.(*ast.Ident); ok {
			set[id.Name] = kv.Value
		}
	}
	return set, true
}
