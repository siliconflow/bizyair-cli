package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var (
	formatVerb  = regexp.MustCompile(`%[-+#0-9.*\[\]]*[A-Za-z%]`)
	englishWord = regexp.MustCompile(`[A-Za-z]{2,}`)
)

// TestUserOutputDoesNotUseHardcodedNaturalLanguage protects the display
// entry points most likely to regress during routine feature work.
func TestUserOutputDoesNotUseHardcodedNaturalLanguage(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	protectedRoots := []string{"cmd", "config", "lib"}

	for _, protectedRoot := range protectedRoots {
		root := filepath.Join(repositoryRoot, protectedRoot)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			checkUserOutputFile(t, path)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLiteralMessageIDsExist(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	catalog := readCatalog(t, "locales/active.en.yaml")
	files := token.NewFileSet()

	err := filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			// Skip nested Go modules (e.g. the gitignored bizyair-cli_参照
			// reference copy). They keep their own go.mod and i18n catalogs
			// and are not part of this module, so their message IDs must not
			// be validated against this module's catalog.
			if path != repositoryRoot {
				if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(files, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			qualifier, ok := selector.X.(*ast.Ident)
			if !ok || qualifier.Name != "i18n" || (selector.Sel.Name != "T" && selector.Sel.Name != "TN" && selector.Sel.Name != "NewError") {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			messageID, unquoteErr := strconv.Unquote(literal.Value)
			if unquoteErr == nil {
				if _, exists := catalog[messageID]; !exists {
					position := files.Position(literal.Pos())
					t.Errorf("unknown message ID %q at %s:%d", messageID, path, position.Line)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkUserOutputFile(t *testing.T, path string) {
	t.Helper()
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	report := func(literal *ast.BasicLit) {
		value, err := strconv.Unquote(literal.Value)
		if err != nil || !containsNaturalLanguage(value) {
			return
		}
		position := files.Position(literal.Pos())
		t.Errorf("hard-coded user-facing text at %s:%d: %q; add a message ID to both catalogs", path, position.Line, value)
	}

	ast.Inspect(parsed, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.AssignStmt:
			for index, left := range current.Lhs {
				selector, ok := left.(*ast.SelectorExpr)
				if !ok || !protectedDisplayField(selector.Sel.Name) || index >= len(current.Rhs) {
					continue
				}
				if literal, ok := current.Rhs[index].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					report(literal)
				}
			}
		case *ast.KeyValueExpr:
			identifier, ok := current.Key.(*ast.Ident)
			if !ok || !protectedDisplayField(identifier.Name) {
				break
			}
			if literal, ok := current.Value.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				report(literal)
			}
		case *ast.CallExpr:
			selector, ok := current.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			qualifier := ""
			if identifier, ok := selector.X.(*ast.Ident); ok {
				qualifier = identifier.Name
			}
			start := -1
			switch selector.Sel.Name {
			case "Errorf":
				if qualifier == "fmt" {
					start = 0
				}
			case "New":
				if qualifier == "errors" {
					start = 0
				}
			case "NewValidationError", "WithStep", "WriteString", "Render", "SetString":
				start = 0
			case "WithHelp":
				start = 1 // the first string is the stable key sequence
			default:
				if qualifier == "fmt" && (strings.HasPrefix(selector.Sel.Name, "Print") || strings.HasPrefix(selector.Sel.Name, "Fprint")) {
					start = 0
				}
			}
			if start < 0 {
				break
			}
			for _, argument := range current.Args[start:] {
				if literal, ok := argument.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					report(literal)
				}
			}
		}
		return true
	})
}

func protectedDisplayField(name string) bool {
	switch strings.ToLower(name) {
	case "usage", "usagetext", "argsusage", "title", "placeholder", "description", "desc":
		return true
	default:
		return false
	}
}

func containsNaturalLanguage(value string) bool {
	withoutFormats := formatVerb.ReplaceAllString(value, "")
	for _, character := range withoutFormats {
		if character >= '\u3400' && character <= '\u9fff' {
			return true
		}
	}
	return englishWord.MatchString(withoutFormats)
}
