/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crypto

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func Hash(data any) string {
	hash := sha1.New()

	var err error
	switch asserted := data.(type) {
	case string:
		_, err = hash.Write([]byte(asserted))
	case []byte:
		_, err = hash.Write(asserted)
	default:
		err = json.NewEncoder(hash).Encode(data)
	}

	if err != nil {
		// This is not something that should ever happen at runtime and is also not
		// something we can really gracefully handle, so crashing and restarting might
		// be a good way to signal the service owner that something is up.
		panic(fmt.Sprintf("Failed to hash: %v", err))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func ShortHash(data any) string {
	return Hash(data)[:20]
}
