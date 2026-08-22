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

			newFile := addHeader(cn, header)
			if newFile == nil {
				return nil
			}

			if err := os.WriteFile(path, newFile, info.Mode().Perm()); err != nil {
				return err
			}

			return nil
		}); err != nil {
		return err
	}
	return nil
}

func addHeader(src []byte, header string) []byte {
	pkgIdx := getIdx(src)
	if pkgIdx < 0 {
		return nil
	}

	// Go build constraints must remain at the beginning of the file. Keep the
	// leading constraint block ahead of the license header when rewriting it.
	buildEnd := getLeadingBuildConstraintEnd(src)
	if buildEnd > 0 {
		pkgIdx = bytes.Index(src, []byte("package "))
		return []byte(string(src[:buildEnd]) + "\n" + header + "\n" + string(src[pkgIdx:]))
	}

	return []byte(header + "\n" + string(src[pkgIdx:]))
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

func getLeadingBuildConstraintEnd(src []byte) int {
	offset := 0
	for offset < len(src) {
		lineEnd := bytes.IndexByte(src[offset:], '\n')
		if lineEnd < 0 {
			lineEnd = len(src)
		} else {
			lineEnd += offset + 1
		}

		line := bytes.TrimSpace(src[offset:lineEnd])
		if !bytes.HasPrefix(line, []byte("//go:build ")) &&
			!bytes.HasPrefix(line, []byte("// +build ")) {
			break
		}

		offset = lineEnd
	}

	if offset > 0 {
		return offset
	}
	return 0
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
