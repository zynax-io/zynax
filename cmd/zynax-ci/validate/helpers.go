// SPDX-License-Identifier: Apache-2.0

package validate

func single(file, path, msg string) []ValidationError {
	return []ValidationError{{File: file, Path: path, Message: msg}}
}
