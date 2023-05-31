/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package radixtree

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"sync"

	v1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"

	"github.com/armon/go-radix"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"
)

// RadixTree is used to match prefix/suffix for ip/host in x509 certs stored in secret.
type RadixTree interface {
	Insert(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (processed bool, err error)
	Update(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (processed bool, err error)
	Delete(obj registry.ResourceKey) (processed bool, err error)
	// If multiple certs specified identity ip/host as their trusted origin, tree cannot ensure which cert will be used
	// TODO:: maybe introduce other options to choose cert
	MatchLongestPrefix(path string) (string, []*v1beta1.TLSCert, bool)
	MatchLongestPrefixWithRandomCert(path string) (string, *v1beta1.TLSCert, bool)
}

// SuffixRadixTree implements RadixTree interface, with suffix match supported
// "*.a.com" will be reversed as "moc.a.*", stored as "moc.a." in tree,
// to match the string with suffix ".a.com".
// All method accept original path, reserved string is NOT NEED
type SuffixRadixTree struct {
	sync.RWMutex
	// secret => tlsCert
	secretMap map[registry.ResourceKey]*v1beta1.TLSCert
	// SNI => []secrets
	pathDocumentMap map[string]map[registry.ResourceKey]struct{}
	// SNI match tree
	radixTree *radix.Tree

	logger *zap.SugaredLogger
}

// NewSuffixRadixTree returns a RadixTree interface
func NewSuffixRadixTree() RadixTree {
	return &SuffixRadixTree{
		secretMap:       make(map[registry.ResourceKey]*v1beta1.TLSCert),
		pathDocumentMap: make(map[string]map[registry.ResourceKey]struct{}),
		radixTree:       radix.New(),
		logger:          logging.GetLogger().Named("suffix-radixtree"),
	}
}

func (tree *SuffixRadixTree) reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func (tree *SuffixRadixTree) getSNIsFromCert(tlsCert *v1beta1.TLSCert) []string {
	snis := make([]string, 0)
	pemdata := []byte(tlsCert.Cert)
	for {
		pb, rest := pem.Decode(pemdata)
		if pb == nil {
			break
		}
		pemdata = rest
		certs, err := x509.ParseCertificates(pb.Bytes)
		if err != nil {
			tree.logger.Error(err, "Parse Certificate from pem formatted cert failed")
			continue
		}
		for _, cert := range certs {
			// build domain
			domains := make(map[string]struct{})
			for _, dns := range cert.DNSNames {
				domains[dns] = struct{}{}
			}
			for _, uri := range cert.URIs {
				domains[uri.Host] = struct{}{}
			}
			for _, ip := range cert.IPAddresses {
				domains[ip.String()] = struct{}{}
			}
			domains[cert.Subject.CommonName] = struct{}{}

			// check domains
			for domain := range domains {
				if utils.HostMatch(domain) {
					snis = append(snis, domain)
				} else {
					tree.logger.Infow("SSL's domain does not match required pattern", "domain", domain)
				}
			}
		}
	}
	return append(snis, tlsCert.SNIs...)
}

// whether key file of cert pair is encrypted, which is not accepted
func (tree *SuffixRadixTree) isKeyEncrypted(key string) bool {
	pemdata := []byte(key)
	for {
		pb, rest := pem.Decode(pemdata)
		if pb == nil {
			break
		}
		pemdata = rest
		if _, ok := pb.Headers["DEK-Info"]; ok {
			return true
		}
	}
	return false
}

func (tree *SuffixRadixTree) diffSNIs(old, new []string) (deleted, added []string) {
	currSNIMap := make(map[string]struct{})
	deleted = make([]string, 0)
	added = make([]string, 0)
	for _, sni := range old {
		currSNIMap[sni] = struct{}{}
	}
	for _, sni := range new {
		if _, ok := currSNIMap[sni]; ok {
			delete(currSNIMap, sni)
			continue
		}
		added = append(added, sni)
	}
	for sni := range currSNIMap {
		deleted = append(deleted, sni)
	}
	return deleted, added
}

// "*.a.com" => "moc.a."
// "a.b.com" => "moc.b.a$"
func (tree *SuffixRadixTree) buildReverseMatchPath(s string) string {
	reverseS := tree.reverse(s)
	// add string finish label to different from wildcard sni
	if !strings.HasSuffix(reverseS, "*") {
		reverseS += "$"
	}
	reverseS = strings.TrimSuffix(reverseS, "*")
	return reverseS
}

func (tree *SuffixRadixTree) insertSNIs(obj registry.ResourceKey, snis []string) {
	for _, sni := range snis {
		reverseSNI := tree.buildReverseMatchPath(sni)
		if secrets, ok := tree.pathDocumentMap[sni]; ok {
			secrets[obj] = struct{}{}
			tree.pathDocumentMap[sni] = secrets
		} else {
			tree.pathDocumentMap[sni] = map[registry.ResourceKey]struct{}{
				obj: {},
			}
			tree.radixTree.Insert(reverseSNI, sni)
		}
	}
}

func (tree *SuffixRadixTree) removeSNIs(obj registry.ResourceKey, snis []string) {
	for _, sni := range snis {
		reverseSNI := tree.buildReverseMatchPath(sni)
		if secrets, ok := tree.pathDocumentMap[sni]; ok {
			delete(secrets, obj)
			tree.pathDocumentMap[sni] = secrets
			if len(secrets) == 0 {
				delete(tree.pathDocumentMap, sni)
				tree.radixTree.Delete(reverseSNI)
			}
		}
	}
}

func (tree *SuffixRadixTree) insert(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (bool, error) {
	if _, ok := tree.secretMap[obj]; ok {
		return tree.update(obj, tlsCert)
	}
	if tree.isKeyEncrypted(tlsCert.Key) {
		tree.logger.Infow("private key is encrypted, skip insert into cert list", "secret", obj)
		return false, nil
	}
	snis := tree.getSNIsFromCert(tlsCert)
	if len(snis) == 0 {
		tree.logger.Infow("No ip/host/url specified in cert, skip insert into cert list", "secret", obj)
		return false, nil
	}

	tree.insertSNIs(obj, snis)
	tree.secretMap[obj] = tlsCert
	return true, nil
}

// Insert insert the secret object descriptor and a cert pair assosiated with the secret
func (tree *SuffixRadixTree) Insert(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (bool, error) {
	tree.Lock()
	defer tree.Unlock()
	return tree.insert(obj, tlsCert)
}

func (tree *SuffixRadixTree) delete(obj registry.ResourceKey) (bool, error) {
	tlsCert, ok := tree.secretMap[obj]
	if !ok {
		return false, nil
	}
	defer delete(tree.secretMap, obj)

	snis := tree.getSNIsFromCert(tlsCert)
	if len(snis) == 0 {
		tree.logger.Infow("No ip/host/url specified in cert, skip deletion process for cert", "secret", obj)
		return false, nil
	}

	tree.removeSNIs(obj, snis)
	return true, nil
}

// Delete delete the secret object descriptor and the assosiated cert pairs from tree
func (tree *SuffixRadixTree) Delete(obj registry.ResourceKey) (bool, error) {
	tree.Lock()
	defer tree.Unlock()
	return tree.delete(obj)
}

func (tree *SuffixRadixTree) update(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (bool, error) {
	oldTLSCert, ok := tree.secretMap[obj]
	if !ok {
		return tree.insert(obj, tlsCert)
	}
	if tree.isKeyEncrypted(tlsCert.Key) {
		tree.logger.Infow("Certs private key is encrypted, skip insert into cert list", "secret", obj)
		tree.delete(obj)
		return false, nil
	}
	oldSNIs := tree.getSNIsFromCert(oldTLSCert)
	newSNIs := tree.getSNIsFromCert(tlsCert)
	if len(newSNIs) == 0 {
		tree.logger.Infow("No ip/host/url specified in new cert, deletion cert", "secret", obj)
		tree.delete(obj)
		return false, nil
	}
	deletedSNIs, addedSNIs := tree.diffSNIs(oldSNIs, newSNIs)

	tree.removeSNIs(obj, deletedSNIs)
	tree.insertSNIs(obj, addedSNIs)
	tree.secretMap[obj] = tlsCert
	return true, nil
}

// Update update the secret object descriptor and the assosiated cert pairs in tree
func (tree *SuffixRadixTree) Update(obj registry.ResourceKey, tlsCert *v1beta1.TLSCert) (bool, error) {
	tree.Lock()
	defer tree.Unlock()
	return tree.update(obj, tlsCert)
}

// MatchLongestPrefixWithRandomCert will match the longest prefix (suffix in host exactly) for provided host
// Return the matched suffix, cert info and flag indicating whether the host is matched
// Empty value and false flag will be returned when host is not matched
func (tree *SuffixRadixTree) MatchLongestPrefixWithRandomCert(path string) (string, *v1beta1.TLSCert, bool) {
	tree.RLock()
	defer tree.RUnlock()
	reversedPath := tree.buildReverseMatchPath(path)
	suffix, sni, matched := tree.radixTree.LongestPrefix(reversedPath)
	if !matched {
		return suffix, nil, false
	}
	secrets, ok := tree.pathDocumentMap[sni.(string)]
	if !ok || len(secrets) == 0 {
		tree.logger.Errorw(
			"matched sni but no secret found",

			eris.New("matched sni but no secret found"),
			"suffix",
			suffix,
			"sni",
			sni,
			"path",
			path,
		)
		return suffix, nil, false
	}
	for secret := range secrets {
		tlsCert, ok := tree.secretMap[secret]
		if !ok {
			tree.logger.Errorw(
				"SNI matched secret but no certs found",
				"err",
				eris.New("SNI matched secret but no certs found"),
				"suffix",
				suffix,
				"sni",
				sni,
				"path",
				path,
				"secret",
				secret,
			)
			continue
		}
		tree.logger.Debugw("cert secret matched", "path", path, "suffix", suffix, "certs", tlsCert, "secret", secret)
		return suffix, tlsCert, true
	}
	return suffix, nil, false
}

// MatchLongestPrefix will match the longest prefix (suffix in host exactly) for provided host
// Return the matched suffix, cert info and flag indicating whether the host is matched
// Empty value and false flag will be returned when host is not matched
func (tree *SuffixRadixTree) MatchLongestPrefix(path string) (string, []*v1beta1.TLSCert, bool) {
	tree.RLock()
	defer tree.RUnlock()
	reversedPath := tree.buildReverseMatchPath(path)
	suffix, sni, matched := tree.radixTree.LongestPrefix(reversedPath)
	if !matched {
		return suffix, nil, false
	}
	secrets, ok := tree.pathDocumentMap[sni.(string)]
	if !ok || len(secrets) == 0 {
		tree.logger.Errorw(
			"matched sni but no secret found",
			"err",
			eris.New("matched sni but no secret found"),
			"suffix",
			suffix,
			"sni",
			sni,
			"path",
			path,
		)
		return suffix, nil, false
	}
	retCerts := make([]*v1beta1.TLSCert, 0)
	for secret := range secrets {
		tlsCert, ok := tree.secretMap[secret]
		if !ok {
			tree.logger.Errorw(
				"SNI matched secret but no certs found",
				"err",
				eris.New("SNI matched secret but no certs found"),
				"suffix",
				suffix,
				"sni",
				sni,
				"path",
				path,
				"secret",
				secret,
			)
			continue
		}
		tree.logger.Debugw("cert secret matched", "path", path, "suffix", suffix, "certs", tlsCert, "secret", secret)
		retCerts = append(retCerts, tlsCert)
	}
	if len(retCerts) != 0 {
		return suffix, retCerts, true
	}
	return suffix, nil, false
}
