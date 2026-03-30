package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := doMain(context.Background()); err != nil {
		panic(err)
	}
}

func doMain(ctx context.Context) error {

	if err := setHeader(ctx, "./apis", goHeader); err != nil {
		return err
	}

	if err := setHeader(ctx, "./pkg", goHeader); err != nil {
		return err
	}

	if err := setHeader(ctx, "./cluster", goHeader); err != nil {
		return err
	}

	return nil
}

func setHeader(ctx context.Context, rootPath string, header string) error {

	if err := filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			cn, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			pkgIdx := getIdx(cn[:])
			if pkgIdx < 0 {
				return nil
			}

			newFile := header + "\n" + string(cn[pkgIdx:])

			if err := os.WriteFile(path, []byte(newFile), info.Mode().Perm()); err != nil {
				return err
			}

			return nil
		}); err != nil {
		return err
	}
	return nil
}

func getIdx(src []byte) int {

	ret := bytes.Index(src, []byte("package "))
	if idx := bytes.Index(src, []byte("//go:build")); idx > 0 && idx < ret {
		ret = idx
	}

	if idx := bytes.Index(src, []byte("// +build")); idx > 0 && idx < ret {
		ret = idx
	}

	if idx := bytes.Index(src, []byte("// Code generated")); idx > 0 && idx < ret {
		ret = idx
	}

	return ret
}

const goHeader = `// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.
`

const tsHeader = `/**
 * Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
 *
 * This software is licensed under the Octelium Enterprise Source-Available License.
 * Commercial and production use is strictly prohibited without a valid
 * Commercial Agreement from Octelium Labs, LLC.
 *
 * See the LICENSE file in the repository root for full license text.
 */`
