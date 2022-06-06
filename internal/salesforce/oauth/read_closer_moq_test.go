// Copyright © 2022 Meroxa, Inc. and Miquido
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth

import "io"

func NewFailedReadCloserMock(reader io.Reader, err error) *ReadCloserMock {
	return &ReadCloserMock{
		reader: reader,
		err:    err,
	}
}

type ReadCloserMock struct {
	reader io.Reader
	err    error
}

func (r ReadCloserMock) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r ReadCloserMock) Close() error {
	return r.err
}
