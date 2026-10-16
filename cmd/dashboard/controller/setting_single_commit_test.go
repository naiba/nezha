package controller

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

// The transition helper owns Save (including unchanged MCP settings). A direct
// second Save after it can fail after persistence and irreversible revocation.
func TestUpdateConfigHasSinglePersistenceOwner(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "setting.go", nil, 0)
	require.NoError(t, err)
	transitions, directSaves := 0, 0
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "updateConfig" {
			continue
		}
		found = true
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "applyEnableMCPTransition" {
				transitions++
			}
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Save" {
				directSaves++
			}
			return true
		})
	}
	require.True(t, found)
	require.Equal(t, 1, transitions)
	require.Zero(t, directSaves)
}

func TestUnchangedMCPTransitionStillSavesOnce(t *testing.T) {
	saves := 0
	err := applyEnableMCPTransition(true, true, func(bool) {}, func() error { saves++; return nil }, func() { t.Error("unchanged MCP must not revoke streams") })
	require.NoError(t, err)
	require.Equal(t, 1, saves)
}
