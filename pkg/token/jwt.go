/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2025 Tencent. All rights reserved.
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

// Package token ...
package token

import (
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

// Issuer issuer for jwt token
type Issuer struct {
	sub      string
	secret   string
	name     string
	interval time.Duration

	token     string
	tokenTime time.Time
	mutex     sync.RWMutex
	secMutex  sync.Mutex
	stopCh    chan struct{}

	logger *zap.SugaredLogger
}

// New create new issuer
func New(sub, secret, name string) *Issuer {
	return &Issuer{
		sub:       sub,
		secret:    secret,
		name:      name,
		interval:  30 * time.Minute,
		stopCh:    make(chan struct{}, 1),
		token:     "",
		tokenTime: time.Now(),
		logger:    logging.GetLogger().Named("token"),
	}
}

// GetToken get token
func (i *Issuer) GetToken() string {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.token
}

// GetSecret get secret
func (i *Issuer) GetSecret() string {
	i.secMutex.Lock()
	defer i.secMutex.Unlock()
	return i.secret
}

// SetSecret set secret
func (i *Issuer) SetSecret(secret string) {
	i.secMutex.Lock()
	defer i.secMutex.Unlock()

	if i.secret == secret {
		return
	}
	i.secret = secret
	i.setToken()
}

// doSignJwtToken sign jwt token for status reporting
func (i *Issuer) doSignJwtToken(sub, name string, t time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"name": name,
		"iat":  t.Unix(),
		"exp":  t.Add(time.Hour).Unix(),
	})
	i.secMutex.Lock()
	tmpSec := i.secret
	i.secMutex.Unlock()
	tokenString, err := token.SignedString([]byte(tmpSec))
	if err != nil {
		return "", eris.Wrapf(err, "sign jwt token failed")
	}
	return tokenString, nil
}

func (i *Issuer) setToken() {
	timeNow := time.Now()
	if len(i.secret) == 0 {
		i.logger.Debug("empty jwt secret")
		i.mutex.Lock()
		i.token = ""
		i.tokenTime = timeNow
		i.mutex.Unlock()
		i.logger.Debug("clean jwt token successfully")
		return
	}
	tokenString, err := i.doSignJwtToken(i.sub, i.name, timeNow)
	if err != nil {
		i.logger.Error(err, "sign jwt token failed")
		return
	}
	i.mutex.Lock()
	i.token = tokenString
	i.tokenTime = timeNow
	i.mutex.Unlock()
	i.logger.Debug("generate new jwt token successfully")
}

func (i *Issuer) needRefresh() bool {
	timeNow := time.Now()
	return i.tokenTime.Add(i.interval).Before(timeNow)
}

// RefreshLoop loop to refresh token
func (i *Issuer) RefreshLoop() {
	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !i.needRefresh() {
				continue
			}
			i.setToken()
		case <-i.stopCh:
			i.logger.Info("jwt token refresh loop exit")
			return
		}
	}
}

// Stop stop refresh loop
func (i *Issuer) Stop() {
	i.stopCh <- struct{}{}
}
