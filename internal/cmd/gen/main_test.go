package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zz_generated.go is the contracts': run go generate ./... after a change.
func TestGenerated(t *testing.T) {
	src, err := generate("../../contracts")
	if err != nil {
		t.Fatal(err)
	}
	have, err := os.ReadFile("../../provider/zz_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, have) {
		t.Error("internal/provider/zz_generated.go is not the contracts': go generate ./...")
	}
	for _, want := range []string{
		`{Attr: "dependencies", Kind: kObjects`, `JSON: "depends_on"`, // a meta-argument's name
		`{Attr: "token", Kind: kString, Desc: "Its secret, at its creation only.", ReadOnly: true, Stable: true, Sensitive: true}`,
		`{Attr: "expires_in", Kind: kString`, `WriteOnly: true`,
		`{Attr: "namespace", Kind: kString, Desc: "The namespace it lives in.", Required: true, PathOnly: true}`,
		`{Attr: "claims", Kind: kStringMap`,
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("no %s", want)
		}
	}
}

// A schema the provider cannot type, or a resource its contract does not
// serve, is refused: no generic block.
func TestGenerate_refuses(t *testing.T) {
	for name, edit := range map[string][2]string{
		"an object of anything": {"        description: {type: string, description: What it is.}\n        metadata:",
			"        description: {type: object, description: Anything.}\n        metadata:"},
		"no such path": {"  /apis/authorization.infrapi.io/v1/tokens/{id}:", "  /apis/authorization.infrapi.io/v1/pats/{id}:"},
	} {
		dir := t.TempDir()
		for _, f := range []string{"system", "catalog", "rbac", "events"} {
			data, err := os.ReadFile(filepath.Join("../../contracts", f+"-v1.yaml")) //#nosec G304 -- the contracts
			if err != nil {
				t.Fatal(err)
			}
			if f == "rbac" {
				if !bytes.Contains(data, []byte(edit[0])) {
					t.Fatalf("%s: no %q in rbac-v1.yaml", name, edit[0])
				}
				data = bytes.Replace(data, []byte(edit[0]), []byte(edit[1]), 1)
			}
			if err := os.WriteFile(filepath.Join(dir, f+"-v1.yaml"), data, 0o600); err != nil { //#nosec G703 -- the test's temporary directory
				t.Fatal(err)
			}
		}
		if _, err := generate(dir); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
