package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_TEST_MAIN") == "main" {
		main()
	} else {
		os.Exit(m.Run())
	}
}

func TestTestData_1(t *testing.T) {
	testCases := []struct {
		description      string
		depGopkgLockPath string
		goListModAllPath string
		wantOutputPath   string
	}{
		{"simple example from stellar/go", "testdata/1/Gopkg.lock", "testdata/1/go.list", "testdata/1/output.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			wantOut, err := ioutil.ReadFile(tc.wantOutputPath)
			if err != nil {
				t.Fatalf("error loading testdata: %v", err)
			}

			cmd := exec.Command(os.Args[0], "-d", tc.depGopkgLockPath, "-m", tc.goListModAllPath)
			cmd.Env = []string{"GO_TEST_MAIN=main"}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("error executing self: %v", err)
			}

			if diff := cmp.Diff(out, wantOut); diff != "" {
				t.Error(diff)
			}
		})
	}
}
